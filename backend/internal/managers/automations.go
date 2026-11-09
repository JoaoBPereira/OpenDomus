package managers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
			fire := func() { manager.evaluate(running, timerID) }
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
		manager.evaluate(running, "")
	}
}

func (manager *AutomationsManager) evaluate(running *runningAutomation, firedTimer string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	if running.stopped {
		return
	}
	outputs := evaluateGraph(running.Graph, firedTimer, manager.lastValue)
	inputs := inputsByNode(running.Graph)
	// a timer or waitCheck firing counts as a new rise, even when the last evaluation was true too
	fired := firedTimer != ""
	for _, node := range running.Graph.Nodes {
		switch {
		case isAction(node.Type):
			on := outputs[node.ID]
			if on && (fired || !running.active[node.ID]) {
				slog.Info("automation action", "automation", running.Name, "node", node.ID, "type", node.Type)
				go manager.runAction(node)
			}
			running.active[node.ID] = on

		case node.Type == models.NodeWaitCheck:
			on := false
			for _, input := range inputs[node.ID] {
				on = on || outputs[input]
			}
			// its own firing does not restart it, or a lasting match would repeat every delay
			rise := on && (!running.active[node.ID] || (fired && firedTimer != node.ID))
			if rise && running.waits[node.ID] == nil {
				waitID := node.ID
				slog.Info("automation wait started", "automation", running.Name, "node", waitID, "seconds", node.DelaySeconds)
				running.waits[waitID] = time.AfterFunc(time.Duration(node.DelaySeconds)*time.Second, func() {
					manager.mutex.Lock()
					delete(running.waits, waitID)
					manager.mutex.Unlock()
					manager.evaluate(running, waitID)
				})
			}
			running.active[node.ID] = on
		}
	}
}

func (manager *AutomationsManager) lastValue(tagID int) models.TagValue {
	tag := manager.tags.findTag(tagID)
	if tag == nil {
		return nil
	}
	return tag.LastValue()
}

func (manager *AutomationsManager) runAction(node models.AutomationNode) {
	ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
	defer cancel()

	switch node.Type {
	case models.NodeWhatsApp:
		sendWhatsApp(manager.cfg, manager.http, node.Text)
	case models.NodeDevice:
		device := manager.devices.PowerDevice(node.DeviceID)
		if device == nil {
			slog.Error("automation: no power device", "device", node.DeviceID)
			return
		}
		// skip the request when the relay is already in the wanted state; an unknown state still sends it
		if on, err := device.RelayOn(ctx); err == nil && on == node.On {
			slog.Info("automation: device already in the wanted state", "device", node.DeviceID, "on", on)
			return
		} else if err != nil {
			slog.Warn("automation: device state unknown, sending anyway", "device", node.DeviceID, "err", err)
		}
		if node.On {
			device.TurnOn(ctx)
		} else {
			device.TurnOff(ctx)
		}
	case models.NodeHTTP:
		method := node.Method
		if method == "" {
			method = http.MethodGet
		}
		if _, status, err := manager.http.Do(ctx, method, node.URL, node.Body, node.Headers); err != nil {
			slog.Error("automation: http request failed", "url", node.URL, "status", status, "err", err)
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
