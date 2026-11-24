package routes

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"od/internal/managers"
	"od/internal/parse"
)

// sendListOrZero answers the rows, or the text "0" when there are none (what the frontend expects)
func sendListOrZero[T any](response http.ResponseWriter, rows []T, err error, failed string) {
	if err != nil {
		SendError(response, err, "", failed)
		return
	}
	if len(rows) == 0 {
		SendText(response, http.StatusOK, "0")
		return
	}
	SendJSON(response, http.StatusOK, rows)
}

// parseCount reads the {count} parameter: 1 to 1000, default 1
func parseCount(text string) int {
	return min(max(1, intParam(text, 1)), 1000)
}

// sendSaveResult answers the text "true" or "{false}", which is what the devices posting values expect
func sendSaveResult(response http.ResponseWriter, err error) {
	if err != nil {
		slog.Error("saving tag value failed", "err", err)
		SendText(response, http.StatusOK, "{false}")
		return
	}
	SendText(response, http.StatusOK, "true")
}

func (h *Handlers) checkSaveAPIKey(response http.ResponseWriter, request *http.Request) bool {
	key := h.Cfg.SaveTagValueAPIKey
	if key == "" || request.PathValue("apikey") != key {
		SendMessage(response, http.StatusUnauthorized, "Invalid API key.")
		return false
	}
	return true
}

