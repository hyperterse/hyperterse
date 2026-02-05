//! Connector trait definition

use async_trait::async_trait;
use hyperterse_core::HyperterseError;
use std::collections::HashMap;

/// Result type for connector operations
pub type ConnectorResult = Vec<HashMap<String, serde_json::Value>>;

/// Trait for database connectors
///
/// All connectors implement this trait to provide a unified interface
/// for query execution across different database types.
#[async_trait]
pub trait Connector: Send + Sync {
    /// Execute a query/command and return the results
    ///
    /// # Arguments
    /// * `statement` - The query/command to execute (SQL, Redis command, or MongoDB JSON)
    /// * `params` - Parameters to substitute into the statement
    ///
    /// # Returns
    /// A vector of rows, where each row is a map of column names to values
    async fn execute(
        &self,
        statement: &str,
        params: &HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError>;

    /// Close the connection and release resources
    async fn close(&self) -> Result<(), HyperterseError>;

    /// Check if the connection is healthy
    async fn health_check(&self) -> Result<(), HyperterseError>;

    /// Get the connector type name
    fn connector_type(&self) -> &'static str;
}
