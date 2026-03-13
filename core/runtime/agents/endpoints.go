package agents

import (
	"fmt"
	"strings"
)

type adkRoute struct {
	methods []string
	pattern string
}

// RuntimeEndpointLogEntries returns concrete ADK REST endpoint log entries for one agent.
// Path parameters are normalized to :paramName style (e.g. {user_id} -> :userId).
func RuntimeEndpointLogEntries(agentName string) []string {
	prefix := fmt.Sprintf("/agent/%s", agentName)
	entries := make([]string, 0, len(adkRoutes)*2)
	for _, route := range adkRoutes {
		path := prefix + normalizeRoutePattern(route.pattern)
		for _, method := range route.methods {
			entries = append(entries, fmt.Sprintf("%s %s", method, path))
		}
	}
	return entries
}

var adkRoutes = []adkRoute{
	// Runtime API
	{methods: []string{"POST"}, pattern: "/run"},
	{methods: []string{"POST"}, pattern: "/run_sse"},

	// Sessions API
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}"},
	{methods: []string{"POST"}, pattern: "/apps/{app_name}/users/{user_id}/sessions"},
	{methods: []string{"POST"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}"},
	{methods: []string{"DELETE"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}"},
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/users/{user_id}/sessions"},

	// Apps API
	{methods: []string{"GET"}, pattern: "/list-apps"},

	// Debug API
	{methods: []string{"GET"}, pattern: "/debug/trace/{event_id}"},
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}/events/{event_id}/graph"},
	{methods: []string{"GET"}, pattern: "/debug/trace/session/{session_id}"},

	// Artifacts API
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}/artifacts"},
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}/artifacts/{artifact_name}"},
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}/artifacts/{artifact_name}/versions/{version}"},
	{methods: []string{"DELETE"}, pattern: "/apps/{app_name}/users/{user_id}/sessions/{session_id}/artifacts/{artifact_name}"},

	// Eval API
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/eval_sets"},
	{methods: []string{"POST"}, pattern: "/apps/{app_name}/eval_sets/{eval_set_name}"},
	{methods: []string{"GET"}, pattern: "/apps/{app_name}/eval_results"},
}

func normalizeRoutePattern(pattern string) string {
	if pattern == "" {
		return ""
	}
	var out strings.Builder
	out.Grow(len(pattern))

	for i := 0; i < len(pattern); i++ {
		if pattern[i] != '{' {
			out.WriteByte(pattern[i])
			continue
		}
		end := strings.IndexByte(pattern[i:], '}')
		if end == -1 {
			out.WriteByte(pattern[i])
			continue
		}
		end += i
		param := pattern[i+1 : end]
		out.WriteByte(':')
		out.WriteString(snakeToLowerCamel(param))
		i = end
	}

	return out.String()
}

func snakeToLowerCamel(input string) string {
	parts := strings.FieldsFunc(input, func(r rune) bool { return r == '_' || r == '-' })
	if len(parts) == 0 {
		return input
	}
	var out strings.Builder
	out.WriteString(strings.ToLower(parts[0]))
	for _, part := range parts[1:] {
		if part == "" {
			continue
		}
		out.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			out.WriteString(strings.ToLower(part[1:]))
		}
	}
	return out.String()
}
