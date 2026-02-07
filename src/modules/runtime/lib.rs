//! Runtime server for Hyperterse
//!
//! This crate provides the HTTP server, database connectors, query execution,
//! and request handlers for the Hyperterse query layer.

pub mod connectors;
pub mod executor;
pub mod handlers;
pub mod server;
pub mod state;

pub use connectors::{Connector, ConnectorManager};
pub use executor::QueryExecutor;
pub use handlers::{LlmsHandler, McpHandler, OpenApiHandler, QueryHandler};
pub use server::Runtime;
