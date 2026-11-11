package managers

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"od/internal/config"
	"od/internal/httpclient"
	"od/internal/models"
	"od/internal/postgres"
	"od/internal/schedule"
)

// AutomationsManager runs the enabled automations. A graph is evaluated when one of its timers
// (cron, sunrise or sunset) fires, when a tag of one of its tagValueFloat/tagValueBoolean triggers
// gets a value, and when a waitCheck delay ends. An action runs when its input becomes true (not
// again while it stays true), and on every timer or waitCheck firing that makes it true.
// A waitCheck starts its delay when its input becomes true (or on a timer firing that makes it
// true); a new rise while the delay runs does not start another one.
type AutomationsManager struct {
	cfg     *config.Config
	db      *postgres.DB
	tags    *TagsManager
	devices *DevicesManager
	http    *httpclient.Client

	mutex   sync.Mutex // timers and tag values evaluate from their own goroutines
	running []*runningAutomation
}

type runningAutomation struct {
	models.Automation
	jobs    []*schedule.Job
	active  map[string]bool        // action outputs, and waitCheck inputs, of the last evaluation
	waits   map[string]*time.Timer // waitCheck delays running
	stopped bool
}

func NewAutomationsManager(cfg *config.Config, db *postgres.DB, tags *TagsManager, devices *DevicesManager, http *httpclient.Client) *AutomationsManager {
	return &AutomationsManager{cfg: cfg, db: db, tags: tags, devices: devices, http: http}
}

// Start loads the enabled automations and schedules their timers. An automation that fails to
// start is logged and skipped, so one bad row does not stop the others.
func (manager *AutomationsManager) Start(ctx context.Context) error {
	automations, err := manager.ReadAllFromDB(ctx)
	if err != nil {
		return err
	}

	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for _, automation := range automations {
		if !automation.Enabled {
			continue
		}
		if err := validateGraph(automation.Graph); err != nil {
			slog.Error("automation skipped", "id", automation.ID, "name", automation.Name, "err", err)
			continue
		}
		running := &runningAutomation{Automation: automation, active: map[string]bool{}, waits: map[string]*time.Timer{}}
		for _, node := range automation.Graph.Nodes {
			timerID := node.ID
			fire := func() { manager.evaluate(running, timerID, "", "") }
			switch node.Type {
			case models.NodeTimer:
				job, err := schedule.New(node.Cron, fire)
				if err != nil {
					slog.Error("automation timer skipped", "id", automation.ID, "node", timerID, "err", err)
					continue
				}
				running.jobs = append(running.jobs, job)
			case models.NodeSunrise, models.NodeSunset:
				if manager.cfg.Latitude == 0 && manager.cfg.Longitude == 0 {
					slog.Warn("automation sun trigger skipped, LATITUDE and LONGITUDE are not set", "id", automation.ID, "node", timerID)
					continue
				}
				lat, lon := manager.cfg.Latitude, manager.cfg.Longitude
				rise, offset := node.Type == models.NodeSunrise, node.OffsetMinutes
				next := func(after time.Time) time.Time { return nextSunEvent(lat, lon, rise, offset, after) }
				slog.Info("automation sun trigger", "automation", automation.Name, "node", timerID, "next", next(time.Now()).Format("2006-01-02 15:04"))
				running.jobs = append(running.jobs, schedule.NewFunc(next, fire))
			}
		}
		manager.running = append(manager.running, running)
	}
	slog.Info("automations started", "count", len(manager.running))
	return nil
}

func (manager *AutomationsManager) Stop() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for _, running := range manager.running {
		for _, job := range running.jobs {
			job.Cancel()
		}
		for _, wait := range running.waits {
			wait.Stop()
		}
		running.stopped = true
	}
	manager.running = nil
}

// Reload restarts the automations after a change in the database
func (manager *AutomationsManager) Reload(ctx context.Context) error {
	manager.Stop()
	return manager.Start(ctx)
}

