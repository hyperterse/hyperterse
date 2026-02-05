//! Configuration validation

use hyperterse_core::{HyperterseError, Model};
use once_cell::sync::Lazy;
use regex::Regex;
use std::collections::HashSet;

/// Regex pattern for valid names (lower-kebab-case or lower_snake_case)
static NAME_PATTERN: Lazy<Regex> = Lazy::new(|| {
    Regex::new(r"^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$").unwrap()
});

/// Regex pattern for input placeholders: {{ inputs.fieldName }}
static INPUT_PATTERN: Lazy<Regex> = Lazy::new(|| {
    Regex::new(r"\{\{\s*inputs\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}").unwrap()
});

/// Configuration validator
pub struct ConfigValidator {
    /// Whether to validate names strictly
    strict_names: bool,
}

impl ConfigValidator {
    /// Create a new validator with default settings
    pub fn new() -> Self {
        Self { strict_names: true }
    }

    /// Create a validator with lenient name checking
    pub fn lenient() -> Self {
        Self { strict_names: false }
    }

    /// Validate the entire model configuration
    pub fn validate(&self, model: &Model) -> Result<(), HyperterseError> {
        self.validate_model_name(&model.name)?;
        self.validate_adapters(model)?;
        self.validate_queries(model)?;
        self.validate_adapter_references(model)?;
        self.validate_input_references(model)?;
        Ok(())
    }

    /// Validate the model name
    fn validate_model_name(&self, name: &str) -> Result<(), HyperterseError> {
        if name.is_empty() {
            return Err(HyperterseError::Validation(
                "Model name cannot be empty".to_string(),
            ));
        }

        if self.strict_names && !NAME_PATTERN.is_match(name) {
            return Err(HyperterseError::Validation(format!(
                "Invalid model name '{}': must be lower-kebab-case or lower_snake_case",
                name
            )));
        }

        Ok(())
    }

    /// Validate adapters configuration
    fn validate_adapters(&self, model: &Model) -> Result<(), HyperterseError> {
        let mut adapter_names = HashSet::new();

        for adapter in &model.adapters {
            // Check for empty name
            if adapter.name.is_empty() {
                return Err(HyperterseError::Validation(
                    "Adapter name cannot be empty".to_string(),
                ));
            }

            // Check name format
            if self.strict_names && !NAME_PATTERN.is_match(&adapter.name) {
                return Err(HyperterseError::Validation(format!(
                    "Invalid adapter name '{}': must be lower-kebab-case or lower_snake_case",
                    adapter.name
                )));
            }

            // Check for duplicate names
            if !adapter_names.insert(&adapter.name) {
                return Err(HyperterseError::Validation(format!(
                    "Duplicate adapter name: '{}'",
                    adapter.name
                )));
            }

            // Check for empty URL
            if adapter.url.is_empty() {
                return Err(HyperterseError::Validation(format!(
                    "Adapter '{}' has an empty URL",
                    adapter.name
                )));
            }
        }

        Ok(())
    }

    /// Validate queries configuration
    fn validate_queries(&self, model: &Model) -> Result<(), HyperterseError> {
        let mut query_names = HashSet::new();

        for query in &model.queries {
            // Check for empty name
            if query.name.is_empty() {
                return Err(HyperterseError::Validation(
                    "Query name cannot be empty".to_string(),
                ));
            }

            // Check name format
            if self.strict_names && !NAME_PATTERN.is_match(&query.name) {
                return Err(HyperterseError::Validation(format!(
                    "Invalid query name '{}': must be lower-kebab-case or lower_snake_case",
                    query.name
                )));
            }

            // Check for duplicate names
            if !query_names.insert(&query.name) {
                return Err(HyperterseError::Validation(format!(
                    "Duplicate query name: '{}'",
                    query.name
                )));
            }

            // Check for empty adapter reference
            if query.adapter.is_empty() {
                return Err(HyperterseError::Validation(format!(
                    "Query '{}' has no adapter specified",
                    query.name
                )));
            }

            // Check for empty statement
            if query.statement.is_empty() {
                return Err(HyperterseError::Validation(format!(
                    "Query '{}' has an empty statement",
                    query.name
                )));
            }

            // Validate inputs
            self.validate_query_inputs(query)?;
        }

        Ok(())
    }

