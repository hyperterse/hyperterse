//! Query and input definitions

use hyperterse_types::Primitive;
use serde::{Deserialize, Serialize};

/// Query definition
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Query {
    /// Unique name for this query (used as endpoint path)
    pub name: String,

    /// Name of the adapter to use for this query
    pub adapter: String,

    /// Query statement (SQL, Redis command, or MongoDB JSON)
    pub statement: String,

    /// Human-readable description of this query
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,

    /// Input parameters for this query
    #[serde(default)]
    pub inputs: Vec<Input>,
}

/// Input parameter definition for a query
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Input {
    /// Name of the input parameter
    pub name: String,

    /// Type of the input parameter
    #[serde(rename = "type")]
    pub primitive_type: Primitive,

    /// Whether this input is required (default: true)
    #[serde(default = "default_required")]
    pub required: bool,

    /// Default value for optional inputs
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default: Option<serde_json::Value>,

    /// Human-readable description of this input
    #[serde(skip_serializing_if = "Option::is_none")]
    pub description: Option<String>,
}

fn default_required() -> bool {
    true
}

impl Query {
    /// Create a new query with the given name, adapter, and statement
    pub fn new(
        name: impl Into<String>,
        adapter: impl Into<String>,
        statement: impl Into<String>,
    ) -> Self {
        Self {
            name: name.into(),
            adapter: adapter.into(),
            statement: statement.into(),
            description: None,
            inputs: Vec::new(),
        }
    }

    /// Set the description for this query
    pub fn with_description(mut self, description: impl Into<String>) -> Self {
        self.description = Some(description.into());
        self
    }

    /// Add an input parameter to this query
    pub fn with_input(mut self, input: Input) -> Self {
        self.inputs.push(input);
        self
    }

    /// Find an input by name
    pub fn find_input(&self, name: &str) -> Option<&Input> {
        self.inputs.iter().find(|i| i.name == name)
    }

    /// Get all required inputs
    pub fn required_inputs(&self) -> impl Iterator<Item = &Input> {
        self.inputs.iter().filter(|i| i.required)
    }

    /// Get all optional inputs
    pub fn optional_inputs(&self) -> impl Iterator<Item = &Input> {
        self.inputs.iter().filter(|i| !i.required)
    }

    /// Check if the statement contains template placeholders
    pub fn has_placeholders(&self) -> bool {
        self.statement.contains("{{") && self.statement.contains("}}")
    }
}

impl Input {
    /// Create a new required input with the given name and type
    pub fn new(name: impl Into<String>, primitive_type: Primitive) -> Self {
        Self {
            name: name.into(),
            primitive_type,
            required: true,
            default: None,
            description: None,
        }
    }

    /// Create a new optional input with the given name, type, and default value
    pub fn optional(
        name: impl Into<String>,
        primitive_type: Primitive,
        default: serde_json::Value,
    ) -> Self {
        Self {
            name: name.into(),
            primitive_type,
            required: false,
            default: Some(default),
            description: None,
        }
    }

    /// Set the description for this input
    pub fn with_description(mut self, description: impl Into<String>) -> Self {
        self.description = Some(description.into());
        self
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_query_new() {
        let query = Query::new("get-users", "main-db", "SELECT * FROM users");
        assert_eq!(query.name, "get-users");
        assert_eq!(query.adapter, "main-db");
        assert_eq!(query.statement, "SELECT * FROM users");
        assert!(query.inputs.is_empty());
    }

    #[test]
    fn test_query_with_inputs() {
        let query = Query::new(
            "get-user",
            "main-db",
            "SELECT * FROM users WHERE id = {{ inputs.id }}",
        )
        .with_description("Get a user by ID")
        .with_input(Input::new("id", Primitive::Int));

        assert!(query.description.is_some());
        assert_eq!(query.inputs.len(), 1);
        assert!(query.has_placeholders());
    }

    #[test]
    fn test_input_required() {
        let input = Input::new("id", Primitive::Int);
        assert!(input.required);
        assert!(input.default.is_none());
    }

    #[test]
    fn test_input_optional() {
        let input = Input::optional("limit", Primitive::Int, json!(10));
        assert!(!input.required);
        assert_eq!(input.default, Some(json!(10)));
    }

    #[test]
    fn test_query_find_input() {
        let query = Query::new("test", "db", "SELECT 1")
            .with_input(Input::new("id", Primitive::Int))
            .with_input(Input::optional("limit", Primitive::Int, json!(10)));

        assert!(query.find_input("id").is_some());
        assert!(query.find_input("limit").is_some());
        assert!(query.find_input("unknown").is_none());
    }
}
