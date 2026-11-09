// Package rest is the HTTP server of the API: the routes of the routes package behind the
// recover, security headers, CORS and body size middlewares.
package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"od/internal/config"
	"od/internal/rest/routes"
)

const maxBodyBytes = 100 * 1024

type Server struct {
	mux    *http.ServeMux
	server *http.Server
}

func New(cfg *config.Config, handlers *routes.Handlers) *Server {
	mux := http.NewServeMux()
	handlers.Register(mux)

	// Innermost first: the body limit wraps the mux, CORS wraps that, and so on
	var handler http.Handler = mux
	handler = BodyLimit(maxBodyBytes)(handler)
	handler = Helmet()(handler)
	handler = Recover()(handler)
	handler = CORS(cfg.CORSOrigins)(handler)

	return &Server{
		mux: mux,
		server: &http.Server{
			Addr:              ":" + cfg.Port,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Mux is the router, so other packages can mount their own handlers (the MCP server at /mcp)
func (s *Server) Mux() *http.ServeMux {
	return s.mux
}

// ListenAndServe serves until Shutdown is called, which is not an error
func (s *Server) ListenAndServe() error {
	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
