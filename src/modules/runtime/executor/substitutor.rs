//! Template substitution for query statements

use hyperterse_core::HyperterseError;
use hyperterse_types::Connector;
use once_cell::sync::Lazy;
use regex::Regex;
use std::collections::HashMap;

/// Regex pattern for input placeholders: {{ inputs.fieldName }}
static INPUT_PATTERN: Lazy<Regex> =
    Lazy::new(|| Regex::new(r"\{\{\s*inputs\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}").unwrap());

/// Regex pattern for quoted input placeholders in MongoDB JSON: "{{ inputs.fieldName }}"
/// This captures the surrounding quotes so they can be replaced together,
/// preventing double-quoting when the value is serialized as JSON.
static QUOTED_INPUT_PATTERN: Lazy<Regex> =
    Lazy::new(|| Regex::new(r#""(\{\{\s*inputs\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\})""#).unwrap());

/// Regex pattern for environment variable placeholders: {{ env.VAR_NAME }}
static ENV_PATTERN: Lazy<Regex> =
    Lazy::new(|| Regex::new(r"\{\{\s*env\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}").unwrap());

/// Template substitutor for query statements
pub struct TemplateSubstitutor;

impl TemplateSubstitutor {
    /// Create a new template substitutor
    pub fn new() -> Self {
        Self
    }

    /// Substitute placeholders in a statement
    ///
    /// Handles both `{{ inputs.fieldName }}` and `{{ env.VAR_NAME }}` patterns.
    /// Values are escaped based on the connector type to prevent injection.
    pub fn substitute(
        &self,
        statement: &str,
        inputs: &HashMap<String, serde_json::Value>,
        connector: Connector,
    ) -> Result<String, HyperterseError> {
        let mut result = statement.to_string();

        // Substitute environment variables first
        result = self.substitute_env_vars(&result)?;

        // Substitute input values
        result = self.substitute_inputs(&result, inputs, connector)?;

        Ok(result)
    }

    /// Substitute environment variable placeholders
    fn substitute_env_vars(&self, statement: &str) -> Result<String, HyperterseError> {
        let mut result = statement.to_string();

        for cap in ENV_PATTERN.captures_iter(statement) {
            let full_match = cap.get(0).unwrap().as_str();
            let var_name = cap.get(1).unwrap().as_str();

            let value = std::env::var(var_name)
                .map_err(|_| HyperterseError::EnvVarNotFound(var_name.to_string()))?;

            result = result.replace(full_match, &value);
        }

        Ok(result)
    }

    /// Substitute input placeholders
    fn substitute_inputs(
        &self,
        statement: &str,
        inputs: &HashMap<String, serde_json::Value>,
        connector: Connector,
    ) -> Result<String, HyperterseError> {
        let mut result = statement.to_string();

        for cap in INPUT_PATTERN.captures_iter(statement) {
            let full_match = cap.get(0).unwrap().as_str();
            let input_name = cap.get(1).unwrap().as_str();

            let value = inputs
                .get(input_name)
                .ok_or_else(|| HyperterseError::MissingInput(input_name.to_string()))?;

            let escaped = self.escape_value(value, connector)?;
            result = result.replace(full_match, &escaped);
        }

        Ok(result)
    }

    /// Escape a value based on the connector type
    fn escape_value(
        &self,
        value: &serde_json::Value,
        connector: Connector,
    ) -> Result<String, HyperterseError> {
        match connector {
            Connector::Postgres | Connector::Mysql => self.escape_sql(value),
            Connector::Redis => self.escape_redis(value),
            Connector::Mongodb => self.escape_mongodb(value),
        }
    }

    /// Escape a value for SQL (PostgreSQL, MySQL)
    fn escape_sql(&self, value: &serde_json::Value) -> Result<String, HyperterseError> {
        match value {
            serde_json::Value::Null => Ok("NULL".to_string()),
            serde_json::Value::Bool(b) => Ok(if *b { "TRUE" } else { "FALSE" }.to_string()),
            serde_json::Value::Number(n) => Ok(n.to_string()),
            serde_json::Value::String(s) => {
                // Escape single quotes by doubling them
                let escaped = s.replace('\'', "''");
                Ok(format!("'{}'", escaped))
            }
            serde_json::Value::Array(arr) => {
                // Convert array to SQL array syntax
                let elements: Result<Vec<String>, _> =
                    arr.iter().map(|v| self.escape_sql(v)).collect();
                Ok(format!("({})", elements?.join(", ")))
            }
            serde_json::Value::Object(_) => {
                // Convert object to JSON string
                let json_str = serde_json::to_string(value).map_err(|e| {
                    HyperterseError::Template(format!("JSON serialization failed: {}", e))
                })?;
                let escaped = json_str.replace('\'', "''");
                Ok(format!("'{}'", escaped))
            }
        }
    }

    /// Escape a value for Redis commands
    fn escape_redis(&self, value: &serde_json::Value) -> Result<String, HyperterseError> {
        match value {
            serde_json::Value::Null => Ok("".to_string()),
            serde_json::Value::Bool(b) => Ok(if *b { "1" } else { "0" }.to_string()),
            serde_json::Value::Number(n) => Ok(n.to_string()),
            serde_json::Value::String(s) => {
                // Escape spaces and special characters
                if s.contains(' ') || s.contains('"') || s.contains('\'') {
                    let escaped = s.replace('\\', "\\\\").replace('"', "\\\"");
                    Ok(format!("\"{}\"", escaped))
                } else {
                    Ok(s.clone())
                }
            }
            serde_json::Value::Array(_) | serde_json::Value::Object(_) => {
                let json_str = serde_json::to_string(value).map_err(|e| {
                    HyperterseError::Template(format!("JSON serialization failed: {}", e))
                })?;
                Ok(format!("\"{}\"", json_str.replace('"', "\\\"")))
            }
        }
    }

    /// Escape a value for MongoDB JSON statements
    fn escape_mongodb(&self, value: &serde_json::Value) -> Result<String, HyperterseError> {
        // For MongoDB, we need to output valid JSON
        serde_json::to_string(value)
            .map_err(|e| HyperterseError::Template(format!("JSON serialization failed: {}", e)))
    }
}

impl Default for TemplateSubstitutor {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_substitute_sql() {
        let substitutor = TemplateSubstitutor::new();
        let mut inputs = HashMap::new();
        inputs.insert("id".to_string(), json!(42));
        inputs.insert("name".to_string(), json!("John's"));

        let statement =
            "SELECT * FROM users WHERE id = {{ inputs.id }} AND name = {{ inputs.name }}";
        let result = substitutor
            .substitute(statement, &inputs, Connector::Postgres)
            .unwrap();

        assert_eq!(
            result,
            "SELECT * FROM users WHERE id = 42 AND name = 'John''s'"
        );
    }

    #[test]
    fn test_substitute_mongodb() {
        let substitutor = TemplateSubstitutor::new();
        let mut inputs = HashMap::new();
        inputs.insert("status".to_string(), json!("active"));
        inputs.insert("limit".to_string(), json!(10));

        let statement = r#"{"collection": "users", "filter": {"status": {{ inputs.status }}}, "options": {"limit": {{ inputs.limit }}}}"#;
        let result = substitutor
            .substitute(statement, &inputs, Connector::Mongodb)
            .unwrap();

        assert!(result.contains("\"status\": \"active\""));
        assert!(result.contains("\"limit\": 10"));
    }

    #[test]
    fn test_substitute_redis() {
        let substitutor = TemplateSubstitutor::new();
        let mut inputs = HashMap::new();
        inputs.insert("key".to_string(), json!("user:123"));
        inputs.insert("value".to_string(), json!("hello world"));

        let statement = "SET {{ inputs.key }} {{ inputs.value }}";
        let result = substitutor
            .substitute(statement, &inputs, Connector::Redis)
            .unwrap();

        assert_eq!(result, "SET user:123 \"hello world\"");
    }

    #[test]
    fn test_sql_injection_prevention() {
        let substitutor = TemplateSubstitutor::new();
        let mut inputs = HashMap::new();
        inputs.insert("name".to_string(), json!("'; DROP TABLE users; --"));

        let statement = "SELECT * FROM users WHERE name = {{ inputs.name }}";
        let result = substitutor
            .substitute(statement, &inputs, Connector::Postgres)
            .unwrap();

        // The malicious input should be escaped - single quotes are doubled
        // The result should be: SELECT * FROM users WHERE name = '''; DROP TABLE users; --'
        // The outer quotes wrap the string, and the inner ' is escaped as ''
        assert!(result.contains("'''")); // Escaped single quote
        assert_eq!(
            result,
            "SELECT * FROM users WHERE name = '''; DROP TABLE users; --'"
        );
    }

    #[test]
    fn test_missing_input() {
        let substitutor = TemplateSubstitutor::new();
        let inputs = HashMap::new();

        let statement = "SELECT * FROM users WHERE id = {{ inputs.id }}";
        let result = substitutor.substitute(statement, &inputs, Connector::Postgres);

        assert!(result.is_err());
    }

    #[test]
    fn test_null_value() {
        let substitutor = TemplateSubstitutor::new();
        let mut inputs = HashMap::new();
        inputs.insert("value".to_string(), serde_json::Value::Null);

        let statement = "UPDATE users SET name = {{ inputs.value }}";
        let result = substitutor
            .substitute(statement, &inputs, Connector::Postgres)
            .unwrap();

        assert_eq!(result, "UPDATE users SET name = NULL");
    }

    #[test]
    fn test_boolean_value() {
        let substitutor = TemplateSubstitutor::new();
        let mut inputs = HashMap::new();
        inputs.insert("active".to_string(), json!(true));

        let statement = "UPDATE users SET active = {{ inputs.active }}";
        let result = substitutor
            .substitute(statement, &inputs, Connector::Postgres)
            .unwrap();

        assert_eq!(result, "UPDATE users SET active = TRUE");
    }
}
