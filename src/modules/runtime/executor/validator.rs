//! Input validation for query execution

use hyperterse_core::{HyperterseError, Query};
use std::collections::HashMap;

/// Input validator for query parameters
pub struct InputValidator;

impl InputValidator {
    /// Create a new input validator
    pub fn new() -> Self {
        Self
    }

    /// Validate inputs against a query's input definitions
    ///
    /// Returns the validated inputs with defaults applied for optional inputs.
    pub fn validate(
        &self,
        query: &Query,
        mut inputs: HashMap<String, serde_json::Value>,
    ) -> Result<HashMap<String, serde_json::Value>, HyperterseError> {
        // Check for missing required inputs and apply defaults
        for input_def in &query.inputs {
            match inputs.get(&input_def.name) {
                Some(value) => {
                    // Validate the type
                    if !input_def.primitive_type.validate(value) {
                        return Err(HyperterseError::InvalidInputType(
                            input_def.name.clone(),
                            input_def.primitive_type.to_string(),
                        ));
                    }
                }
                None => {
                    if input_def.required {
                        return Err(HyperterseError::MissingInput(input_def.name.clone()));
                    } else if let Some(default) = &input_def.default {
                        // Apply default value
                        inputs.insert(input_def.name.clone(), default.clone());
                    }
                }
            }
        }

        Ok(inputs)
    }
}

impl Default for InputValidator {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use hyperterse_core::Input;
    use hyperterse_types::Primitive;
    use serde_json::json;

    fn create_test_query() -> Query {
        Query::new("test", "db", "SELECT 1")
            .with_input(Input::new("id", Primitive::Int))
            .with_input(Input::optional("limit", Primitive::Int, json!(10)))
    }

    #[test]
    fn test_validate_valid_inputs() {
        let validator = InputValidator::new();
        let query = create_test_query();

        let mut inputs = HashMap::new();
        inputs.insert("id".to_string(), json!(42));

        let result = validator.validate(&query, inputs);
        assert!(result.is_ok());

        let validated = result.unwrap();
        assert_eq!(validated.get("id"), Some(&json!(42)));
        assert_eq!(validated.get("limit"), Some(&json!(10))); // Default applied
    }

    #[test]
    fn test_validate_missing_required() {
        let validator = InputValidator::new();
        let query = create_test_query();

        let inputs = HashMap::new(); // Missing 'id'

        let result = validator.validate(&query, inputs);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), HyperterseError::MissingInput(_)));
    }

    #[test]
    fn test_validate_wrong_type() {
        let validator = InputValidator::new();
        let query = create_test_query();

        let mut inputs = HashMap::new();
        inputs.insert("id".to_string(), json!("not an int"));

        let result = validator.validate(&query, inputs);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), HyperterseError::InvalidInputType(_, _)));
    }

    #[test]
    fn test_validate_optional_with_value() {
        let validator = InputValidator::new();
        let query = create_test_query();

        let mut inputs = HashMap::new();
        inputs.insert("id".to_string(), json!(1));
        inputs.insert("limit".to_string(), json!(50));

        let result = validator.validate(&query, inputs);
        assert!(result.is_ok());

        let validated = result.unwrap();
        assert_eq!(validated.get("limit"), Some(&json!(50))); // Value overrides default
    }
}
