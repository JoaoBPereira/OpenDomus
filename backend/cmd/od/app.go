package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"od/internal/config"
	"od/internal/httpclient"
	"od/internal/managers"
	"od/internal/mcp"
	"od/internal/postgres"
	"od/internal/rest"
	"od/internal/rest/routes"
)

// app wires the managers and the servers together and owns their lifecycle
type app struct {
	cfg *config.Config

	tags        *managers.TagsManager
	devices     *managers.DevicesManager
	users       *managers.UsersManager
	tasks       *managers.TasksManager
	automations *managers.AutomationsManager
	weather     *managers.WeatherIPMAManager

	rest *rest.Server
	mcp  *mcp.Server
}

func newApp(cfg *config.Config, db *postgres.DB, debug bool) *app {
	http := httpclient.New()
	tags := managers.NewTagsManager(db, debug)
	devices := managers.NewDevicesManager(cfg, db, tags, http)

	a := &app{
		cfg:         cfg,
		tags:        tags,
		devices:     devices,
		users:       managers.NewUsersManager(db),
		tasks:       managers.NewTasksManager(cfg, db, tags, devices, http),
		automations: managers.NewAutomationsManager(cfg, db, tags, devices, http),
		weather:     managers.NewWeatherIPMAManager(http, ""),
	}
	tags.OnTagValue(a.automations.TagValueChanged)

	handlers := routes.New(routes.Deps{
		Cfg:         cfg,
		DB:          db,
		Tags:        a.tags,
		Devices:     a.devices,
		Automations: a.automations,
		Weather:     a.weather,
		Users:       a.users,
		Restart:     a.restart,
	})
	a.rest = rest.New(cfg, handlers)
	a.mcp = mcp.New(cfg.MCPAuthToken, a.tags, a.devices, db)
	a.mcp.Mount(a.rest.Mux())
	return a
}

// Start creates the admin user when it is missing and starts the managers in dependency order:
// the devices need the tags, the tasks and the automations need both
func (a *app) Start(ctx context.Context) error {
	if err := a.ensureAdmin(ctx); err != nil {
		return err
	}
	if err := a.tags.Start(ctx); err != nil {
		return fmt.Errorf("tags manager: %w", err)
	}
	if err := a.devices.Start(ctx); err != nil {
		return fmt.Errorf("devices manager: %w", err)
	}
	if err := a.tasks.Start(); err != nil {
		return fmt.Errorf("tasks manager: %w", err)
	}
	if err := a.automations.Start(ctx); err != nil {
		return fmt.Errorf("automations manager: %w", err)
	}
	if err := a.weather.Start(); err != nil {
		return fmt.Errorf("weather manager: %w", err)
	}

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)

	return nil
}

// Stop stops the managers in the reverse order of Start
func (a *app) Stop() {
	a.weather.Stop()
	a.automations.Stop()
	a.tasks.Stop()
	a.devices.Stop()
	a.tags.Stop()
}

// restart reloads everything from the database (POST /restart); the HTTP and MCP servers keep running
func (a *app) restart(ctx context.Context) error {
	slog.Info("restarting the managers")
	a.Stop()
	return a.Start(ctx)
}

func (a *app) ensureAdmin(ctx context.Context) error {
	exists, err := a.users.Exists(ctx, "admin")
	if err != nil {
		return fmt.Errorf("checking the admin user: %w", err)
	}
	if exists {
		return nil
	}
	if a.cfg.AdminPassword == "" {
		return errors.New("ADMIN_PASSWORD environment variable is required to create the admin user")
	}
	slog.Info("creating the admin user")
	return a.users.Register(ctx, "admin", a.cfg.AdminPassword, "admin")
}
