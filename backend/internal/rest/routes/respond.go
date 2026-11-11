package routes

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"od/internal/managers"
)

func SendJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	json.NewEncoder(response).Encode(body)
}

// SendMessage answers { "message": message }
func SendMessage(response http.ResponseWriter, status int, message string) {
	SendJSON(response, status, map[string]string{"message": message})
}

func SendText(response http.ResponseWriter, status int, text string) {
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.WriteHeader(status)
	response.Write([]byte(text))
}

// SendError answers a manager error: 404 with notFound for ErrNotFound, otherwise 500 with
// failed, logging the cause
func SendError(response http.ResponseWriter, err error, notFound string, failed string) {
	if errors.Is(err, managers.ErrNotFound) {
		SendMessage(response, http.StatusNotFound, notFound)
		return
	}
	// Log as the route handler that called SendError, so the log prefix names it instead of this helper
	var pcs [1]uintptr
	runtime.Callers(2, pcs[:])
	record := slog.NewRecord(time.Now(), slog.LevelError, failed, pcs[0])
	record.Add("err", err)
	slog.Default().Handler().Handle(context.Background(), record)
	SendMessage(response, http.StatusInternalServerError, failed)
}

// ParseBody reads a JSON body or an URL-encoded form into target
func ParseBody(request *http.Request, target any) error {
	if strings.HasPrefix(request.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		if err := request.ParseForm(); err != nil {
			return err
		}
		formValues := map[string]any{}
		for key, values := range request.PostForm {
			formValues[key] = values[0]
		}
		formJSON, err := json.Marshal(formValues)
		if err != nil {
			return err
		}
		return json.Unmarshal(formJSON, target)
	}
	return json.NewDecoder(request.Body).Decode(target)
}

// ParseIDParam reads a numeric path parameter. Answers 400 and returns false when it is not a number.
func ParseIDParam(response http.ResponseWriter, request *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(request.PathValue(name))
	if err != nil {
		SendMessage(response, http.StatusBadRequest, "Invalid id.")
		return 0, false
	}
	return id, true
}

// intParam reads a numeric query or path value, with a fallback when it is missing or not a number
func intParam(value string, fallback int) int {
	number, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return number
}

// HandleFunc registers a route. Express matched paths case-insensitively, so when the pattern
// has uppercase letters it is also registered in lowercase: the frontend calls
// /powerconsumption/... for the /powerConsumption/... route.
func HandleFunc(mux *http.ServeMux, pattern string, handler http.HandlerFunc) {
	mux.HandleFunc(pattern, handler)
	if lowercase := lowercaseLiteralSegments(pattern); lowercase != pattern {
		mux.HandleFunc(lowercase, handler)
	}
}

// lowercaseLiteralSegments lowercases the fixed parts of "METHOD /path/{param}" and keeps the {param} names
func lowercaseLiteralSegments(pattern string) string {
	method, path, found := strings.Cut(pattern, " ")
	if !found {
		return pattern
	}
	segments := strings.Split(path, "/")
	for index, segment := range segments {
		if !strings.HasPrefix(segment, "{") {
			segments[index] = strings.ToLower(segment)
		}
	}
	return method + " " + strings.Join(segments, "/")
}
