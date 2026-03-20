package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	protoconnectors "github.com/hyperterse/hyperterse/core/proto/connectors"
)

func TestNewSQLiteConnector_LocalMemory(t *testing.T) {
	t.Parallel()

	connector, err := NewSQLiteConnector(&protoconnectors.ConnectorDef{
		ConnectionString: ":memory:",
		Config:           &protoconnectors.ConnectorConfig{},
	})
	if err != nil {
		t.Fatalf("expected sqlite memory connector to initialize: %v", err)
	}
	defer func() { _ = connector.Close() }()

	results, err := connector.Execute(context.Background(), "SELECT 1 AS value", nil)
	if err != nil {
		t.Fatalf("expected query to execute: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected exactly 1 row, got %d", len(results))
	}

	gotValue := results[0]["value"]
	if gotValue != int64(1) {
		t.Fatalf("expected value=1, got %#v", gotValue)
	}
}

func TestNewSQLiteConnector_LocalFile(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "app.db")
	connector, err := NewSQLiteConnector(&protoconnectors.ConnectorDef{
		ConnectionString: dbPath,
		Config:           &protoconnectors.ConnectorConfig{},
	})
	if err != nil {
		t.Fatalf("expected sqlite file connector to initialize: %v", err)
	}
	defer func() { _ = connector.Close() }()

	results, err := connector.Execute(context.Background(), "SELECT sqlite_version() AS version", nil)
	if err != nil {
		t.Fatalf("expected query to execute: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected exactly 1 row, got %d", len(results))
	}

	version, ok := results[0]["version"].(string)
	if !ok || version == "" {
		t.Fatalf("expected sqlite version string, got %#v", results[0]["version"])
	}
}

func TestNewSQLiteConnector_RemoteHTTP_ParsesPipelineAndPassesOptions(t *testing.T) {
	t.Parallel()

	type observedRequest struct {
		path      string
		query     string
		auth      string
		statement string
	}

	var (
		mu       sync.Mutex
		observed observedRequest
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		observed.path = r.URL.Path
		observed.query = r.URL.RawQuery
		observed.auth = r.Header.Get("Authorization")
		mu.Unlock()

		var req sqlitePipelineRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if len(req.Requests) != 1 || req.Requests[0].Stmt == nil || req.Requests[0].Stmt.SQL == nil {
			http.Error(w, "unexpected request payload", http.StatusBadRequest)
			return
		}

		mu.Lock()
		observed.statement = *req.Requests[0].Stmt.SQL
		mu.Unlock()

		colName := "id"
		colType := "INTEGER"
		stmtResultRaw, err := json.Marshal(sqliteStmtResult{
			Cols: []sqliteColumn{
				{Name: &colName, Type: &colType},
			},
			Rows: [][]sqliteValue{
				{
					{Type: "integer", Value: "1"},
				},
			},
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := sqlitePipelineResponse{
			Results: []sqliteStreamResult{
				{
					Response: &sqliteStreamResponse{
						Type:   "execute",
						Result: stmtResultRaw,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	connector, err := NewSQLiteConnector(&protoconnectors.ConnectorDef{
		ConnectionString: server.URL + "/tenant",
		Options: map[string]string{
			"authToken":     "top-secret-token",
			"sync_interval": "5s",
		},
		Config: &protoconnectors.ConnectorConfig{},
	})
	if err != nil {
		t.Fatalf("expected sqlite remote connector to initialize: %v", err)
	}
	defer func() { _ = connector.Close() }()

	results, err := connector.Execute(context.Background(), "SELECT 1 AS id", nil)
	if err != nil {
		t.Fatalf("expected remote query to execute: %v", err)
	}

	mu.Lock()
	got := observed
	mu.Unlock()

	if got.path != "/tenant/v2/pipeline" {
		t.Fatalf("expected request path '/tenant/v2/pipeline', got %q", got.path)
	}
	if !strings.Contains(got.query, "sync_interval=5s") {
		t.Fatalf("expected passthrough option in query string, got %q", got.query)
	}
	if got.auth != "Bearer top-secret-token" {
		t.Fatalf("expected bearer token auth header, got %q", got.auth)
	}
	if got.statement != "SELECT 1 AS id" {
		t.Fatalf("expected statement to be forwarded unchanged, got %q", got.statement)
	}

	if len(results) != 1 {
		t.Fatalf("expected exactly 1 row, got %d", len(results))
	}
	if results[0]["id"] != int64(1) {
		t.Fatalf("expected id=1, got %#v", results[0]["id"])
	}
}

func TestNewSQLiteConnector_RejectsWebSocketURLs(t *testing.T) {
	t.Parallel()

	_, err := NewSQLiteConnector(&protoconnectors.ConnectorDef{
		ConnectionString: "ws://example.com/db",
		Config:           &protoconnectors.ConnectorConfig{},
	})
	if err == nil {
		t.Fatalf("expected websocket URL to be rejected")
	}
	if !strings.Contains(err.Error(), "ws:// and wss://") {
		t.Fatalf("expected websocket rejection error, got: %v", err)
	}
}
