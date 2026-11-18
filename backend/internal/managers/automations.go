package managers

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
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
	wasOn   map[string]bool        // inputs of the actions and waitChecks on the last evaluation
	waits   map[string]*time.Timer // waitCheck delays running
	stopped bool
}

func newRunningAutomation(automation models.Automation) *runningAutomation {
	return &runningAutomation{Automation: automation, wasOn: map[string]bool{}, waits: map[string]*time.Timer{}}
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
		running := newRunningAutomation(automation)
		for _, node := range automation.Graph.Nodes {
			if job := manager.scheduleNode(running, node); job != nil {
				running.jobs = append(running.jobs, job)
			}
		}
		manager.running = append(manager.running, running)
	}
	slog.Info("automations started", "count", len(manager.running))
	return nil
}

// scheduleNode starts the job that fires a timer, sunrise or sunset node. The other nodes have
// none, and a node that cannot be scheduled is logged and skipped: both return nil.
func (manager *AutomationsManager) scheduleNode(running *runningAutomation, node models.AutomationNode) *schedule.Job {
	fire := func() { manager.evaluate(running, node.ID, "", "") }

	switch node.Type {
	case models.NodeTimer:
		job, err := schedule.New(node.Cron, fire)
		if err != nil {
			slog.Error("automation timer skipped", "id", running.ID, "node", node.ID, "err", err)
			return nil
		}
		return job

	case models.NodeSunrise, models.NodeSunset:
		lat, lon := manager.cfg.Latitude, manager.cfg.Longitude
		if lat == 0 && lon == 0 {
			slog.Warn("automation sun trigger skipped, LATITUDE and LONGITUDE are not set", "id", running.ID, "node", node.ID)
			return nil
		}
		next := func(after time.Time) time.Time {
			return nextSunEvent(lat, lon, node.Type == models.NodeSunrise, node.OffsetMinutes, after)
		}
		slog.Info("automation sun trigger", "automation", running.Name, "node", node.ID, "next", next(time.Now()).Format("2006-01-02 15:04"))
		return schedule.NewFunc(next, fire)
	}
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
			if watchesTag(node.Type) && node.TagID == tagID {
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

// evaluate runs the graph once, runs the actions and starts the waits it decides, and logs every
// decision with the values behind it. firedNode is the timer, sunrise, sunset or waitCheck that
// fired ("" for a tag change, which says why in cause). runID is the id shown in the logs: empty
// starts a new run, a waitCheck firing passes the id of the run that started the wait so the
// chain keeps one id.
func (manager *AutomationsManager) evaluate(running *runningAutomation, firedNode string, runID string, cause string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if running.stopped {
		return
	}
	delete(running.waits, firedNode) // a wait that fired is over
	if runID == "" {
		runID = newRunID()
	}
	log := slog.With("run", runID)

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
	names, types := map[string]string{}, map[string]string{}
	for _, node := range running.Graph.Nodes {
		names[node.ID] = manager.nodeName(node)
		types[node.ID] = node.Type
	}
	fired := firedNode != ""
	if fired {
		cause = names[firedNode] + " fired"
	}

	results := evaluateGraph(running.Graph, firedNode, read)
	inputs := inputsByNode(running.Graph)
	// kinds names the inputs of a node by type for the logs ("timer or and"), all or only the true ones
	kinds := func(id string, onlyTrue bool) string { return inputTypes(inputs[id], types, results, onlyTrue) }

	// changed: something happened worth an info line (a timer or wait fired, the input of an action
	// or wait rose or fell); otherwise the run logs at debug only
	changed := fired
	conditions, outcomes := []step{}, []step{}
	for _, node := range running.Graph.Nodes {
		name, result, value := names[node.ID], results[node.ID], values[node.TagID]
		wasOn := running.wasOn[node.ID]

		switch {
		case isAction(node.Type):
			text, run := actionStep(result.input, wasOn, fired, cause, kinds(node.ID, false), kinds(node.ID, true))
			if run {
				go manager.runAction(log.With("node", node.ID), node, running.Name+": "+name)
			}
			outcomes = append(outcomes, step{node.ID, name + ": " + text, true})

		case node.Type == models.NodeWaitCheck:
			ownCheck := firedNode == node.ID
			if ownCheck {
				conditions = append(conditions, step{node.ID, "check " + check(manager.tagName(node.TagID), node, value, result.output), true})
			}
			waiting := running.waits[node.ID] != nil
			text, start := waitStep(result.input, wasOn, fired, ownCheck, waiting, kinds(node.ID, false), kinds(node.ID, true))
			if start {
				running.waits[node.ID] = time.AfterFunc(time.Duration(node.DelaySeconds)*time.Second, func() {
					manager.evaluate(running, node.ID, runID, "")
				})
			}
			outcomes = append(outcomes, step{node.ID, name + ": " + text, true})

		default: // triggers and gates
			// a timer that did not fire is only noise in the summary
			inSummary := !isEvent(node.Type) || result.output
			conditions = append(conditions, step{node.ID, manager.conditionText(node, name, value, result, kinds(node.ID, false)), inSummary})
		}

		if isAction(node.Type) || node.Type == models.NodeWaitCheck {
			changed = changed || result.input != wasOn
			running.wasOn[node.ID] = result.input
		}
	}

	// one line per decision, with the node id for the editor: at info when something changed (but
	// never the timers that did not fire, only noise), otherwise at debug
	logStep := log.Debug
	if changed {
		logStep = log.Info
		log.Info(running.Name+": run", "cause", cause)
	} else {
		log.Debug(running.Name+": run (nothing changed)", "cause", cause)
	}
	for _, step := range append(conditions, outcomes...) {
		if step.summary {
			logStep(running.Name+": "+step.text, "node", step.node)
		} else {
			log.Debug(running.Name+": "+step.text, "node", step.node)
		}
	}
}

// actionStep decides whether an action runs, and says why in words. on and wasOn are its input now
// and on the last evaluation. It runs when its input becomes true, and again on every timer or
// wait firing while it is true. all and trueOnes name its inputs by type (see inputTypes).
func actionStep(on bool, wasOn bool, fired bool, cause string, all string, trueOnes string) (text string, run bool) {
	switch {
	case !on:
		return "not run, no " + all + " is true" + pick(wasOn, " (it can run again on the next rise)", ""), false
	case fired:
		return "RUN, " + cause, true
	case !wasOn:
		return "RUN, " + trueOnes + " became true", true
	}
	return "not run, its " + trueOnes + " was already true on the last run", false
}

// waitStep decides whether a waitCheck starts its delay, and says why in words. It starts when its
// input becomes true, or on a timer or other wait firing while it is true; ownCheck is its own
// delay ending, which does not start it again, or a lasting match would repeat every delay.
// waiting is true while its delay runs.
func waitStep(on bool, wasOn bool, fired bool, ownCheck bool, waiting bool, all string, trueOnes string) (text string, start bool) {
	rise := !wasOn || (fired && !ownCheck)
	switch {
	case !on:
		return "not started, no " + all + " is true", false
	case !rise && ownCheck:
		return "not restarted, its own check does not start a new one", false
	case !rise:
		return "not started, its " + trueOnes + " was already true on the last run", false
	case waiting:
		return "not started, one is already running", false
	}
	return "STARTED", true
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
		on, err := device.RelayOn(ctx)
		if err != nil {
			log.Warn(name+": its state is unknown, sending anyway", "err", err)
		} else if on == node.On {
			log.Info(name + ": nothing sent, it is already " + pick(on, "on", "off"))
			return
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