    /// Validate query inputs
    fn validate_query_inputs(
        &self,
        query: &hyperterse_core::Query,
    ) -> Result<(), HyperterseError> {
        let mut input_names = HashSet::new();

        for input in &query.inputs {
            // Check for empty name
            if input.name.is_empty() {
                return Err(HyperterseError::Validation(format!(
                    "Query '{}' has an input with empty name",
                    query.name
                )));
            }

            // Check for duplicate input names
            if !input_names.insert(&input.name) {
                return Err(HyperterseError::Validation(format!(
                    "Query '{}' has duplicate input: '{}'",
                    query.name, input.name
                )));
            }

            // Check that optional inputs have defaults
            if !input.required && input.default.is_none() {
                return Err(HyperterseError::Validation(format!(
                    "Query '{}': optional input '{}' must have a default value",
                    query.name, input.name
                )));
            }

            // Validate default value type if present
            if let Some(default) = &input.default {
                if !input.primitive_type.validate(default) {
                    return Err(HyperterseError::Validation(format!(
                        "Query '{}': default value for '{}' has invalid type (expected {})",
                        query.name, input.name, input.primitive_type
                    )));
                }
            }
        }

        Ok(())
    }

    /// Validate that all adapter references in queries exist
    fn validate_adapter_references(&self, model: &Model) -> Result<(), HyperterseError> {
        let adapter_names: HashSet<&str> = model.adapters.iter().map(|a| a.name.as_str()).collect();

        for query in &model.queries {
            if !adapter_names.contains(query.adapter.as_str()) {
                return Err(HyperterseError::Validation(format!(
                    "Query '{}' references non-existent adapter: '{}'",
                    query.name, query.adapter
                )));
            }
        }

        Ok(())
    }

    /// Validate that all input placeholders in statements have corresponding input definitions
    fn validate_input_references(&self, model: &Model) -> Result<(), HyperterseError> {
        for query in &model.queries {
            let defined_inputs: HashSet<&str> =
                query.inputs.iter().map(|i| i.name.as_str()).collect();

            let referenced_inputs: Vec<String> = INPUT_PATTERN
                .captures_iter(&query.statement)
                .map(|cap| cap.get(1).unwrap().as_str().to_string())
                .collect();

            for input_name in referenced_inputs {
                if !defined_inputs.contains(input_name.as_str()) {
                    return Err(HyperterseError::Validation(format!(
                        "Query '{}' uses undefined input: '{{ inputs.{} }}'",
                        query.name, input_name
                    )));
                }
            }
        }

        Ok(())
    }
}

impl Default for ConfigValidator {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use hyperterse_core::{Adapter, Input, Query};
    use hyperterse_types::{Connector, Primitive};

    fn create_model_with_adapter() -> Model {
        Model {
            name: "test-api".to_string(),
            adapters: vec![Adapter::new("main-db", Connector::Postgres, "postgres://localhost/test")],
            queries: vec![],
            server: None,
            export: None,
        }
    }

    #[test]
    fn test_valid_model() {
        let mut model = create_model_with_adapter();
        model.queries.push(
            Query::new("get-users", "main-db", "SELECT * FROM users")
                .with_input(Input::new("id", Primitive::Int)),
        );

        let validator = ConfigValidator::new();
        assert!(validator.validate(&model).is_ok());
    }

    #[test]
    fn test_invalid_model_name() {
        let model = Model {
            name: "Invalid Name".to_string(),
            adapters: vec![],
            queries: vec![],
            server: None,
            export: None,
        };

        let validator = ConfigValidator::new();
        assert!(validator.validate(&model).is_err());
    }

    #[test]
    fn test_duplicate_adapter_names() {
        let model = Model {
            name: "test-api".to_string(),
            adapters: vec![
                Adapter::new("main-db", Connector::Postgres, "url1"),
                Adapter::new("main-db", Connector::Mysql, "url2"),
            ],
            queries: vec![],
            server: None,
            export: None,
        };

        let validator = ConfigValidator::new();
        assert!(validator.validate(&model).is_err());
    }

    #[test]
    fn test_missing_adapter_reference() {
        let model = Model {
            name: "test-api".to_string(),
            adapters: vec![Adapter::new("main-db", Connector::Postgres, "url")],
            queries: vec![Query::new("test", "other-db", "SELECT 1")],
            server: None,
            export: None,
        };

        let validator = ConfigValidator::new();
        let result = validator.validate(&model);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("non-existent adapter"));
    }

    #[test]
    fn test_undefined_input_reference() {
        let model = Model {
            name: "test-api".to_string(),
            adapters: vec![Adapter::new("main-db", Connector::Postgres, "url")],
            queries: vec![Query::new(
                "test",
                "main-db",
                "SELECT * FROM users WHERE id = {{ inputs.id }}",
            )],
            server: None,
            export: None,
        };

        let validator = ConfigValidator::new();
        let result = validator.validate(&model);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("undefined input"));
    }

    #[test]
    fn test_optional_input_without_default() {
        let mut model = create_model_with_adapter();
        let mut input = Input::new("limit", Primitive::Int);
        input.required = false;
        // No default value set

        model.queries.push(Query::new("test", "main-db", "SELECT 1").with_input(input));

        let validator = ConfigValidator::new();
        let result = validator.validate(&model);
        assert!(result.is_err());
        assert!(result.unwrap_err().to_string().contains("default value"));
    }
}
