//! Query execution module
//!
//! Handles input validation, template substitution, and query execution.

mod substitutor;
mod validator;

pub use substitutor::TemplateSubstitutor;
pub use validator::InputValidator;

use hyperterse_core::{HyperterseError, Model, Query};
use std::collections::HashMap;
use std::sync::Arc;

use crate::connectors::{ConnectorManager, ConnectorResult};

/// Query executor that orchestrates validation, substitution, and execution
pub struct QueryExecutor {
    model: Arc<Model>,
    connectors: Arc<ConnectorManager>,
    validator: InputValidator,
    substitutor: TemplateSubstitutor,
}

impl QueryExecutor {
    /// Create a new query executor
    pub fn new(model: Arc<Model>, connectors: Arc<ConnectorManager>) -> Self {
        Self {
            model,
            connectors,
            validator: InputValidator::new(),
            substitutor: TemplateSubstitutor::new(),
        }
    }

    /// Execute a query by name with the given inputs
    pub async fn execute(
        &self,
        query_name: &str,
        inputs: HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError> {
        // Find the query
        let query = self
            .model
            .find_query(query_name)
            .ok_or_else(|| HyperterseError::QueryNotFound(query_name.to_string()))?;

        // Validate inputs
        let validated_inputs = self.validator.validate(query, inputs)?;

        // Get the connector
        let connector = self.connectors.get(&query.adapter).await?;

        // Get the connector type for proper escaping
        let adapter = self
            .model
            .find_adapter(&query.adapter)
            .ok_or_else(|| HyperterseError::AdapterNotFound(query.adapter.clone()))?;

        // Substitute template variables
        let statement =
            self.substitutor
                .substitute(&query.statement, &validated_inputs, adapter.connector)?;

        // Execute the query
        connector.execute(&statement, &validated_inputs).await
    }

    /// Get all available query names
    pub fn query_names(&self) -> Vec<&str> {
        self.model.queries.iter().map(|q| q.name.as_str()).collect()
    }

    /// Get a query by name
    pub fn get_query(&self, name: &str) -> Option<&Query> {
        self.model.find_query(name)
    }

    /// Get the underlying model
    pub fn model(&self) -> &Model {
        &self.model
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use hyperterse_core::{Adapter, Input};
    use hyperterse_types::{Connector as ConnectorType, Primitive};

    fn create_test_model() -> Model {
        Model {
            name: "test".to_string(),
            adapters: vec![Adapter::new(
                "db",
                ConnectorType::Postgres,
                "postgres://localhost/test",
            )],
            queries: vec![Query::new(
                "get-user",
                "db",
                "SELECT * FROM users WHERE id = {{ inputs.id }}",
            )
            .with_input(Input::new("id", Primitive::Int))],
            server: None,
            export: None,
        }
    }

    #[test]
    fn test_query_names() {
        let model = Arc::new(create_test_model());
        let connectors = Arc::new(ConnectorManager::new());
        let executor = QueryExecutor::new(model, connectors);

        let names = executor.query_names();
        assert_eq!(names, vec!["get-user"]);
    }

    #[test]
    fn test_get_query() {
        let model = Arc::new(create_test_model());
        let connectors = Arc::new(ConnectorManager::new());
        let executor = QueryExecutor::new(model, connectors);

        assert!(executor.get_query("get-user").is_some());
        assert!(executor.get_query("nonexistent").is_none());
    }
}
