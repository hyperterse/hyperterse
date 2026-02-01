package connectors

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestNewMongoDBConnector_Integration(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set, skipping MongoDB connector integration test")
	}

	conn, err := NewMongoDBConnector(uri, nil)
	if err != nil {
		t.Fatalf("NewMongoDBConnector failed: %v", err)
	}
	defer conn.Close()

	// Execute a find with empty filter (list documents)
	stmt := `{"database":"test","collection":"_hyperterse_ping","operation":"find","filter":{},"options":{"limit":1}}`
	results, err := conn.Execute(context.Background(), stmt, nil)
	if err != nil {
		t.Fatalf("Execute find failed: %v", err)
	}
	_ = results // may be empty
}

func TestMongoDBConnector_Execute_InvalidJSON(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set, skipping MongoDB connector test")
	}

	conn, err := NewMongoDBConnector(uri, nil)
	if err != nil {
		t.Fatalf("NewMongoDBConnector failed: %v", err)
	}
	defer conn.Close()

	_, err = conn.Execute(context.Background(), "not json", nil)
	if err == nil {
		t.Error("Execute with invalid JSON should return error")
	}
}

func TestMongoDBConnector_Execute_InvalidOperation(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set, skipping MongoDB connector test")
	}

	conn, err := NewMongoDBConnector(uri, nil)
	if err != nil {
		t.Fatalf("NewMongoDBConnector failed: %v", err)
	}
	defer conn.Close()

	stmt := `{"database":"test","collection":"users","operation":"invalidOp","filter":{}}`
	_, err = conn.Execute(context.Background(), stmt, nil)
	if err == nil {
		t.Error("Execute with invalid operation should return error")
	}
}

func TestMongoStatement_JSONUnmarshal(t *testing.T) {
	// Unit test: verify statement JSON structure unmarshals correctly
	stmt := `{"database":"mydb","collection":"users","operation":"findOne","filter":{"name":"alice"},"options":{"limit":1}}`
	var s mongoStatement
	if err := json.Unmarshal([]byte(stmt), &s); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if s.Database != "mydb" || s.Collection != "users" || s.Operation != "findOne" {
		t.Errorf("unexpected values: database=%q collection=%q operation=%q", s.Database, s.Collection, s.Operation)
	}
	if s.Filter["name"] != "alice" {
		t.Errorf("unexpected filter: %v", s.Filter)
	}
}
