//! Connector manager for managing multiple database connections

use hyperterse_core::{Adapter, HyperterseError};
use hyperterse_types::Connector as ConnectorType;
use std::collections::HashMap;
use std::sync::Arc;
use tokio::sync::RwLock;

use super::mongodb::MongoDbConnector;
use super::mysql::MySqlConnector;
use super::postgres::PostgresConnector;
use super::redis::RedisConnector;
use super::traits::Connector;

/// Manages multiple database connectors
pub struct ConnectorManager {
    connectors: RwLock<HashMap<String, Arc<dyn Connector>>>,
}

impl ConnectorManager {
    /// Create a new empty connector manager
    pub fn new() -> Self {
        Self {
            connectors: RwLock::new(HashMap::new()),
        }
    }

    /// Initialize connectors from adapter configurations
    ///
    /// This initializes all connectors in parallel for faster startup.
    pub async fn initialize(&self, adapters: &[Adapter]) -> Result<(), HyperterseError> {
        use tokio::task::JoinSet;

        let mut set = JoinSet::new();

        // Spawn connector initialization tasks
        for adapter in adapters.iter().cloned() {
            set.spawn(async move {
                let connector = Self::create_connector(&adapter).await?;
                Ok::<_, HyperterseError>((adapter.name.clone(), connector))
            });
        }

        // Collect results
        let mut connectors = self.connectors.write().await;
        while let Some(result) = set.join_next().await {
            let (name, connector) = result
                .map_err(|e| HyperterseError::Connector(format!("Task join error: {}", e)))??;
            connectors.insert(name, connector);
        }

        Ok(())
    }

    /// Create a single connector based on adapter configuration
    async fn create_connector(adapter: &Adapter) -> Result<Arc<dyn Connector>, HyperterseError> {
        match adapter.connector {
            ConnectorType::Postgres => {
                let connector = PostgresConnector::new(&adapter.url).await?;
                Ok(Arc::new(connector))
            }
            ConnectorType::Mysql => {
                let connector = MySqlConnector::new(&adapter.url).await?;
                Ok(Arc::new(connector))
            }
            ConnectorType::Redis => {
                let connector = RedisConnector::new(&adapter.url).await?;
                Ok(Arc::new(connector))
            }
            ConnectorType::Mongodb => {
                let connector = MongoDbConnector::new(&adapter.url).await?;
                Ok(Arc::new(connector))
            }
        }
    }

    /// Get a connector by adapter name
    pub async fn get(&self, name: &str) -> Result<Arc<dyn Connector>, HyperterseError> {
        let connectors = self.connectors.read().await;
        connectors
            .get(name)
            .cloned()
            .ok_or_else(|| HyperterseError::AdapterNotFound(name.to_string()))
    }

    /// Check if a connector exists
    pub async fn has(&self, name: &str) -> bool {
        let connectors = self.connectors.read().await;
        connectors.contains_key(name)
    }

    /// Get the names of all registered connectors
    pub async fn names(&self) -> Vec<String> {
        let connectors = self.connectors.read().await;
        connectors.keys().cloned().collect()
    }

    /// Run health checks on all connectors in parallel
    pub async fn health_check_all(&self) -> HashMap<String, Result<(), String>> {
        use futures::stream::{self, StreamExt};

        let connectors = self.connectors.read().await;
        let connector_list: Vec<_> = connectors
            .iter()
            .map(|(name, connector)| (name.clone(), connector.clone()))
            .collect();
        drop(connectors); // Release read lock before async work

        // Run all health checks concurrently
        let results: Vec<_> = stream::iter(connector_list)
            .map(|(name, connector)| async move {
                let result = connector.health_check().await.map_err(|e| e.to_string());
                (name, result)
            })
            .buffer_unordered(16) // Run up to 16 health checks concurrently
            .collect()
            .await;

        results.into_iter().collect()
    }

    /// Close all connectors gracefully
    pub async fn close_all(&self) -> Result<(), HyperterseError> {
        let connectors = self.connectors.read().await;
        let mut errors = Vec::new();

        for (name, connector) in connectors.iter() {
            if let Err(e) = connector.close().await {
                errors.push(format!("{}: {}", name, e));
            }
        }

        if errors.is_empty() {
            Ok(())
        } else {
            Err(HyperterseError::Connector(format!(
                "Errors closing connectors: {}",
                errors.join(", ")
            )))
        }
    }
}

impl Default for ConnectorManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_empty_manager() {
        let manager = ConnectorManager::new();
        assert!(manager.names().await.is_empty());
    }

    #[tokio::test]
    async fn test_get_nonexistent() {
        let manager = ConnectorManager::new();
        let result = manager.get("nonexistent").await;
        assert!(result.is_err());
    }
}
