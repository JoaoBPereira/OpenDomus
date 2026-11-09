package routes

import (
	"fmt"
	"net/http"

	"od/internal/models"
)

func (h *Handlers) registerDevices(mux *http.ServeMux) {
	// The running devices (only the common fields: the tags and jobs are not serialized)
	HandleFunc(mux, "GET /getDevices", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		devices := []*models.DeviceFields{}
		for _, device := range h.Devices.Devices() {
			devices = append(devices, device.Fields())
		}
		SendJSON(response, http.StatusOK, devices)
	}))

	// The rows of the device table
	HandleFunc(mux, "GET /devices", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		devices, err := h.Devices.ReadAllDevicesFromDB(request.Context())
		if err != nil {
			SendError(response, err, "", "Error fetching devices.")
			return
		}
		SendJSON(response, http.StatusOK, devices)
	}))

	HandleFunc(mux, "PUT /devices/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		var body DeviceUpdateSchema
		if err := ParseBody(request, &body); err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		if message := body.Validate(); message != "" {
			SendMessage(response, http.StatusBadRequest, message)
			return
		}

		device, err := h.Devices.UpdateDeviceInDB(request.Context(), id, body.Device.ToDevice())
		if err != nil {
			SendError(response, err, fmt.Sprintf("Device with ID %d not found.", id), "Error updating device.")
			return
		}
		SendJSON(response, http.StatusOK, device)
	}))

	HandleFunc(mux, "POST /devices", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		var body DeviceCreateSchema
		if err := ParseBody(request, &body); err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		if message := body.Validate(); message != "" {
			SendMessage(response, http.StatusBadRequest, message)
			return
		}

		device, err := h.Devices.CreateDeviceInDB(request.Context(), body.Device.ToDevice())
		if err != nil {
			SendError(response, err, "", "Error creating device.")
			return
		}
		SendJSON(response, http.StatusCreated, device)
	}))
}
