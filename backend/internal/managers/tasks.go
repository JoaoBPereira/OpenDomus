package managers

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"od/internal/config"
	"od/internal/httpclient"
	"od/internal/postgres"
	"od/internal/schedule"
)

// TasksManager runs the house automations: the gate alarm and the P1/P2 plug rule every 10
// seconds, the hot water check at 19:30 and the cleanup at 01:00
type TasksManager struct {
	cfg     *config.Config
	db      *postgres.DB
	tags    *TagsManager
	devices *DevicesManager
	http    *httpclient.Client

	jobs              []*schedule.Job
	gateLastAlertTime time.Time
}

func NewTasksManager(cfg *config.Config, db *postgres.DB, tags *TagsManager, devices *DevicesManager, http *httpclient.Client) *TasksManager {
	return &TasksManager{cfg: cfg, db: db, tags: tags, devices: devices, http: http}
}

// jobTimeout bounds one run of a job
const jobTimeout = time.Minute

func (manager *TasksManager) Start() error {
	jobs := []struct {
		expression string
		run        func(ctx context.Context)
	}{
		{"00 01 * * *", manager.maintenance},
	}
	for _, job := range jobs {
		run := job.run
		scheduled, err := schedule.New(job.expression, func() {
			ctx, cancel := context.WithTimeout(context.Background(), jobTimeout)
			defer cancel()
			run(ctx)
		})
		if err != nil {
			manager.Stop()
			return err
		}
		manager.jobs = append(manager.jobs, scheduled)
	}
	return nil
}

func (manager *TasksManager) Stop() {
	for _, job := range manager.jobs {
		job.Cancel()
	}
	manager.jobs = nil
}

func (manager *TasksManager) maintenance(ctx context.Context) {
	slog.Info("maintenance job started")
	manager.cleanLogTable(ctx)
	if err := manager.tags.DeleteExpiredTagValues(ctx); err != nil {
		slog.Error("deleting expired tag values failed", "err", err)
	}
	slog.Info("maintenance job finished")
}

// cleanLogTable removes the rows of the public.log table older than 24 hours
func (manager *TasksManager) cleanLogTable(ctx context.Context) {
	deleted, err := manager.db.Exec(ctx, "DELETE FROM public.log WHERE date < NOW() - INTERVAL '24 hours'")
	if err != nil {
		slog.Error("cleaning the log table failed", "err", err)
		return
	}
	slog.Info("log table cleaned", "rows", deleted)
}

// sendWhatsApp sends the text in the background through callmebot
func sendWhatsApp(cfg *config.Config, client *httpclient.Client, text string) {
	whatsApp := cfg.WhatsApp
	if whatsApp.Phone == "" || whatsApp.APIKey == "" {
		slog.Error("whatsapp: WHATSAPP_PHONE and WHATSAPP_API_KEY are required")
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, status, err := client.Get(ctx, fmt.Sprintf("https://api.callmebot.com/whatsapp.php?phone=%s&text=%s&apikey=%s", whatsApp.Phone, url.QueryEscape(text), whatsApp.APIKey))
		if err != nil {
			slog.Error("whatsapp: sending failed", "err", err)
			return
		}
		slog.Info("whatsapp: sent", "status", status)
	}()
}
