//! MCP (Model Context Protocol) handler

use axum::{
    extract::State,
    http::{HeaderMap, StatusCode},
    response::IntoResponse,
    Json,
};
use hyperterse_types::runtime::{error_codes, McpRequest, McpResponse};
use serde_json::json;
use std::sync::Arc;
use tracing::{error, info};
use uuid::Uuid;

use crate::executor::QueryExecutor;

/// Handler for MCP protocol requests
pub struct McpHandler;

impl McpHandler {
    /// Handle POST /mcp (JSON-RPC 2.0)
    pub async fn handle_rpc(
        State(executor): State<Arc<QueryExecutor>>,
        headers: HeaderMap,
        Json(request): Json<McpRequest>,
    ) -> impl IntoResponse {
        // Validate JSON-RPC version
        if request.jsonrpc != "2.0" {
            return (
                StatusCode::BAD_REQUEST,
                Json(McpResponse::error(
                    request.id,
                    error_codes::INVALID_REQUEST,
                    "Invalid JSON-RPC version",
                )),
            );
        }

        info!("MCP request: method={}", request.method);

        match request.method.as_str() {
            "tools/list" => Self::handle_tools_list(&executor, request.id),
            "tools/call" => Self::handle_tools_call(&executor, request.id, request.params).await,
            "initialize" => Self::handle_initialize(request.id, &headers),
            "ping" => Self::handle_ping(request.id),
            _ => (
                StatusCode::OK,
                Json(McpResponse::error(
                    request.id,
                    error_codes::METHOD_NOT_FOUND,
                    format!("Method not found: {}", request.method),
                )),
            ),
        }
    }

    /// Handle GET /mcp (SSE endpoint for server-initiated messages)
    pub async fn handle_sse() -> impl IntoResponse {
        // For now, return a simple message indicating SSE is available
        // Full SSE implementation would use axum's SSE response type
        (
            StatusCode::OK,
            Json(json!({
                "message": "MCP SSE endpoint",
                "session_id": Uuid::new_v4().to_string()
            })),
        )
    }

    /// Handle DELETE /mcp (session termination)
    pub async fn handle_delete() -> impl IntoResponse {
        info!("MCP session termination requested");
        (StatusCode::OK, Json(json!({"message": "Session terminated"})))
    }

    /// Handle initialize method
    fn handle_initialize(
        id: serde_json::Value,
        _headers: &HeaderMap,
    ) -> (StatusCode, Json<McpResponse>) {
        let result = json!({
            "protocolVersion": "2024-11-05",
            "capabilities": {
                "tools": {}
            },
            "serverInfo": {
                "name": "hyperterse",
                "version": env!("CARGO_PKG_VERSION")
            }
        });

        (StatusCode::OK, Json(McpResponse::success(id, result)))
    }

    /// Handle ping method
    fn handle_ping(id: serde_json::Value) -> (StatusCode, Json<McpResponse>) {
        (StatusCode::OK, Json(McpResponse::success(id, json!({}))))
    }

    /// Handle tools/list method
    fn handle_tools_list(
        executor: &QueryExecutor,
        id: serde_json::Value,
    ) -> (StatusCode, Json<McpResponse>) {
        let model = executor.model();

        let tools: Vec<serde_json::Value> = model
            .queries
            .iter()
            .map(|query| {
                let mut properties = serde_json::Map::new();
                let mut required: Vec<String> = Vec::new();

                for input in &query.inputs {
                    let type_str = match input.primitive_type {
                        hyperterse_types::Primitive::String => "string",
                        hyperterse_types::Primitive::Int => "integer",
                        hyperterse_types::Primitive::Float => "number",
                        hyperterse_types::Primitive::Boolean => "boolean",
                        hyperterse_types::Primitive::Uuid => "string",
                        hyperterse_types::Primitive::Datetime => "string",
                    };

                    let mut prop = serde_json::Map::new();
                    prop.insert("type".to_string(), json!(type_str));
                    if let Some(desc) = &input.description {
                        prop.insert("description".to_string(), json!(desc));
                    }

                    properties.insert(input.name.clone(), serde_json::Value::Object(prop));

                    if input.required {
                        required.push(input.name.clone());
                    }
                }

                json!({
                    "name": query.name,
                    "description": query.description.as_deref().unwrap_or(""),
                    "inputSchema": {
                        "type": "object",
                        "properties": properties,
                        "required": required
                    }
                })
            })
            .collect();

        (
            StatusCode::OK,
            Json(McpResponse::success(id, json!({ "tools": tools }))),
        )
    }

    /// Handle tools/call method
    async fn handle_tools_call(
        executor: &QueryExecutor,
        id: serde_json::Value,
        params: serde_json::Value,
    ) -> (StatusCode, Json<McpResponse>) {
        let name = params.get("name").and_then(|v| v.as_str());
        let arguments = params.get("arguments");

        let Some(tool_name) = name else {
            return (
                StatusCode::OK,
                Json(McpResponse::error(
                    id,
                    error_codes::INVALID_PARAMS,
                    "Missing tool name",
                )),
            );
        };

        let inputs = arguments
            .and_then(|v| v.as_object())
            .map(|obj| {
                obj.iter()
                    .map(|(k, v)| (k.clone(), v.clone()))
                    .collect()
            })
            .unwrap_or_default();

        match executor.execute(tool_name, inputs).await {
            Ok(results) => {
                let content = json!([{
                    "type": "text",
                    "text": serde_json::to_string_pretty(&results).unwrap_or_default()
                }]);

                (
                    StatusCode::OK,
                    Json(McpResponse::success(id, json!({ "content": content }))),
                )
            }
            Err(e) => {
                error!("Tool call failed: {}", e);
                (
                    StatusCode::OK,
                    Json(McpResponse::success(
                        id,
                        json!({
                            "content": [{
                                "type": "text",
                                "text": format!("Error: {}", e.sanitized_message())
                            }],
                            "isError": true
                        }),
                    )),
                )
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mcp_response_success() {
        let response = McpResponse::success(json!(1), json!({"status": "ok"}));
        assert_eq!(response.jsonrpc, "2.0");
        assert!(response.result.is_some());
        assert!(response.error.is_none());
    }

    #[test]
    fn test_mcp_response_error() {
        let response = McpResponse::error(json!(1), error_codes::METHOD_NOT_FOUND, "Not found");
        assert_eq!(response.jsonrpc, "2.0");
        assert!(response.result.is_none());
        assert!(response.error.is_some());
    }
}
