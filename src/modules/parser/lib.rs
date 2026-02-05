//! Configuration parsing for Hyperterse
//!
//! This crate handles parsing of YAML (.terse) configuration files,
//! validation, and environment variable substitution.

pub mod env;
pub mod validator;
pub mod yaml;

pub use validator::ConfigValidator;
pub use yaml::YamlParser;

use hyperterse_core::{HyperterseError, Model};

/// Parse a configuration file from a path
pub fn parse_file(path: &str) -> Result<Model, HyperterseError> {
    let content = std::fs::read_to_string(path)
        .map_err(|e| HyperterseError::Config(format!("Failed to read file '{}': {}", path, e)))?;

    parse_string(&content)
}

/// Parse a configuration from a string
pub fn parse_string(content: &str) -> Result<Model, HyperterseError> {
    // Parse YAML
    let model = YamlParser::parse(content)?;

    // Validate configuration
    let validator = ConfigValidator::new();
    validator.validate(&model)?;

    Ok(model)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_simple_config() {
        // README/`.terse` schema (map-based)
        let yaml = r#"
name: test-api
adapters:
  main-db:
    connector: postgres
    connection_string: "postgres://localhost/test"
queries:
  get-users:
    use: main-db
    statement: "SELECT * FROM users"
"#;
        let model = parse_string(yaml).unwrap();
        assert_eq!(model.name, "test-api");
        assert_eq!(model.adapters.len(), 1);
        assert_eq!(model.queries.len(), 1);
    }
}
