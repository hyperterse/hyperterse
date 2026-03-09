package mcp

import (
	"math"
	"testing"

	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/proto/primitives"
)

func TestSearchIndex_HandlerOnlyToolIgnoresSyntheticStatement(t *testing.T) {
	tools := []*hyperterse.Tool{
		{
			Name:        "get-weather",
			Description: "Returns weather details",
			Statement:   "SELECT 1",
			// No adapter bindings => handler-only tool.
			Use: nil,
		},
		{
			Name:        "get-users",
			Description: "Returns users by email",
			Statement:   "SELECT * FROM users WHERE email = {{ inputs.email }}",
			Use:         []string{"primary"},
			Inputs: []*hyperterse.Input{
				{
					Name:        "email",
					Type:        primitives.Primitive_PRIMITIVE_STRING,
					Description: "user email",
				},
			},
		},
	}

	index := newToolSearchIndex(tools)
	hits := index.Search("select", 10)
	if len(hits) == 0 {
		t.Fatalf("expected at least one hit")
	}

	if hits[0].Tool.Name != "get-users" {
		t.Fatalf("expected adapter-backed SQL tool to rank first for statement query, got %q", hits[0].Tool.Name)
	}

	for _, hit := range hits {
		if hit.Tool.Name == "get-weather" {
			t.Fatalf("handler-only tool should not be matched by synthetic statement content")
		}
	}
}

func TestSearchIndex_HandlerOnlyToolNotPenalizedForMissingStatementAndInputs(t *testing.T) {
	tool := &hyperterse.Tool{
		Name:      "weather",
		Statement: "SELECT 1",
		// No adapter bindings => handler-only tool, statement should be ignored.
		Use: nil,
	}

	index := newToolSearchIndex([]*hyperterse.Tool{tool})
	hits := index.Search("weather", 10)
	if len(hits) != 1 {
		t.Fatalf("expected exactly one hit, got %d", len(hits))
	}

	if hits[0].RelevanceScore != 100 {
		t.Fatalf("expected perfect relevance score for exact-name handler-only query, got %d", hits[0].RelevanceScore)
	}
}

func TestSearchIndex_ConversationalQueryRanksWeatherHigh(t *testing.T) {
	tools := []*hyperterse.Tool{
		{
			Name:        "get-user",
			Description: "Get user by id, with input validation and response mapping.",
			Statement:   "SELECT * FROM users WHERE id = {{ inputs.userId }}",
			Use:         []string{"primary"},
		},
		{
			Name:        "get-weather",
			Description: "Custom weather MCP tool implemented entirely in TypeScript handler.",
			Statement:   "SELECT 1",
			Use:         nil, // handler-only
		},
	}

	index := newToolSearchIndex(tools)
	hits := index.Search("I want to know the what is the weather today", 10)
	if len(hits) == 0 {
		t.Fatalf("expected at least one hit")
	}
	if hits[0].Tool.Name != "get-weather" {
		t.Fatalf("expected weather tool to rank first for conversational weather query, got %q", hits[0].Tool.Name)
	}
	if hits[0].RelevanceScore < 60 {
		t.Fatalf("expected conversational weather query to score >= 60, got %d", hits[0].RelevanceScore)
	}
}

func TestNormalizeActiveComponentWeights_SumsToOne(t *testing.T) {
	components := []scoreComponent{
		{weight: 0.30, active: true},
		{weight: 0.25, active: false},
		{weight: 0.18, active: false},
		{weight: 0.14, active: true},
		{weight: 0.08, active: false},
		{weight: 0.05, active: true},
	}

	normalized := normalizeActiveComponentWeights(components)

	sum := 0.0
	for _, component := range normalized {
		if component.active {
			sum += component.weight
			continue
		}
		if component.weight != 0 {
			t.Fatalf("expected inactive component weight to be 0, got %f", component.weight)
		}
	}

	if math.Abs(sum-1.0) > 1e-9 {
		t.Fatalf("expected active component weights to sum to 1.0, got %f", sum)
	}
}

func TestNormalizeActiveComponentWeights_AllInactiveBecomeZero(t *testing.T) {
	components := []scoreComponent{
		{weight: 0.30, active: false},
		{weight: 0.25, active: false},
	}

	normalized := normalizeActiveComponentWeights(components)
	for _, component := range normalized {
		if component.weight != 0 {
			t.Fatalf("expected zero weight when all components are inactive, got %f", component.weight)
		}
	}
}