func (h *Handlers) registerTagValues(mux *http.ServeMux) {
	HandleFunc(mux, "GET /latestTagValueFloat/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		value, err := h.Tags.LatestTagValue(request.Context(), id)
		if err != nil {
			SendError(response, err, "Tag not found.", "Error fetching the latest value.")
			return
		}
		SendJSON(response, http.StatusOK, value)
	}))

	// ?timestamp=... (required)
	HandleFunc(mux, "GET /nearestTagValueFloat/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		timestamp := request.URL.Query().Get("timestamp")
		if timestamp == "" {
			SendMessage(response, http.StatusBadRequest, "timestamp is required.")
			return
		}
		value, err := h.Tags.NearestTagValueFloat(request.Context(), id, timestamp)
		if errors.Is(err, managers.ErrNotFound) {
			SendText(response, http.StatusOK, "0")
			return
		}
		if err != nil {
			SendError(response, err, "", "Error fetching the nearest value.")
			return
		}
		SendJSON(response, http.StatusOK, value)
	}))

	// ?count=1&offset=0 (count 1 to 1000, default 1), newest first
	HandleFunc(mux, "GET /tagValueFloatHistory/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		query := request.URL.Query()
		offset := max(0, intParam(query.Get("offset"), 0))
		rows, err := h.Tags.TagValueFloatHistory(request.Context(), id, parseCount(query.Get("count")), offset)
		sendListOrZero(response, rows, err, "Error fetching values.")
	}))

	// ?timestampBegin=...&timestampEnd=... (both required)
	HandleFunc(mux, "GET /tagValueFloatRange/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		query := request.URL.Query()
		timestampBegin, timestampEnd := query.Get("timestampBegin"), query.Get("timestampEnd")
		if timestampBegin == "" || timestampEnd == "" {
			SendMessage(response, http.StatusBadRequest, "timestampBegin and timestampEnd are required.")
			return
		}
		rows, err := h.Tags.TagValueFloatRange(request.Context(), id, timestampBegin, timestampEnd)
		sendListOrZero(response, rows, err, "Error fetching values.")
	}))

	HandleFunc(mux, "GET /tagValueFloatTimeSeries/{id}/{timestampBegin}/{timestampEnd}/{intervalMinutes}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		intervalMinutes := intParam(request.PathValue("intervalMinutes"), 0)
		rows, err := h.Tags.TagValueFloatTimeSeries(request.Context(), id, request.PathValue("timestampBegin"), request.PathValue("timestampEnd"), intervalMinutes)
		sendListOrZero(response, rows, err, "Error fetching the time series.")
	}))

	// ?count=1&offset=0 (count 1 to 1000, default 1), newest first
	HandleFunc(mux, "GET /tagValueBooleanHistory/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		query := request.URL.Query()
		offset := max(0, intParam(query.Get("offset"), 0))
		rows, err := h.Tags.TagValueBooleanHistory(request.Context(), id, parseCount(query.Get("count")), offset)
		sendListOrZero(response, rows, err, "Error fetching values.")
	}))

	// ?timestampBegin=...&timestampEnd=... (both required)
	HandleFunc(mux, "GET /powerConsumption/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		query := request.URL.Query()
		timestampBegin, timestampEnd := query.Get("timestampBegin"), query.Get("timestampEnd")
		if timestampBegin == "" || timestampEnd == "" {
			SendMessage(response, http.StatusBadRequest, "timestampBegin and timestampEnd are required.")
			return
		}
		consumption, err := h.Tags.PowerConsumption(request.Context(), id, timestampBegin, timestampEnd)
		if err != nil {
			SendError(response, err, "", "Error computing the power consumption.")
			return
		}
		SendText(response, http.StatusOK, strconv.FormatFloat(consumption, 'f', -1, 64))
	}))

	HandleFunc(mux, "GET /powerConsumptionNearTimestamp/{timestamp}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		rows, err := h.Tags.PowerConsumptionNearTimestamp(request.Context(), request.PathValue("timestamp"))
		sendListOrZero(response, rows, err, "Error fetching the power consumption.")
	}))

	HandleFunc(mux, "GET /latestTagValueLocation/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		value, err := h.Tags.LatestTagValueLocation(request.Context(), id)
		if err != nil {
			SendError(response, err, "Tag not found.", "Error fetching the latest location.")
			return
		}
		SendJSON(response, http.StatusOK, value)
	}))

	// ?count=1&offset=0 (count 1 to 1000, default 1), newest first
	HandleFunc(mux, "GET /tagValueLocationHistory/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		query := request.URL.Query()
		offset := max(0, intParam(query.Get("offset"), 0))
		rows, err := h.Tags.TagValueLocationHistory(request.Context(), id, parseCount(query.Get("count")), offset)
		sendListOrZero(response, rows, err, "Error fetching locations.")
	}))

	// ?startDate=...&endDate=... (both required), oldest first
	HandleFunc(mux, "GET /tagValueLocationRange/{id}", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		query := request.URL.Query()
		startDate, endDate := query.Get("startDate"), query.Get("endDate")
		if startDate == "" || endDate == "" {
			SendMessage(response, http.StatusBadRequest, "startDate and endDate are required.")
			return
		}
		rows, err := h.Tags.TagValueLocationRange(request.Context(), id, startDate, endDate)
		sendListOrZero(response, rows, err, "Error fetching locations.")
	}))

	HandleFunc(mux, "GET /saveTagValue/{id}/{value}/{apikey}", func(response http.ResponseWriter, request *http.Request) {
		if !h.checkSaveAPIKey(response, request) {
			return
		}
		id, ok := ParseIDParam(response, request, "id")
		if !ok {
			return
		}
		sendSaveResult(response, h.Tags.SaveTagValueByTagID(request.Context(), id, request.PathValue("value"), ""))
	})

	HandleFunc(mux, "POST /saveTagValue/{apikey}", func(response http.ResponseWriter, request *http.Request) {
		if !h.checkSaveAPIKey(response, request) {
			return
		}
		var body struct {
			ID        any `json:"id"`
			Value     any `json:"value"`
			Timestamp any `json:"timestamp"`
		}
		if err := ParseBody(request, &body); err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		slog.Info("tag value posted", "id", body.ID, "value", body.Value, "timestamp", body.Timestamp)

		id, err := parse.Int(body.ID)
		if err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid id.")
			return
		}
		sendSaveResult(response, h.Tags.SaveTagValueByTagID(request.Context(), id, body.Value, parse.String(body.Timestamp)))
	})

	// Supports query params: ?offset=0&count=1000&tag_id=11 (defaults: offset=0, count=1000, tag_id optional)
	HandleFunc(mux, "GET /tagValueFloat", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		var tagID *int
		if id, err := strconv.Atoi(query.Get("tag_id")); err == nil {
			tagID = &id
		}
		page, err := h.Tags.TagValueFloatPage(request.Context(), intParam(query.Get("offset"), 0), intParam(query.Get("count"), 1000), tagID)
		if err != nil {
			SendError(response, err, "", "Error fetching tagValueFloat")
			return
		}
		SendJSON(response, http.StatusOK, page)
	}))
}
