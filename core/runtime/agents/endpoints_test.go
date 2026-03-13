package agents

import (
	"slices"
	"strings"
	"testing"
)

func TestRuntimeEndpointLogEntries_EnumeratesADKRoutesWithColonParams(t *testing.T) {
	entries := RuntimeEndpointLogEntries("assistant")

	if len(entries) != 18 {
		t.Fatalf("expected 18 endpoint log entries, got %d", len(entries))
	}

	expected := []string{
		"POST /agent/assistant/run",
		"POST /agent/assistant/run_sse",
		"GET /agent/assistant/list-apps",
		"GET /agent/assistant/apps/:appName/users/:userId/sessions/:sessionId",
		"DELETE /agent/assistant/apps/:appName/users/:userId/sessions/:sessionId/artifacts/:artifactName",
		"POST /agent/assistant/apps/:appName/eval_sets/:evalSetName",
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