// TagValueChanged evaluates the automations that watch the tag (TagsManager calls it)
func (manager *AutomationsManager) TagValueChanged(tagID int) {
	manager.mutex.Lock()
	watching := []*runningAutomation{}
	for _, running := range manager.running {
		for _, node := range running.Graph.Nodes {
			if (node.Type == models.NodeTagValueFloat || node.Type == models.NodeTagValueBool) && node.TagID == tagID {
				watching = append(watching, running)
				break
			}
		}
	}
	manager.mutex.Unlock()

	for _, running := range watching {
		manager.evaluate(running, "", "", manager.tagName(tagID)+" changed")
	}
}

// newRunID is a short temporary id that ties together the logs of one automation run
func newRunID() string {
	return fmt.Sprintf("%06x", rand.Uint32()&0xffffff)
}

// evaluate runs the graph once and logs every decision with the values behind it. run is the id
// shown in its logs: empty starts a new run, a waitCheck firing passes the id of the run that
// started the wait so the chain keeps one id. cause says why a tag change evaluates it; a timer,
// sun or waitCheck firing is named by firedTimer.
func (manager *AutomationsManager) evaluate(running *runningAutomation, firedTimer string, run string, cause string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if running.stopped {
		return
	}
	if run == "" {
		run = newRunID()
	}
	log := slog.With("run", run)

	// each tag is read once, so the logged value is the one the decision used
	values := map[int]models.TagValue{}
	read := func(tagID int) models.TagValue {
		if value, found := values[tagID]; found {
			return value
		}
		value := manager.lastValue(tagID)
		values[tagID] = value
		return value
	}
	names := map[string]string{}
	for _, node := range running.Graph.Nodes {
		names[node.ID] = manager.nodeName(node)
	}
	if firedTimer != "" {
		cause = names[firedTimer] + " fired"
	}

	outputs := evaluateGraph(running.Graph, firedTimer, read)
	inputs := inputsByNode(running.Graph)
	// a timer or waitCheck firing counts as a new rise, even when the last evaluation was true too
	fired := firedTimer != ""
	// changed: something happened worth an info line (an action ran, a wait started or checked, an
	// input of an action or wait rose or fell, a timer fired); otherwise the run logs at debug only
	changed := fired
	conditions, outcomes := []step{}, []step{}
	for _, node := range running.Graph.Nodes {
		name := names[node.ID]
		switch {
		case isAction(node.Type):
			on := outputs[node.ID]
			changed = changed || on != running.active[node.ID]
			switch {
			case !on:
				outcomes = append(outcomes, step{node.ID, name + ": not run, no input is true" + pick(running.active[node.ID], " (it can run again on the next rise)", ""), true})
			case fired || !running.active[node.ID]:
				outcomes = append(outcomes, step{node.ID, name + ": RUN, " + pick(fired, cause, "an input became true"), true})
				go manager.runAction(log.With("node", node.ID), node, running.Name+": "+name)
			default:
				outcomes = append(outcomes, step{node.ID, name + ": not run, its input was already true on the last run", true})
			}
			running.active[node.ID] = on

		case node.Type == models.NodeWaitCheck:
			if firedTimer == node.ID {
				conditions = append(conditions, step{node.ID, "check " + check(manager.tagName(node.TagID), node, values[node.TagID], outputs[node.ID]), true})
			}
			on := false
			for _, input := range inputs[node.ID] {
				on = on || outputs[input]
			}
			changed = changed || on != running.active[node.ID]
			// its own firing does not restart it, or a lasting match would repeat every delay
			rise := on && (!running.active[node.ID] || (fired && firedTimer != node.ID))
			switch {
			case !on:
				outcomes = append(outcomes, step{node.ID, name + ": not started, no input is true", true})
			case !rise:
				outcomes = append(outcomes, step{node.ID, name + pick(firedTimer == node.ID, ": not restarted, its own check does not start a new one", ": not started, its input was already true on the last run"), true})
			case running.waits[node.ID] != nil:
				outcomes = append(outcomes, step{node.ID, name + ": not started, one is already running", true})
			default:
				outcomes = append(outcomes, step{node.ID, name + ": STARTED", true})
				waitID := node.ID
				running.waits[waitID] = time.AfterFunc(time.Duration(node.DelaySeconds)*time.Second, func() {
					manager.mutex.Lock()
					delete(running.waits, waitID)
					manager.mutex.Unlock()
					manager.evaluate(running, waitID, run, "")
				})
			}
			running.active[node.ID] = on

		case isEvent(node.Type):
			// a timer that did not fire is only noise in the summary
			conditions = append(conditions, step{node.ID, name + pick(outputs[node.ID], " fired ✓", " did not fire ✗"), outputs[node.ID]})

		case node.Type == models.NodeTagValueFloat || node.Type == models.NodeTagValueBool:
			conditions = append(conditions, step{node.ID, check(name, node, values[node.TagID], outputs[node.ID]), true})

		case node.Type == models.NodeCondition:
			inputsOn := true
			for _, input := range inputs[node.ID] {
				inputsOn = inputsOn && outputs[input]
			}
			if inputsOn {
				conditions = append(conditions, step{node.ID, "if " + check(manager.tagName(node.TagID), node, values[node.TagID], outputs[node.ID]), true})
			} else {
				conditions = append(conditions, step{node.ID, name + " ✗ (its inputs are not all true)", true})
			}

		default: // and, or
			conditions = append(conditions, step{node.ID, name + " " + mark(outputs[node.ID]), true})
		}
	}

	// one readable line per run; at debug also one line per node, with its id for the editor
	summary := running.Name + ": " + joinSteps(conditions, ", ") + " → " + joinSteps(outcomes, "; ")
	if changed {
		log.Info(summary, "cause", cause)
	} else {
		log.Debug(summary+" (nothing changed)", "cause", cause)
	}
	for _, step := range append(conditions, outcomes...) {
		log.Debug(running.Name+": "+step.text, "node", step.node)
	}
}

