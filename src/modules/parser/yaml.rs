//! YAML configuration parser

use hyperterse_core::{Adapter, ExportConfig, HyperterseError, Input, Model, Query, ServerConfig};
use hyperterse_types::{Connector, Primitive};
use serde::Deserialize;
use std::collections::HashMap;

use crate::env::EnvSubstitutor;

/// YAML parser for Hyperterse configuration files
pub struct YamlParser;

/// README/`.terse`-style config schema (map-based adapters/queries).
///
/// This is intentionally permissive so older configs keep working.
#[derive(Debug, Deserialize)]
struct TerseConfig {
    name: String,

    #[serde(default)]
    adapters: HashMap<String, TerseAdapter>,

    #[serde(default)]
    queries: HashMap<String, TerseQuery>,

    #[serde(default)]
    server: Option<TerseServer>,

    #[serde(default)]
    export: Option<TerseExport>,
}

#[derive(Debug, Deserialize)]
struct TerseAdapter {
    connector: Connector,

    /// Legacy field name used in README examples.
    #[serde(default)]
    connection_string: Option<String>,

    /// Newer/alternate field name.
    #[serde(default)]
    url: Option<String>,

    /// Adapter-specific options (ignored by core model for now).
    #[serde(default)]
    _options: Option<serde_yaml::Value>,
}

#[derive(Debug, Deserialize)]
struct TerseQuery {
    /// Legacy field name used in README examples.
    #[serde(rename = "use")]
    adapter_use: Option<String>,

    /// Alternate field name.
    #[serde(default)]
    adapter: Option<String>,

    statement: String,

    #[serde(default)]
    description: Option<String>,

    #[serde(default)]
    inputs: HashMap<String, TerseInput>,
}

#[derive(Debug, Deserialize)]
struct TerseInput {
    #[serde(rename = "type")]
    primitive_type: Primitive,

    #[serde(default)]
    description: Option<String>,

    /// README uses `optional: true/false`
    #[serde(default)]
    optional: Option<bool>,

    /// Allow `required: true/false` as well (wins over `optional` if set).
    #[serde(default)]
    required: Option<bool>,

    #[serde(default)]
    default: Option<serde_yaml::Value>,
}

#[derive(Debug, Deserialize)]
struct TerseServer {
    #[serde(default)]
    port: Option<serde_yaml::Value>,

    #[serde(default)]
    log_level: Option<u8>,
}

#[derive(Debug, Deserialize)]
struct TerseExport {
    /// Legacy field name used in some configs.
    #[serde(default)]
    out: Option<String>,

    /// Newer/alternate field name.
    #[serde(default)]
    output_dir: Option<String>,

    #[serde(default)]
    base_url: Option<String>,
}

impl YamlParser {
    /// Parse a YAML string into a Model
    pub fn parse(content: &str) -> Result<Model, HyperterseError> {
        // First, substitute environment variables in the YAML content
        let substitutor = EnvSubstitutor::new();
        let substituted = substitutor.substitute(content)?;

        // Try the current (vector-based) schema first.
        if let Ok(model) = serde_yaml::from_str::<Model>(&substituted) {
            return Ok(model);
        }

        // Fallback to README/`.terse` map-based schema.
        let terse = serde_yaml::from_str::<TerseConfig>(&substituted)
            .map_err(|e| HyperterseError::Config(format!("YAML parse error: {}", e)))?;
        terse_to_model(terse)
    }

    /// Parse a YAML string without environment variable substitution
    pub fn parse_raw(content: &str) -> Result<Model, HyperterseError> {
        if let Ok(model) = serde_yaml::from_str::<Model>(content) {
            return Ok(model);
        }

        let terse = serde_yaml::from_str::<TerseConfig>(content)
            .map_err(|e| HyperterseError::Config(format!("YAML parse error: {}", e)))?;
        terse_to_model(terse)
    }
}

