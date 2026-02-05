//! Root model configuration

use serde::{Deserialize, Serialize};

use super::{Adapter, ExportConfig, Query, ServerConfig};

/// Root configuration model that represents a Hyperterse configuration file
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Model {
    /// Name of the model/API
    pub name: String,

    /// Database adapters configuration
    #[serde(default)]
    pub adapters: Vec<Adapter>,

    /// Query definitions
    #[serde(default)]
    pub queries: Vec<Query>,

    /// Server configuration (optional)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub server: Option<ServerConfig>,

    /// Export configuration (optional)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub export: Option<ExportConfig>,
}

impl Model {
    /// Create a new empty model with the given name
    pub fn new(name: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            adapters: Vec::new(),
            queries: Vec::new(),
            server: None,
            export: None,
        }
    }

    /// Find an adapter by name
    pub fn find_adapter(&self, name: &str) -> Option<&Adapter> {
        self.adapters.iter().find(|a| a.name == name)
    }

    /// Find a query by name
    pub fn find_query(&self, name: &str) -> Option<&Query> {
        self.queries.iter().find(|q| q.name == name)
    }

    /// Get the server port, defaulting to 8080
    pub fn port(&self) -> u16 {
        self.server
            .as_ref()
            .and_then(|s| s.port.as_ref())
            .and_then(|p| p.parse().ok())
            .unwrap_or(8080)
    }

    /// Get the log level, defaulting to 1 (INFO)
    pub fn log_level(&self) -> u8 {
        self.server
            .as_ref()
            .and_then(|s| s.log_level)
            .unwrap_or(1)
    }
}

impl Default for Model {
    fn default() -> Self {
        Self::new("hyperterse")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_model_new() {
        let model = Model::new("test-api");
        assert_eq!(model.name, "test-api");
        assert!(model.adapters.is_empty());
        assert!(model.queries.is_empty());
    }

    #[test]
    fn test_model_default_port() {
        let model = Model::new("test");
        assert_eq!(model.port(), 8080);
    }

    #[test]
    fn test_model_custom_port() {
        let model = Model {
            name: "test".to_string(),
            adapters: Vec::new(),
            queries: Vec::new(),
            server: Some(ServerConfig {
                port: Some("3000".to_string()),
                log_level: None,
                pool: None,
            }),
            export: None,
        };
        assert_eq!(model.port(), 3000);
    }
}
