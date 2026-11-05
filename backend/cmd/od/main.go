// Command od is the home automation backend: it polls the devices, stores their readings and
// serves the REST and MCP APIs.
//
//	go run ./cmd/od            # reads .env from the parent directory
//	go run ./cmd/od --debug    # keep tag values in memory only (no tag value writes)
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"od/internal/config"
	"od/internal/logger"
	"od/internal/postgres"
)

func main() {
	debug := flag.Bool("debug", false, "keep tag values in memory only (no tag value writes)")
	flag.Parse()

	godotenv.Load("../.env") // the .env sits in the repo root, shared with the other services
	logger.Setup()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	if err := run(cfg, *debug); err != nil {
		slog.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, debug bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	slog.Info("starting Open Domus", "debug", debug)

	if err := postgres.EnsureDatabase(ctx, cfg.Database); err != nil {
		return err
	}
	db, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Migrate(ctx); err != nil {
		return err
	}
	logger.SetDatabase(db)
	defer logger.SetDatabase(nil) // runs before db.Close (deferred calls run last in, first out)

	app := newApp(cfg, db, debug)
	if err := app.Start(ctx); err != nil {
		return err
	}
	defer app.Stop()
	defer app.mcp.Close()

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- app.rest.ListenAndServe()
	}()
	slog.Info("listening", "port", cfg.Port)

	select {
	case <-ctx.Done():
		slog.Info("signal received, shutting down")
	case err := <-serverErrors:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.rest.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		slog.Error("http server shutdown failed", "err", err)
	}
	return nil
}
