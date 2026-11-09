package routes

import "net/http"

func (h *Handlers) registerWeather(mux *http.ServeMux) {
	HandleFunc(mux, "GET /weather/ipma", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		forecast := h.Weather.WeatherData()
		if forecast == nil {
			SendMessage(response, http.StatusServiceUnavailable, "IPMA weather data not available yet")
			return
		}
		SendJSON(response, http.StatusOK, forecast)
	}))
}