// nodeName describes a node in words for the logs, with the tag and device names:
// "Captur battery", "turn off Captur plug", "wait 300s then Captur battery"
func (manager *AutomationsManager) nodeName(node models.AutomationNode) string {
	switch node.Type {
	case models.NodeTimer:
		return "timer " + node.Cron
	case models.NodeSunrise, models.NodeSunset:
		if node.OffsetMinutes == 0 {
			return node.Type
		}
		return fmt.Sprintf("%s %+d min", node.Type, node.OffsetMinutes)
	case models.NodeTagValueFloat, models.NodeTagValueBool:
		return manager.tagName(node.TagID)
	case models.NodeCondition:
		return "if " + manager.tagName(node.TagID)
	case models.NodeWaitCheck:
		return fmt.Sprintf("wait %ds then %s", node.DelaySeconds, manager.tagName(node.TagID))
	case models.NodeAnd:
		return "AND"
	case models.NodeOr:
		return "OR"
	case models.NodeWhatsApp:
		text := []rune(node.Text)
		if len(text) > 30 {
			text = append(text[:30], '…')
		}
		return "WhatsApp " + strconv.Quote(string(text))
	case models.NodeDevice:
		return pick(node.On, "turn on ", "turn off ") + manager.deviceName(node.DeviceID)
	case models.NodeHTTP:
		return cmp.Or(node.Method, http.MethodGet) + " " + node.URL
	}
	return node.Type + " " + node.ID
}

func (manager *AutomationsManager) tagName(tagID int) string {
	if tag := manager.tags.findTag(tagID); tag != nil && tag.Name != "" {
		return tag.Name
	}
	return "tag " + strconv.Itoa(tagID)
}

func (manager *AutomationsManager) deviceName(deviceID int) string {
	if manager.devices != nil {
		for _, device := range manager.devices.Devices() {
			if fields := device.Fields(); fields.ID == deviceID && fields.Name != "" {
				return fields.Name
			}
		}
	}
	return "device " + strconv.Itoa(deviceID)
}

