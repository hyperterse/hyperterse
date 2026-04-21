package agents

import (
	"slices"
	"strings"
	"testing"
)

func TestRegisteredRouteCountPerAgent_MatchesRuntimeEndpointLogEntries(t *testing.T) {
	n := RegisteredRouteCountPerAgent()
	entries := RuntimeEndpointLogEntries("any")
	if n != len(entries) {
		t.Fatalf("RegisteredRouteCountPerAgent()=%d but RuntimeEndpointLogEntries len=%d", n, len(entries))
	}
}

func TestRuntimeEndpointLogEntries_ListsA2AMountsForAgent(t *testing.T) {
	entries := RuntimeEndpointLogEntries("assistant")

	if len(entries) != 2 {
		t.Fatalf("expected 2 endpoint log entries, got %d", len(entries))
	}

	expected := []string{
		"GET /agent/assistant/.well-known/agent-card.json",
		"POST /agent/assistant",
	}
	for _, want := range expected {
		if !slices.Contains(entries, want) {
			t.Fatalf("expected endpoint log entry not found: %s", want)
		}
	}

	for _, entry := range entries {
		if strings.Contains(entry, "{") || strings.Contains(entry, "}") {
			t.Fatalf("endpoint entry still contains brace params: %s", entry)
		}
		if strings.HasPrefix(entry, "OPTIONS ") {
			t.Fatalf("endpoint entry should not include OPTIONS method: %s", entry)
		}
	}
}
