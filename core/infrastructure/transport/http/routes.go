package http

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"

	"github.com/hyperterse/hyperterse/core/domain/interfaces"
	"github.com/hyperterse/hyperterse/core/infrastructure/logging"
	"github.com/hyperterse/hyperterse/core/proto/hyperterse"
	"github.com/hyperterse/hyperterse/core/runtime/handlers"
)

// RegisterRoutes registers all HTTP routes
func RegisterRoutes(
	r *chi.Mux,
	queryService interfaces.QueryService,
	mcpService interfaces.MCPService,
	model *hyperterse.Model,
	port string,
	shutdownCtx context.Context,
) {
	log := logging.New("routes")
	log.Infof("Registering HTTP routes")

	var utilityRoutes []string
	var queryRoutes []string

	// Register MCP endpoint - Streamable HTTP transport
	r.Route("/mcp", func(r chi.Router) {
		r.Options("/", handleMCPOptions)
		r.Post("/", handleMCPPost(queryService, mcpService, shutdownCtx))
		r.Get("/", handleMCPGet(shutdownCtx))
		r.Delete("/", handleMCPDelete)
	})
	utilityRoutes = append(utilityRoutes, "POST /mcp (Streamable HTTP - JSON-RPC requests)")
	utilityRoutes = append(utilityRoutes, "GET /mcp (Streamable HTTP - server-initiated messages)")
	utilityRoutes = append(utilityRoutes, "DELETE /mcp (Streamable HTTP - session termination)")

	// LLM documentation endpoint
	r.Get("/llms.txt", handlers.LLMTxtHandler(model, fmt.Sprintf("http://localhost:%s", port)))
	utilityRoutes = append(utilityRoutes, "GET /llms.txt")

	// OpenAPI/Swagger docs endpoint
	r.Get("/docs", handlers.GenerateOpenAPISpecHandler(model, fmt.Sprintf("http://localhost:%s", port)))
	utilityRoutes = append(utilityRoutes, "GET /docs")

	// Heartbeat endpoint for health checks
	r.Get("/heartbeat", handleHeartbeat)
	utilityRoutes = append(utilityRoutes, "GET /heartbeat")

	// Register individual endpoints for each query
	for _, query := range model.Queries {
		queryName := query.Name
		endpointPath := "/query/" + queryName

		r.Post(endpointPath, handleQuery(queryService, query))
		queryRoutes = append(queryRoutes, fmt.Sprintf("POST %s", endpointPath))
	}

	// Log all registered routes
	log.Infof("Routes registered: %d utility, %d query", len(utilityRoutes), len(queryRoutes))
	log.Debugf("Utility routes:")
	for _, route := range utilityRoutes {
		log.Debugf("  %s", route)
	}
	if len(queryRoutes) > 0 {
		log.Debugf("Query routes:")
		for _, route := range queryRoutes {
			log.Debugf("  %s", route)
		}
	}
}