func (manager *AutomationsManager) lastValue(tagID int) models.TagValue {
	tag := manager.tags.findTag(tagID)
	if tag == nil {
		return nil
	}
	return tag.LastValue()
}

// runAction does the action; name is "automation: action" in words, used in the logs
func (manager *AutomationsManager) runAction(log *slog.Logger, node models.AutomationNode, name string) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	switch node.Type {
	case models.NodeWhatsApp:
		log.Info(name + ": sending")
		sendWhatsApp(manager.cfg, manager.http, node.Text)
	case models.NodeDevice:
		device := manager.devices.PowerDevice(node.DeviceID)
		if device == nil {
			log.Error(name+": not done, it is not a power device", "device", node.DeviceID)
			return
		}
		// skip the request when the relay is already in the wanted state; an unknown state still sends it
		if on, err := device.RelayOn(ctx); err == nil && on == node.On {
			log.Info(name + ": nothing sent, it is already " + pick(on, "on", "off"))
			return
		} else if err != nil {
			log.Warn(name+": its state is unknown, sending anyway", "err", err)
		}
		if node.On {
			device.TurnOn(ctx)
		} else {
			device.TurnOff(ctx)
		}
		// TurnOn/TurnOff report their own failures, so this only says the command went out
		log.Info(name + ": command sent")
	case models.NodeHTTP:
		if _, status, err := manager.http.Do(ctx, cmp.Or(node.Method, http.MethodGet), node.URL, node.Body, node.Headers); err != nil {
			log.Error(name+": failed", "status", status, "err", err)
		} else {
			log.Info(name+": done", "status", status)
		}
	}
}

// ValidateAutomation is the check of the REST handlers before saving
func ValidateAutomation(automation *models.Automation) error {
	if automation.Name == "" {
		return fmt.Errorf("name is required")
	}
	return validateGraph(automation.Graph)
}

const automationColumns = "id, name, enabled, graph"

func (manager *AutomationsManager) ReadAllFromDB(ctx context.Context) ([]models.Automation, error) {
	var rows []models.Automation
	err := manager.db.QueryInto(ctx, &rows, "SELECT "+automationColumns+" FROM public.automation ORDER BY id")
	return rows, err
}

func (manager *AutomationsManager) ReadFromDB(ctx context.Context, id int) (*models.Automation, error) {
	var rows []models.Automation
	if err := manager.db.QueryInto(ctx, &rows, "SELECT "+automationColumns+" FROM public.automation WHERE id = $1", id); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], nil
}

func (manager *AutomationsManager) CreateInDB(ctx context.Context, automation *models.Automation) (*models.Automation, error) {
	return manager.write(ctx, "INSERT INTO public.automation (name, enabled, graph) VALUES ($1, $2, $3::jsonb) RETURNING "+automationColumns, automation)
}

func (manager *AutomationsManager) UpdateInDB(ctx context.Context, id int, automation *models.Automation) (*models.Automation, error) {
	return manager.write(ctx, "UPDATE public.automation SET name = $1, enabled = $2, graph = $3::jsonb WHERE id = $4 RETURNING "+automationColumns, automation, id)
}

func (manager *AutomationsManager) DeleteInDB(ctx context.Context, id int) error {
	deleted, err := manager.db.Exec(ctx, "DELETE FROM public.automation WHERE id = $1", id)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return manager.Reload(ctx)
}

// write runs an INSERT/UPDATE of an automation and reloads the running automations
func (manager *AutomationsManager) write(ctx context.Context, sql string, automation *models.Automation, extra ...any) (*models.Automation, error) {
	graph, err := json.Marshal(automation.Graph)
	if err != nil {
		return nil, err
	}
	var rows []models.Automation
	args := append([]any{automation.Name, automation.Enabled, string(graph)}, extra...)
	if err := manager.db.QueryInto(ctx, &rows, sql, args...); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return &rows[0], manager.Reload(ctx)
}
