//! Additional configuration types

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Connection pool configuration for database connectors
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PoolConfig {
    /// Maximum number of connections in the pool (default: 10)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_connections: Option<u32>,

    /// Minimum number of connections to maintain (default: 1)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub min_connections: Option<u32>,

    /// Connection acquire timeout in seconds (default: 30)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub acquire_timeout_secs: Option<u64>,

    /// Idle connection timeout in seconds (default: 600)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub idle_timeout_secs: Option<u64>,

    /// Maximum lifetime of a connection in seconds (default: 1800)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_lifetime_secs: Option<u64>,
}

impl Default for PoolConfig {
    fn default() -> Self {
        Self {
            max_connections: Some(10),
            min_connections: Some(1),
            acquire_timeout_secs: Some(30),
            idle_timeout_secs: Some(600),
            max_lifetime_secs: Some(1800),
        }
    }
}

impl PoolConfig {
    /// Get max connections with default fallback
    pub fn max_connections(&self) -> u32 {
        self.max_connections.unwrap_or(10)
    }

    /// Get min connections with default fallback
    pub fn min_connections(&self) -> u32 {
        self.min_connections.unwrap_or(1)
    }

    /// Get acquire timeout with default fallback
    pub fn acquire_timeout(&self) -> std::time::Duration {
        std::time::Duration::from_secs(self.acquire_timeout_secs.unwrap_or(30))
    }

    /// Get idle timeout with default fallback
    pub fn idle_timeout(&self) -> std::time::Duration {
        std::time::Duration::from_secs(self.idle_timeout_secs.unwrap_or(600))
    }

    /// Get max lifetime with default fallback
    pub fn max_lifetime(&self) -> std::time::Duration {
        std::time::Duration::from_secs(self.max_lifetime_secs.unwrap_or(1800))
    }
}

/// Server configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServerConfig {
    /// Port to listen on (default: 8080)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub port: Option<String>,

    /// Log level: 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR
    #[serde(skip_serializing_if = "Option::is_none")]
    pub log_level: Option<u8>,

    /// Connection pool configuration
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pool: Option<PoolConfig>,
}

impl Default for ServerConfig {
    fn default() -> Self {
        Self {
            port: Some("8080".to_string()),
            log_level: Some(1),
            pool: Some(PoolConfig::default()),
        }
    }
}

/// Export configuration for generating documentation and artifacts
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ExportConfig {
    /// Base URL for generated documentation
    #[serde(skip_serializing_if = "Option::is_none")]
    pub base_url: Option<String>,

    /// Output directory for generated files
    #[serde(skip_serializing_if = "Option::is_none")]
    pub output_dir: Option<String>,
}

/// Data object for query inputs (generic key-value map)
pub type Data = HashMap<String, serde_json::Value>;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_server_config_default() {
        let config = ServerConfig::default();
        assert_eq!(config.port, Some("8080".to_string()));
        assert_eq!(config.log_level, Some(1));
    }

    #[test]
    fn test_server_config_serde() {
        let config = ServerConfig {
            port: Some("3000".to_string()),
            log_level: Some(2),
            pool: None,
        };
        let json = serde_json::to_string(&config).unwrap();
        assert!(json.contains("\"port\":\"3000\""));
        assert!(json.contains("\"log_level\":2"));

        let parsed: ServerConfig = serde_json::from_str(&json).unwrap();
        assert_eq!(parsed.port, config.port);
        assert_eq!(parsed.log_level, config.log_level);
    }

    #[test]
    fn test_pool_config_default() {
        let config = PoolConfig::default();
        assert_eq!(config.max_connections(), 10);
        assert_eq!(config.min_connections(), 1);
        assert_eq!(config.acquire_timeout().as_secs(), 30);
        assert_eq!(config.idle_timeout().as_secs(), 600);
        assert_eq!(config.max_lifetime().as_secs(), 1800);
    }
}
