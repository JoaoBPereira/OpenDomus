// Package mcp exposes the smart home to MCP clients over Streamable HTTP at /mcp.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"od/internal/managers"
	"od/internal/models"
	"od/internal/postgres"
)

// Server holds the MCP server and the HTTP handler that manages its sessions (Mcp-Session-Id header)
type Server struct {
	token   string // empty: no authentication
	tags    *managers.TagsManager
	devices *managers.DevicesManager
	db      *postgres.DB
	server  *mcpsdk.Server
	handler *mcpsdk.StreamableHTTPHandler
}

func New(token string, tags *managers.TagsManager, devices *managers.DevicesManager, db *postgres.DB) *Server {
	s := &Server{token: token, tags: tags, devices: devices, db: db}
	s.server = s.newServer()
	s.handler = mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return s.server }, nil)
	return s
}

// Mount serves the POST, GET and DELETE requests of /mcp on the mux
func (s *Server) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/mcp", func(response http.ResponseWriter, request *http.Request) {
		if s.token != "" && request.Header.Get("Authorization") != "Bearer "+s.token {
			response.Header().Set("Content-Type", "application/json; charset=utf-8")
			response.WriteHeader(http.StatusUnauthorized)
			response.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		s.handler.ServeHTTP(response, request)
	})
	slog.Info("mcp server mounted", "path", "/mcp")
}

func (s *Server) Close() {
	for session := range s.server.Sessions() {
		if err := session.Close(); err != nil {
			slog.Error("closing mcp session failed", "session", session.ID(), "err", err)
		}
	}
}

// Inputs of the tools. The jsonschema tags are the parameter descriptions shown to the client.

type tagLatestValueInput struct {
	TagID int `json:"tag_id" jsonschema:"The numeric ID of the tag"`
}

type tagHistoryInput struct {
	TagID int `json:"tag_id" jsonschema:"The numeric ID of the tag"`
	Limit int `json:"limit,omitempty" jsonschema:"Number of records to return (1-100, default 10)"`
}

type tagHistoryBetweenDatesInput struct {
	TagID int    `json:"tag_id" jsonschema:"The numeric ID of the tag"`
	From  string `json:"from" jsonschema:"Start datetime in ISO 8601 format (e.g. 2026-01-01T00:00:00Z)"`
	To    string `json:"to" jsonschema:"End datetime in ISO 8601 format (e.g. 2026-01-31T23:59:59Z)"`
}

// tagHistoryRow is a row of the history queries (SELECT tag_id, value, time FROM tagvaluefloat)
type tagHistoryRow struct {
	TagID int       `json:"tag_id" db:"tag_id"`
	Value float64   `json:"value" db:"value"`
	Time  time.Time `json:"time" db:"time"`
}

func textResult(text string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: text}}}
}

func jsonResult(value any) *mcpsdk.CallToolResult {
	text, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return textResult(fmt.Sprint(value))
	}
	return textResult(string(text))
}

// noInput is the schema of the tools without parameters: the SDK default omits "properties", which strict clients reject
var noInput = json.RawMessage(`{"type":"object","properties":{}}`)

func (s *Server) newServer() *mcpsdk.Server {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "od-smart-home", Version: "1.0.0"}, nil)

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_devices",
		Title:       "List Devices",
		Description: "Returns all registered smart home devices with their id, name, type, enabled state, and URL.",
		InputSchema: noInput,
	}, func(ctx context.Context, request *mcpsdk.CallToolRequest, input any) (*mcpsdk.CallToolResult, any, error) {
		devices := []map[string]any{}
		for _, device := range s.devices.Devices() {
			fields := device.Fields()
			devices = append(devices, map[string]any{"id": fields.ID, "name": fields.Name, "type": fields.Type, "enabled": fields.Enabled, "url": fields.URL})
		}
		return jsonResult(devices), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "list_tags",
		Title:       "List Tags",
		Description: "Returns all registered tags with their id, name, type, valueType, and latest value.",
		InputSchema: noInput,
	}, func(ctx context.Context, request *mcpsdk.CallToolRequest, input any) (*mcpsdk.CallToolResult, any, error) {
		tags := []map[string]any{}
		for _, tag := range s.tags.Tags() {
			tags = append(tags, map[string]any{"id": tag.ID, "name": tag.Name, "type": tag.Type, "valueType": tag.ValueType, "lastTagValue": tag.LastValue()})
		}
		return jsonResult(tags), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_tag_latest_value",
		Title:       "Get Tag Latest Value",
		Description: "Returns the most recent in-memory value for a specific tag.",
	}, func(ctx context.Context, request *mcpsdk.CallToolRequest, input tagLatestValueInput) (*mcpsdk.CallToolResult, any, error) {
		var tag *models.Tag
		for _, candidate := range s.tags.Tags() {
			if candidate.ID == input.TagID {
				tag = candidate
			}
		}
		if tag == nil {
			return textResult(fmt.Sprintf("Tag with id %d not found.", input.TagID)), nil, nil
		}
		return jsonResult(map[string]any{"id": tag.ID, "name": tag.Name, "lastTagValue": tag.LastValue()}), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_tag_history",
		Title:       "Get Tag History",
		Description: "Returns the last N float values for a specific tag (max 100).",
	}, func(ctx context.Context, request *mcpsdk.CallToolRequest, input tagHistoryInput) (*mcpsdk.CallToolResult, any, error) {
		limit := input.Limit
		if limit == 0 {
			limit = 10
		}
		if limit < 1 || limit > 100 {
			return textResult("limit must be between 1 and 100"), nil, nil
		}
		var rows []tagHistoryRow
		err := s.db.QueryInto(ctx, &rows, "SELECT tag_id, value, time FROM public.tagvaluefloat WHERE tag_id = $1 ORDER BY time DESC LIMIT $2", input.TagID, limit)
		if err != nil {
			return textResult("Error querying tag history: " + err.Error()), nil, nil
		}
		return jsonResult(rows), nil, nil
	})

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        "get_tag_history_between_dates",
		Title:       "Get Tag History Between Dates",
		Description: "Returns float values for a specific tag between two dates.",
	}, func(ctx context.Context, request *mcpsdk.CallToolRequest, input tagHistoryBetweenDatesInput) (*mcpsdk.CallToolResult, any, error) {
		var rows []tagHistoryRow
		err := s.db.QueryInto(ctx, &rows, "SELECT tag_id, value, time FROM public.tagvaluefloat WHERE tag_id = $1 AND time >= $2 AND time <= $3 ORDER BY time ASC", input.TagID, input.From, input.To)
		if err != nil {
			return textResult("Error querying tag history: " + err.Error()), nil, nil
		}
		return jsonResult(rows), nil, nil
	})

	return server
}
