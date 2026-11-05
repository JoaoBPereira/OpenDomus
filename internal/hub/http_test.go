package hub

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestReadingAutomationAndMCPQuery(t *testing.T) {
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(store)
	request(t, handler, http.MethodPost, "/api/automations", `{"id":"warm","tag":"temperature","operator":"gte","value":25,"message":"It is warm"}`, http.StatusCreated)
	request(t, handler, http.MethodPost, "/api/integrations/shelly/readings", `{"tags":{"temperature":26.5}}`, http.StatusCreated)

	tags := request(t, handler, http.MethodGet, "/api/tags", "", http.StatusOK)
	var tagValues []TagValue
	if err := json.Unmarshal(tags, &tagValues); err != nil {
		t.Fatal(err)
	}
	if len(tagValues) != 1 || tagValues[0].Name != "temperature" || tagValues[0].Source != "shelly" {
		t.Fatalf("unexpected tags response: %s", tags)
	}

	events := request(t, handler, http.MethodGet, "/api/events", "", http.StatusOK)
	var triggered []Event
	if err := json.Unmarshal(events, &triggered); err != nil {
		t.Fatal(err)
	}
	if len(triggered) != 1 || triggered[0].Message != "It is warm" {
		t.Fatalf("automation did not trigger: %s", events)
	}

	mcp := request(t, handler, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"query_readings","arguments":{"tag":"temperature"}}}`, http.StatusOK)
	var response struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(mcp, &response); err != nil {
		t.Fatal(err)
	}
	var readings []Reading
	if len(response.Result.Content) != 1 || json.Unmarshal([]byte(response.Result.Content[0].Text), &readings) != nil || len(readings) != 1 {
		t.Fatalf("unexpected MCP query result: %s", mcp)
	}
}

func TestStorePersistsReadings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hub", "state.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	request(t, NewHandler(store), http.MethodPost, "/api/readings", `{"source":"test","tags":{"power":12}}`, http.StatusCreated)

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Readings("power", 10); len(got) != 1 || got[0].Source != "test" {
		t.Fatalf("persisted readings not restored: %#v", got)
	}
}

func TestRejectsInvalidReadingAndMCPLimit(t *testing.T) {
	store, _ := NewStore("")
	handler := NewHandler(store)
	request(t, handler, http.MethodPost, "/api/readings", `{"source":"test","tags":{}}`, http.StatusBadRequest)
	response := request(t, handler, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"query_readings","arguments":{"limit":9000}}}`, http.StatusOK)
	var result struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response, &result); err != nil {
		t.Fatal(err)
	}
	if result.Error.Message != "limit must be between 1 and 5000" {
		t.Fatalf("unexpected MCP validation response: %s", response)
	}
}

func request(t *testing.T, handler http.Handler, method, path, body string, status int) []byte {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
	if recorder.Code != status {
		t.Fatalf("%s %s: got status %d, want %d; body: %s", method, path, recorder.Code, status, recorder.Body.String())
	}
	return recorder.Body.Bytes()
}
