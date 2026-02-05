//! Runtime type definitions for request/response handling

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Query execution request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryRequest {
    /// Input parameters for the query
    #[serde(default)]
    pub inputs: HashMap<String, serde_json::Value>,
}

/// Query execution response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct QueryResponse {
    /// Whether the query succeeded
    pub success: bool,
    /// Error message if the query failed
    #[serde(default)]
    pub error: String,
    /// Query results
    #[serde(default)]
    pub results: Vec<HashMap<String, serde_json::Value>>,
}

impl QueryResponse {
    /// Create a successful response with results
    pub fn success(results: Vec<HashMap<String, serde_json::Value>>) -> Self {
        Self {
            success: true,
            error: String::new(),
            results,
        }
    }

    /// Create an error response
    pub fn error(message: impl Into<String>) -> Self {
        Self {
            success: false,
            error: message.into(),
            results: Vec::new(),
        }
    }
}

/// MCP (Model Context Protocol) JSON-RPC request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpRequest {
    /// JSON-RPC version (always "2.0")
    pub jsonrpc: String,
    /// Request ID
    pub id: serde_json::Value,
    /// Method name
    pub method: String,
    /// Method parameters
    #[serde(default)]
    pub params: serde_json::Value,
}

/// MCP JSON-RPC response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpResponse {
    /// JSON-RPC version (always "2.0")
    pub jsonrpc: String,
    /// Request ID (matches the request)
    pub id: serde_json::Value,
    /// Result on success
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<serde_json::Value>,
    /// Error on failure
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<McpError>,
}

impl McpResponse {
    /// Create a successful MCP response
    pub fn success(id: serde_json::Value, result: serde_json::Value) -> Self {
        Self {
            jsonrpc: "2.0".to_string(),
            id,
            result: Some(result),
            error: None,
        }
    }

    /// Create an error MCP response
    pub fn error(id: serde_json::Value, code: i32, message: impl Into<String>) -> Self {
        Self {
            jsonrpc: "2.0".to_string(),
            id,
            result: None,
            error: Some(McpError {
                code,
                message: message.into(),
                data: None,
            }),
        }
    }
}

/// MCP JSON-RPC error
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct McpError {
    /// Error code
    pub code: i32,
    /// Error message
    pub message: String,
    /// Additional error data
    #[serde(skip_serializing_if = "Option::is_none")]
    pub data: Option<serde_json::Value>,
}

/// Standard JSON-RPC error codes
pub mod error_codes {
    /// Parse error
    pub const PARSE_ERROR: i32 = -32700;
    /// Invalid request
    pub const INVALID_REQUEST: i32 = -32600;
    /// Method not found
    pub const METHOD_NOT_FOUND: i32 = -32601;
    /// Invalid params
    pub const INVALID_PARAMS: i32 = -32602;
    /// Internal error
    pub const INTERNAL_ERROR: i32 = -32603;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_query_response_success() {
        let results = vec![{
            let mut map = HashMap::new();
            map.insert("id".to_string(), serde_json::json!(1));
            map.insert("name".to_string(), serde_json::json!("test"));
            map
        }];

        let response = QueryResponse::success(results.clone());
        assert!(response.success);
        assert!(response.error.is_empty());
        assert_eq!(response.results, results);
    }

    #[test]
    fn test_query_response_error() {
        let response = QueryResponse::error("Something went wrong");
        assert!(!response.success);
        assert_eq!(response.error, "Something went wrong");
        assert!(response.results.is_empty());
    }

    #[test]
    fn test_mcp_response_success() {
        let response = McpResponse::success(
            serde_json::json!(1),
            serde_json::json!({"status": "ok"}),
        );
        assert_eq!(response.jsonrpc, "2.0");
        assert!(response.result.is_some());
        assert!(response.error.is_none());
    }

    #[test]
    fn test_mcp_response_error() {
        let response = McpResponse::error(
            serde_json::json!(1),
            error_codes::METHOD_NOT_FOUND,
            "Method not found",
        );
        assert_eq!(response.jsonrpc, "2.0");
        assert!(response.result.is_none());
        assert!(response.error.is_some());
        assert_eq!(response.error.as_ref().unwrap().code, error_codes::METHOD_NOT_FOUND);
    }
}
