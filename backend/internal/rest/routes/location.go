package routes

import (
	"net/http"
	"time"

	"od/internal/models"
	"od/internal/parse"
)

// locationBody is the body sent by the phone apps; the numbers may arrive as numbers or as strings
type locationBody struct {
	Tid       any `json:"tid"`
	Lat       any `json:"lat"`
	Lon       any `json:"lon"`
	Acc       any `json:"acc"`
	Alt       any `json:"alt"`
	Speed     any `json:"speed"`
	Timestamp any `json:"timestamp"`
}

func (h *Handlers) registerLocation(mux *http.ServeMux) {
	// POST /location/{apikey} receives the phone location from an Android app (GPSLogger, OwnTracks, HTTP Shortcuts).
	// Body: { tid, lat, lon, acc?, alt?, speed?, timestamp? }
	HandleFunc(mux, "POST /location/{apikey}", func(response http.ResponseWriter, request *http.Request) {
		if request.PathValue("apikey") != h.Cfg.LocationPostAPIKey {
			SendMessage(response, http.StatusUnauthorized, "Invalid API key")
			return
		}

		var body locationBody
		if err := ParseBody(request, &body); err != nil {
			SendMessage(response, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		if body.Lat == nil || body.Lon == nil {
			SendMessage(response, http.StatusBadRequest, "latitude and longitude are required")
			return
		}
		latitude, latitudeErr := parse.Float(body.Lat)
		longitude, longitudeErr := parse.Float(body.Lon)
		if latitudeErr != nil || longitudeErr != nil || latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
			SendMessage(response, http.StatusBadRequest, "Invalid latitude or longitude values")
			return
		}

		timestamp := parse.String(body.Timestamp)
		at := time.Now()
		if timestamp != "" {
			if parsed, err := parse.Date(timestamp); err == nil {
				at = parsed
			}
		}
		tagID, _ := parse.Int(body.Tid)
		accuracy, _ := parse.Float(body.Acc)
		altitude, _ := parse.Float(body.Alt)
		speed, _ := parse.Float(body.Speed)
		location := models.NewTagValueLocation(tagID, latitude, longitude, accuracy, altitude, speed, at)

		if err := h.Tags.SaveTagValueByTagID(request.Context(), tagID, location, timestamp); err != nil {
			SendError(response, err, "Tag not found.", "Error saving location")
			return
		}
		SendJSON(response, http.StatusCreated, true)
	})
}
