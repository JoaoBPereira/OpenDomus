package routes

import (
	"context"
	"net/http"
	"time"
)

func (h *Handlers) registerSystem(mux *http.ServeMux) {
	HandleFunc(mux, "POST /restart", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		// Not the request context: a client going away must not leave the managers half started
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := h.Restart(ctx); err != nil {
			SendError(response, err, "", "Error restarting Open Domus: "+err.Error())
			return
		}
		SendMessage(response, http.StatusOK, "Open Domus restarted successfully")
	}))
}
