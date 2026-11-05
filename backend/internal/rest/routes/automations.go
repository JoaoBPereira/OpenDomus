package routes

import (
	"net/http"

	"od/internal/managers"
	"od/internal/models"
)

func (h *Handlers) registerAutomations(mux *http.ServeMux) {
	HandleFunc(mux, "GET /automations", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		automations, err := h.Automations.ReadAllFromDB(request.Context())
		if err != nil {
			SendError(response, err, "", "Error fetching automations.")
			return
		}
		SendJSON(response, http.StatusOK, automations)
	}))

	HandleFunc(mux, "GET /automations/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		automation, err := h.Automations.ReadFromDB(request.Context(), id)
		if err != nil {
			SendError(response, err, "Automation not found.", "Error fetching the automation.")
			return
		}
		SendJSON(response, http.StatusOK, automation)
	}))

	HandleFunc(mux, "POST /automations", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		automation, ok := parseAutomation(response, request)
		if !ok {
			return
		}
		created, err := h.Automations.CreateInDB(request.Context(), automation)
		if err != nil {
			SendError(response, err, "", "Error creating the automation.")
			return
		}
		SendJSON(response, http.StatusCreated, created)
	}))

	HandleFunc(mux, "PUT /automations/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		automation, ok := parseAutomation(response, request)
		if !ok {
			return
		}
		updated, err := h.Automations.UpdateInDB(request.Context(), id, automation)
		if err != nil {
			SendError(response, err, "Automation not found.", "Error updating the automation.")
			return
		}
		SendJSON(response, http.StatusOK, updated)
	}))

	HandleFunc(mux, "DELETE /automations/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		if err := h.Automations.DeleteInDB(request.Context(), id); err != nil {
			SendError(response, err, "Automation not found.", "Error deleting the automation.")
			return
		}
		SendMessage(response, http.StatusOK, "Automation deleted.")
	}))
}

// parseAutomation reads and validates the body (the automation itself, without wrapper)
func parseAutomation(response http.ResponseWriter, request *http.Request) (*models.Automation, bool) {
	var automation models.Automation
	if err := ParseBody(request, &automation); err != nil {
		SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return nil, false
	}
	if err := managers.ValidateAutomation(&automation); err != nil {
		SendMessage(response, http.StatusBadRequest, err.Error())
		return nil, false
	}
	return &automation, true
}
