//! MySQL connector implementation

use async_trait::async_trait;
use hyperterse_core::{HyperterseError, PoolConfig};
use sqlx::mysql::{MySqlPool, MySqlPoolOptions, MySqlRow};
use sqlx::{Column, Row};
use std::collections::HashMap;

use super::traits::{Connector, ConnectorResult};

/// MySQL database connector
pub struct MySqlConnector {
    pool: MySqlPool,
}

impl MySqlConnector {
    /// Create a new MySQL connector with default pool settings
    pub async fn new(url: &str) -> Result<Self, HyperterseError> {
        Self::with_config(url, &PoolConfig::default()).await
    }

    /// Create a new MySQL connector with custom pool settings
    pub async fn with_config(url: &str, config: &PoolConfig) -> Result<Self, HyperterseError> {
        let pool = MySqlPoolOptions::new()
            .max_connections(config.max_connections())
            .min_connections(config.min_connections())
            .acquire_timeout(config.acquire_timeout())
            .idle_timeout(config.idle_timeout())
            .max_lifetime(config.max_lifetime())
            .connect(url)
            .await
            .map_err(|e| HyperterseError::Database(format!("MySQL connection failed: {}", e)))?;

        Ok(Self { pool })
    }

    /// Convert a MySQL row to a JSON-compatible map
    fn row_to_map(row: &MySqlRow) -> HashMap<String, serde_json::Value> {
        let mut map = HashMap::new();
        let columns = row.columns();

        for column in columns {
            let name = column.name().to_string();
            let value = Self::get_column_value(row, column);
            map.insert(name, value);
        }

        map
    }

    /// Get a column value as a JSON value
    fn get_column_value(row: &MySqlRow, column: &sqlx::mysql::MySqlColumn) -> serde_json::Value {
        use sqlx::TypeInfo;

        let type_name = column.type_info().name();
        let idx = column.ordinal();

        match type_name {
            "BOOLEAN" | "TINYINT(1)" => row
                .try_get::<bool, _>(idx)
                .map(serde_json::Value::Bool)
                .unwrap_or(serde_json::Value::Null),
            "TINYINT" | "SMALLINT" => row
                .try_get::<i16, _>(idx)
                .map(|v| serde_json::Value::Number(v.into()))
                .unwrap_or(serde_json::Value::Null),
            "INT" | "MEDIUMINT" => row
                .try_get::<i32, _>(idx)
                .map(|v| serde_json::Value::Number(v.into()))
                .unwrap_or(serde_json::Value::Null),
            "BIGINT" => row
                .try_get::<i64, _>(idx)
                .map(|v| serde_json::Value::Number(v.into()))
                .unwrap_or(serde_json::Value::Null),
            "FLOAT" => row
                .try_get::<f32, _>(idx)
                .map(|v| {
                    serde_json::Number::from_f64(v as f64)
                        .map(serde_json::Value::Number)
                        .unwrap_or(serde_json::Value::Null)
                })
                .unwrap_or(serde_json::Value::Null),
            "DOUBLE" => row
                .try_get::<f64, _>(idx)
                .map(|v| {
                    serde_json::Number::from_f64(v)
                        .map(serde_json::Value::Number)
                        .unwrap_or(serde_json::Value::Null)
                })
                .unwrap_or(serde_json::Value::Null),
            "DATETIME" | "TIMESTAMP" => row
                .try_get::<chrono::NaiveDateTime, _>(idx)
                .map(|v| serde_json::Value::String(v.format("%Y-%m-%dT%H:%M:%S").to_string()))
                .unwrap_or(serde_json::Value::Null),
            "DATE" => row
                .try_get::<chrono::NaiveDate, _>(idx)
                .map(|v| serde_json::Value::String(v.to_string()))
                .unwrap_or(serde_json::Value::Null),
            "JSON" => row
                .try_get::<serde_json::Value, _>(idx)
                .unwrap_or(serde_json::Value::Null),
            _ => row
                .try_get::<String, _>(idx)
                .map(serde_json::Value::String)
                .unwrap_or(serde_json::Value::Null),
        }
    }
}

#[async_trait]
impl Connector for MySqlConnector {
    async fn execute(
        &self,
        statement: &str,
        _params: &HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError> {
        let rows = sqlx::query(statement)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| HyperterseError::QueryExecution(format!("MySQL query failed: {}", e)))?;

        let results: ConnectorResult = rows.iter().map(Self::row_to_map).collect();
        Ok(results)
    }

    async fn close(&self) -> Result<(), HyperterseError> {
        self.pool.close().await;
        Ok(())
    }

    async fn health_check(&self) -> Result<(), HyperterseError> {
        sqlx::query("SELECT 1")
            .fetch_one(&self.pool)
            .await
            .map_err(|e| HyperterseError::Database(format!("MySQL health check failed: {}", e)))?;
        Ok(())
    }

    fn connector_type(&self) -> &'static str {
        "mysql"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    #[ignore] // Requires a running MySQL instance
    async fn test_mysql_connection() {
        let connector = MySqlConnector::new("mysql://localhost/test").await;
        assert!(connector.is_ok());
    }
}
