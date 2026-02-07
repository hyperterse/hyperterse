//! MCP (Model Context Protocol) handler

use axum::{
    extract::State,
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    Json,
};
use hyperterse_types::runtime::{error_codes, McpResponse};
use serde_json::json;
use tracing::{error, info};

use crate::state::{AppState, MCP_LATEST_PROTOCOL_VERSION, MCP_SESSION_ID_HEADER};

/// Handler for MCP protocol requests
pub struct McpHandler;

impl McpHandler {
    /// Handle POST /mcp (JSON-RPC 2.0)
    pub async fn handle_rpc(
        State(state): State<AppState>,
        headers: HeaderMap,
        Json(message): Json<serde_json::Value>,
    ) -> Response {
        let jsonrpc = message
            .get("jsonrpc")
            .and_then(|v| v.as_str())
            .unwrap_or_default();

        if jsonrpc != "2.0" {
            return (
                StatusCode::BAD_REQUEST,
                Json(McpResponse::error(
                    message.get("id").cloned().unwrap_or(json!(null)),
                    error_codes::INVALID_REQUEST,
                    "Invalid JSON-RPC version",
                )),
            )
                .into_response();
        }

        // Responses or notifications from the client can be acknowledged with 202.
        // (Streamable HTTP transport spec)
        let method = message.get("method").and_then(|v| v.as_str());
        let id = message.get("id").cloned();

        // Session management is optional: if a client provides MCP-Session-Id we
        // accept it, but we do NOT reject requests that omit it.  This allows
        // simple / direct MCP connections (e.g. Claude Desktop, curl) to work
        // without first calling initialize to obtain a session.

        // If this is a JSON-RPC response (no method, has result/error), accept it.
        if method.is_none() && (message.get("result").is_some() || message.get("error").is_some()) {
            return StatusCode::ACCEPTED.into_response();
        }

        let Some(method) = method else {
            return (
                StatusCode::BAD_REQUEST,
                Json(McpResponse::error(
                    id.unwrap_or(json!(null)),
                    error_codes::INVALID_REQUEST,
                    "Invalid JSON-RPC message",
                )),
            )
                .into_response();
        };

        // Notifications: accept and do not respond with a JSON body.
        // (Most notably: notifications/initialized)
        if id.is_none() {
            info!("MCP notification: method={}", method);
            return StatusCode::ACCEPTED.into_response();
        }

        let id = id.unwrap();

        info!("MCP request: method={}", method);
        let params = message.get("params").cloned().unwrap_or(json!({}));

        match method {
            "tools/list" => Self::handle_tools_list(&state, id, &headers)
                .await
                .into_response(),
            "tools/call" => Self::handle_tools_call(&state, id, params, &headers)
                .await
                .into_response(),
            "initialize" => Self::handle_initialize(&state, id, &headers).await,
            "ping" => Self::handle_ping(id).into_response(),
            _ => (
                StatusCode::OK,
                Json(McpResponse::error(
                    id,
                    error_codes::METHOD_NOT_FOUND,
                    format!("Method not found: {}", method),
                )),
            )
                .into_response(),
        }
    }

