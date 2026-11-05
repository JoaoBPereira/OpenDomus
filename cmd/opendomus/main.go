package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/JoaoBPereira/OpenDomus/internal/hub"
)

func main() {
	dataPath := os.Getenv("OPENDOMUS_DATA")
	if dataPath == "" {
		dataPath = "opendomus.json"
	}
	store, err := hub.NewStore(dataPath)
	if err != nil {
		log.Fatalf("load hub state: %v", err)
	}
	handler := hub.NewHandler(store)
	webDir := os.Getenv("OPENDOMUS_WEB")
	if webDir == "" {
		webDir = filepath.Join("web", "dist")
	}
	if _, err := os.Stat(filepath.Join(webDir, "index.html")); err == nil {
		mux := http.NewServeMux()
		mux.Handle("/", http.FileServer(http.Dir(webDir)))
		mux.Handle("/api/", handler)
		mux.Handle("/mcp", handler)
		handler = mux
	}

	address := os.Getenv("OPENDOMUS_ADDR")
	if address == "" {
		address = "127.0.0.1:8080"
	}
	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()

	log.Printf("OpenDomus listening on %s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}