fn terse_to_model(cfg: TerseConfig) -> Result<Model, HyperterseError> {
    let mut adapters: Vec<Adapter> = Vec::with_capacity(cfg.adapters.len());
    for (name, adapter) in cfg.adapters {
        let url = adapter.connection_string.or(adapter.url).ok_or_else(|| {
            HyperterseError::Config(format!(
                "Adapter '{}' is missing 'connection_string' (or 'url')",
                name
            ))
        })?;

        adapters.push(Adapter::new(name, adapter.connector, url));
    }

    let mut queries: Vec<Query> = Vec::with_capacity(cfg.queries.len());
    for (name, query) in cfg.queries {
        let adapter_name = query.adapter_use.or(query.adapter).ok_or_else(|| {
            HyperterseError::Config(format!("Query '{}' is missing 'use' (or 'adapter')", name))
        })?;

        let mut inputs: Vec<Input> = Vec::with_capacity(query.inputs.len());
        for (input_name, input) in query.inputs {
            let required = input
                .required
                .unwrap_or_else(|| !input.optional.unwrap_or(false));

            let default = match input.default {
                None => None,
                Some(v) => Some(serde_json::to_value(v).map_err(|e| {
                    HyperterseError::Config(format!(
                        "Failed to parse default for input '{}.{}': {}",
                        name, input_name, e
                    ))
                })?),
            };

            inputs.push(Input {
                name: input_name,
                primitive_type: input.primitive_type,
                required,
                default,
                description: input.description,
            });
        }

        queries.push(Query {
            name,
            adapter: adapter_name,
            statement: query.statement,
            description: query.description,
            inputs,
        });
    }

    let server = cfg.server.map(|s| ServerConfig {
        port: s.port.and_then(yaml_scalar_to_string),
        log_level: s.log_level,
        pool: None,
    });

    let export = cfg.export.map(|e| ExportConfig {
        base_url: e.base_url,
        output_dir: e.output_dir.or(e.out),
    });

    Ok(Model {
        name: cfg.name,
        adapters,
        queries,
        server,
        export,
    })
}

fn yaml_scalar_to_string(value: serde_yaml::Value) -> Option<String> {
    match value {
        serde_yaml::Value::Null => None,
        serde_yaml::Value::Bool(b) => Some(b.to_string()),
        serde_yaml::Value::Number(n) => Some(n.to_string()),
        serde_yaml::Value::String(s) => Some(s),
        // For non-scalars (seq/map), just serialize them.
        other => serde_yaml::to_string(&other)
            .ok()
            .map(|s| s.trim().to_string()),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use hyperterse_types::Connector;

    #[test]
    fn test_parse_minimal_config() {
        let yaml = r#"
name: minimal-api
adapters: []
queries: []
"#;
        let model = YamlParser::parse(yaml).unwrap();
        assert_eq!(model.name, "minimal-api");
        assert!(model.adapters.is_empty());
        assert!(model.queries.is_empty());
    }

    #[test]
    fn test_parse_full_config() {
        // README/`.terse` schema (map-based)
        let yaml = r#"
name: full-api
adapters:
  postgres-db:
    connector: postgres
    connection_string: "postgres://localhost:5432/mydb"
  redis-cache:
    connector: redis
    connection_string: "redis://localhost:6379"
queries:
  get-users:
    use: postgres-db
    statement: "SELECT * FROM users LIMIT {{ inputs.limit }}"
    description: "Get all users"
    inputs:
      limit:
        type: int
        optional: true
        default: 10
server:
  port: 3000
  log_level: 2
"#;
        let model = YamlParser::parse(yaml).unwrap();
        assert_eq!(model.name, "full-api");
        assert_eq!(model.adapters.len(), 2);
        assert_eq!(model.adapters[0].connector, Connector::Postgres);
        assert_eq!(model.adapters[1].connector, Connector::Redis);
        assert_eq!(model.queries.len(), 1);
        assert_eq!(model.queries[0].inputs.len(), 1);
        assert!(!model.queries[0].inputs[0].required);
        assert_eq!(
            model.server.as_ref().unwrap().port,
            Some("3000".to_string())
        );
    }

    #[test]
    fn test_parse_mongodb_config() {
        let yaml = r#"
name: mongodb-api
adapters:
  mongo-db:
    connector: mongodb
    connection_string: "mongodb://localhost:27017/mydb"
queries:
  find-users:
    use: mongo-db
    statement: |
      {
        "database": "mydb",
        "collection": "users",
        "operation": "find",
        "filter": { "status": "active" }
      }
"#;
        let model = YamlParser::parse(yaml).unwrap();
        assert_eq!(model.adapters[0].connector, Connector::Mongodb);
        assert!(model.queries[0]
            .statement
            .contains("\"operation\": \"find\""));
    }

    #[test]
    fn test_parse_invalid_yaml() {
        let yaml = "invalid: yaml: content: [";
        let result = YamlParser::parse(yaml);
        assert!(result.is_err());
    }
}