    /// Handle GET /mcp (SSE endpoint for server-initiated messages)
    pub async fn handle_sse(State(state): State<AppState>, headers: HeaderMap) -> Response {
        use axum::response::sse::{Event, KeepAlive, Sse};
        use futures::StreamExt;
        use std::convert::Infallible;
        use tokio_stream::wrappers::BroadcastStream;

        // Resolve session: use existing session if header provided, otherwise
        // create an ephemeral session so that sessionless clients can still
        // open an SSE stream.
        let session = if let Some(session_id) = headers
            .get(MCP_SESSION_ID_HEADER)
            .and_then(|v| v.to_str().ok())
        {
            match state.mcp_sessions.get(session_id).await {
                Some(s) => s,
                None => {
                    return (
                        StatusCode::NOT_FOUND,
                        Json(json!({"error": "Unknown MCP session"})),
                    )
                        .into_response();
                }
            }
        } else {
            // No session header — create an ephemeral session for this SSE
            // connection so the stream machinery works without auth.
            let ephemeral_id = state.mcp_sessions.create().await;
            state.mcp_sessions.get(&ephemeral_id).await.unwrap()
        };

        let rx = session.tx.subscribe();
        let session_for_events = session.clone();
        let stream = BroadcastStream::new(rx).filter_map(move |msg| {
            let session = session_for_events.clone();
            async move {
                match msg {
                    Ok(value) => {
                        let id = session.next_event_seq().to_string();
                        let data =
                            serde_json::to_string(&value).unwrap_or_else(|_| "{}".to_string());
                        Some(Ok::<Event, Infallible>(Event::default().id(id).data(data)))
                    }
                    Err(_) => None,
                }
            }
        });

        // Prime the client with an event id + empty data field (recommended by spec).
        let priming_event = {
            let session = session.clone();
            let id = session.next_event_seq().to_string();
            futures::stream::once(async move {
                Ok::<Event, Infallible>(Event::default().id(id).data(""))
            })
        };

        let combined = priming_event.chain(stream);

        Sse::new(combined)
            .keep_alive(KeepAlive::new())
            .into_response()
    }

    /// Handle DELETE /mcp (session termination)
    pub async fn handle_delete(State(state): State<AppState>, headers: HeaderMap) -> Response {
        info!("MCP session termination requested");
        let session_id = headers
            .get(MCP_SESSION_ID_HEADER)
            .and_then(|v| v.to_str().ok())
            .map(|s| s.to_string());

        // If no session header is provided, just acknowledge — there's nothing
        // to terminate when sessions are not in use.
        let Some(session_id) = session_id else {
            return (
                StatusCode::OK,
                Json(json!({"message": "No session to terminate"})),
            )
                .into_response();
        };

        let removed = state.mcp_sessions.remove(&session_id).await;
        if removed {
            (
                StatusCode::OK,
                Json(json!({"message": "Session terminated"})),
            )
                .into_response()
        } else {
            (
                StatusCode::NOT_FOUND,
                Json(json!({"error": "Unknown MCP session"})),
            )
                .into_response()
        }
    }

    /// Handle initialize method
    async fn handle_initialize(
        state: &AppState,
        id: serde_json::Value,
        _headers: &HeaderMap,
    ) -> Response {
        // Accept any MCP-Protocol-Version header value (or absent).  We respond
        // with our latest supported version and let the client negotiate down if
        // needed.  This keeps the server compatible with older and newer clients
        // without rejecting them during initialization.

        // Create a new server-side session and return it in MCP-Session-Id header.
        let session_id = state.mcp_sessions.create().await;

        let result = json!({
            "protocolVersion": MCP_LATEST_PROTOCOL_VERSION,
            "capabilities": {
                "tools": {}
            },
            "serverInfo": {
                "name": "hyperterse",
                "version": env!("CARGO_PKG_VERSION")
            }
        });

        (
            StatusCode::OK,
            [(MCP_SESSION_ID_HEADER, session_id)],
            Json(McpResponse::success(id, result)),
        )
            .into_response()
    }

    /// Handle ping method
    fn handle_ping(id: serde_json::Value) -> (StatusCode, Json<McpResponse>) {
        (StatusCode::OK, Json(McpResponse::success(id, json!({}))))
    }

    /// Handle tools/list method
    async fn handle_tools_list(
        state: &AppState,
        id: serde_json::Value,
        _headers: &HeaderMap,
    ) -> (StatusCode, Json<McpResponse>) {
        let model = state.executor.model();

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
        state: &AppState,
        id: serde_json::Value,
        params: serde_json::Value,
        _headers: &HeaderMap,
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
            .map(|obj| obj.iter().map(|(k, v)| (k.clone(), v.clone())).collect())
            .unwrap_or_default();

        match state.executor.execute(tool_name, inputs).await {
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
