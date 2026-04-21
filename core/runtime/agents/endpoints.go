package agents

import (
	"fmt"
)

type agentRoute struct {
	methods []string
	pattern string
}

// RegisteredRouteCountPerAgent is the number of HTTP method+path pairs served under each /agent/<name>/ mount.
func RegisteredRouteCountPerAgent() int {
	n := 0
	for _, route := range agentRoutes {
		n += len(route.methods)
	}
	return n
}

// RuntimeEndpointLogEntries returns the concrete A2A mount entries for one agent.
func RuntimeEndpointLogEntries(agentName string) []string {
	prefix := fmt.Sprintf("/agent/%s", agentName)
	entries := make([]string, 0, len(agentRoutes)*2)
	for _, route := range agentRoutes {
		path := prefix + route.pattern
		for _, method := range route.methods {
			entries = append(entries, fmt.Sprintf("%s %s", method, path))
		}
	}
	return entries
}

var agentRoutes = []agentRoute{
	{methods: []string{"GET"}, pattern: "/.well-known/agent-card.json"},
	{methods: []string{"POST"}, pattern: ""},
}
