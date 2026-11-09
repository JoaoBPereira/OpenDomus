// Package routes holds the HTTP handlers of the REST API.
package routes

import (
	"context"
	"net/http"

	"od/internal/config"
	"od/internal/managers"
	"od/internal/postgres"
)

// Deps is what the handlers need
type Deps struct {
	Cfg         *config.Config
	DB          *postgres.DB // only /logs queries the database directly
	Tags        *managers.TagsManager
	Devices     *managers.DevicesManager
	Automations *managers.AutomationsManager
	Weather     *managers.WeatherIPMAManager
	Users       *managers.UsersManager
	// Restart stops and starts the managers again (POST /restart)
	Restart func(ctx context.Context) error
}

type Handlers struct {
	Deps
	auth *Auth
}

func New(deps Deps) *Handlers {
	return &Handlers{Deps: deps, auth: NewAuth(deps.Cfg.JWTSecret)}
}

// Register adds every route to the mux, plus the 404 answer for everything else
func (h *Handlers) Register(mux *http.ServeMux) {
	h.registerAuth(mux)
	h.registerSystem(mux)
	h.registerTagValues(mux)
	h.registerTags(mux)
	h.registerDevices(mux)
	h.registerAutomations(mux)
	h.registerLogs(mux)
	h.registerWeather(mux)
	h.registerLocation(mux)

	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		SendMessage(response, http.StatusNotFound, "Not found")
	})
}
