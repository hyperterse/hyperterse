//! Database adapter configuration

use hyperterse_types::Connector;
use serde::{Deserialize, Serialize};

/// Database adapter configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Adapter {
    /// Unique name for this adapter (used in query references)
    pub name: String,

    /// Database connector type
    pub connector: Connector,

    /// Connection URL (supports environment variable substitution)
    pub url: String,
}

impl Adapter {
    /// Create a new adapter with the given name, connector, and URL
    pub fn new(name: impl Into<String>, connector: Connector, url: impl Into<String>) -> Self {
        Self {
            name: name.into(),
            connector,
            url: url.into(),
        }
    }

    /// Check if the URL contains environment variable placeholders
    pub fn has_env_placeholders(&self) -> bool {
        self.url.contains("{{") && self.url.contains("}}")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_adapter_new() {
        let adapter = Adapter::new("main-db", Connector::Postgres, "postgres://localhost/test");
        assert_eq!(adapter.name, "main-db");
        assert_eq!(adapter.connector, Connector::Postgres);
        assert_eq!(adapter.url, "postgres://localhost/test");
    }

    #[test]
    fn test_adapter_env_placeholders() {
        let adapter = Adapter::new("db", Connector::Postgres, "{{ env.DATABASE_URL }}");
        assert!(adapter.has_env_placeholders());

        let adapter2 = Adapter::new("db", Connector::Postgres, "postgres://localhost/test");
        assert!(!adapter2.has_env_placeholders());
    }

    #[test]
    fn test_adapter_serde() {
        let adapter = Adapter::new("main-db", Connector::Postgres, "postgres://localhost/test");
        let json = serde_json::to_string(&adapter).unwrap();
        assert!(json.contains("\"name\":\"main-db\""));
        assert!(json.contains("\"connector\":\"postgres\""));

        let parsed: Adapter = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.name, adapter.name);
        assert_eq!(parsed.connector, adapter.connector);
    }
}
