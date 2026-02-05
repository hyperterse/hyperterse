//! HTTP request handlers for the Hyperterse server
//!
//! This module contains handlers for query execution, MCP protocol,
//! OpenAPI documentation, and LLM documentation.

mod llms;
mod mcp;
mod openapi;
mod query;

pub use llms::LlmsHandler;
pub use mcp::McpHandler;
pub use openapi::OpenApiHandler;
pub use query::QueryHandler;
