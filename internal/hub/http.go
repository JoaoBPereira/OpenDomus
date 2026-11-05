package hub

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxRequestSize = 1 << 20

type Server struct {
	store *Store
}

func NewHandler(store *Store) http.Handler {
	server := &Server{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", server.health)
	mux.HandleFunc("GET /api/tags", server.tags)
	mux.HandleFunc("GET /api/readings", server.readings)
	mux.HandleFunc("POST /api/readings", server.addReading)
	mux.HandleFunc("GET /api/automations", server.automations)
	mux.HandleFunc("POST /api/automations", server.addAutomation)
	mux.HandleFunc("GET /api/events", server.events)
	mux.HandleFunc("POST /api/integrations/{provider}/readings", server.integrationReading)
	mux.HandleFunc("POST /mcp", server.mcp)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) tags(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Tags())
}

func (s *Server) readings(w http.ResponseWriter, r *http.Request) {
	limit := 500
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 5000 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 5000")
			return
		}
		limit = parsed
	}
	writeJSON(w, http.StatusOK, s.store.Readings(r.URL.Query().Get("tag"), limit))
}

func (s *Server) addReading(w http.ResponseWriter, r *http.Request) {
	var reading Reading
	if err := decodeJSON(w, r, &reading); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if reading.Timestamp.IsZero() {
		reading.Timestamp = time.Now().UTC()
	}
	if _, err := s.store.AddReading(reading); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reading)
}

func (s *Server) integrationReading(w http.ResponseWriter, r *http.Request) {
	provider := strings.TrimSpace(r.PathValue("provider"))
	if provider == "" {
		writeError(w, http.StatusBadRequest, "integration provider is required")
		return
	}
	var reading Reading
	if err := decodeJSON(w, r, &reading); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	reading.Source = provider
	if reading.Timestamp.IsZero() {
		reading.Timestamp = time.Now().UTC()
	}
	if _, err := s.store.AddReading(reading); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, reading)
}

func (s *Server) automations(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Automations())
}

func (s *Server) addAutomation(w http.ResponseWriter, r *http.Request) {
	var automation Automation
	if err := decodeJSON(w, r, &automation); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.AddAutomation(automation); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, automation)
}

func (s *Server) events(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Events())
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func (s *Server) mcp(w http.ResponseWriter, r *http.Request) {
	var request mcpRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, mcpError(nil, -32700, "Parse error"))
		return
	}
	if request.JSONRPC != "2.0" {
		writeJSON(w, http.StatusBadRequest, mcpError(request.ID, -32600, "Invalid Request"))
		return
	}
	switch request.Method {
	case "initialize":
		writeJSON(w, http.StatusOK, mcpResponse(request.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": "opendomus", "version": "0.1.0"},
		}))
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
	case "tools/list":
		writeJSON(w, http.StatusOK, mcpResponse(request.ID, map[string]any{
			"tools": []map[string]any{
				{
					"name": "get_tags", "description": "Get the latest value of each home tag.",
					"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
				},
				{
					"name": "query_readings", "description": "Query recent readings, optionally filtered by tag.",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"tag":   map[string]string{"type": "string", "description": "Optional exact tag name"},
							"limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 5000},
						},
					},
				},
			},
		}))
	case "tools/call":
		s.callTool(w, request)
	default:
		writeJSON(w, http.StatusOK, mcpError(request.ID, -32601, "Method not found"))
	}
}

func (s *Server) callTool(w http.ResponseWriter, request mcpRequest) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSON(w, http.StatusOK, mcpError(request.ID, -32602, "Invalid params"))
		return
	}
	var result any
	switch params.Name {
	case "get_tags":
		result = s.store.Tags()
	case "query_readings":
		tag, _ := params.Arguments["tag"].(string)
		limit := 500
		if value, ok := params.Arguments["limit"]; ok {
			n, valid := number(value)
			if !valid || n < 1 || n > 5000 {
				writeJSON(w, http.StatusOK, mcpError(request.ID, -32602, "limit must be between 1 and 5000"))
				return
			}
			limit = int(n)
		}
		result = s.store.Readings(tag, limit)
	default:
		writeJSON(w, http.StatusOK, mcpError(request.ID, -32602, "Unknown tool"))
		return
	}
	text, err := json.Marshal(result)
	if err != nil {
		writeJSON(w, http.StatusOK, mcpError(request.ID, -32603, "Internal error"))
		return
	}
	writeJSON(w, http.StatusOK, mcpResponse(request.ID, map[string]any{
		"content": []map[string]string{{"type": "text", "text": string(text)}},
		"isError": false,
	}))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request must contain a single JSON value")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func mcpResponse(id json.RawMessage, result any) map[string]any {
	var value any
	if len(id) > 0 {
		_ = json.Unmarshal(id, &value)
	}
	return map[string]any{"jsonrpc": "2.0", "id": value, "result": result}
}

func mcpError(id json.RawMessage, code int, message string) map[string]any {
	var value any
	if len(id) > 0 {
		_ = json.Unmarshal(id, &value)
	}
	return map[string]any{
		"jsonrpc": "2.0", "id": value,
		"error": map[string]any{"code": code, "message": message},
	}
}
