//! PostgreSQL connector implementation

use async_trait::async_trait;
use hyperterse_core::{HyperterseError, PoolConfig};
use sqlx::postgres::{PgPool, PgPoolOptions, PgRow};
use sqlx::{Column, Row};
use std::collections::HashMap;

use super::traits::{Connector, ConnectorResult};

/// PostgreSQL database connector
pub struct PostgresConnector {
    pool: PgPool,
}

impl PostgresConnector {
    /// Create a new PostgreSQL connector with default pool settings
    pub async fn new(url: &str) -> Result<Self, HyperterseError> {
        Self::with_config(url, &PoolConfig::default()).await
    }

    /// Create a new PostgreSQL connector with custom pool settings
    pub async fn with_config(url: &str, config: &PoolConfig) -> Result<Self, HyperterseError> {
        let pool = PgPoolOptions::new()
            .max_connections(config.max_connections())
            .min_connections(config.min_connections())
            .acquire_timeout(config.acquire_timeout())
            .idle_timeout(config.idle_timeout())
            .max_lifetime(config.max_lifetime())
            .connect(url)
            .await
            .map_err(|e| HyperterseError::Database(format!("PostgreSQL connection failed: {}", e)))?;

        Ok(Self { pool })
    }

    /// Convert a PostgreSQL row to a JSON-compatible map
    fn row_to_map(row: &PgRow) -> HashMap<String, serde_json::Value> {
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
    fn get_column_value(row: &PgRow, column: &sqlx::postgres::PgColumn) -> serde_json::Value {
        use sqlx::TypeInfo;

        let type_name = column.type_info().name();
        let idx = column.ordinal();

        match type_name {
            "BOOL" => row
                .try_get::<bool, _>(idx)
                .map(serde_json::Value::Bool)
                .unwrap_or(serde_json::Value::Null),
            "INT2" => row
                .try_get::<i16, _>(idx)
                .map(|v| serde_json::Value::Number(v.into()))
                .unwrap_or(serde_json::Value::Null),
            "INT4" => row
                .try_get::<i32, _>(idx)
                .map(|v| serde_json::Value::Number(v.into()))
                .unwrap_or(serde_json::Value::Null),
            "INT8" => row
                .try_get::<i64, _>(idx)
                .map(|v| serde_json::Value::Number(v.into()))
                .unwrap_or(serde_json::Value::Null),
            "FLOAT4" => row
                .try_get::<f32, _>(idx)
                .map(|v| {
                    serde_json::Number::from_f64(v as f64)
                        .map(serde_json::Value::Number)
                        .unwrap_or(serde_json::Value::Null)
                })
                .unwrap_or(serde_json::Value::Null),
            "FLOAT8" => row
                .try_get::<f64, _>(idx)
                .map(|v| {
                    serde_json::Number::from_f64(v)
                        .map(serde_json::Value::Number)
                        .unwrap_or(serde_json::Value::Null)
                })
                .unwrap_or(serde_json::Value::Null),
            "UUID" => row
                .try_get::<uuid::Uuid, _>(idx)
                .map(|v| serde_json::Value::String(v.to_string()))
                .unwrap_or(serde_json::Value::Null),
            "TIMESTAMPTZ" | "TIMESTAMP" => row
                .try_get::<chrono::DateTime<chrono::Utc>, _>(idx)
                .map(|v| serde_json::Value::String(v.to_rfc3339()))
                .unwrap_or(serde_json::Value::Null),
            "DATE" => row
                .try_get::<chrono::NaiveDate, _>(idx)
                .map(|v| serde_json::Value::String(v.to_string()))
                .unwrap_or(serde_json::Value::Null),
            "JSON" | "JSONB" => row
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
impl Connector for PostgresConnector {
    async fn execute(
        &self,
        statement: &str,
        _params: &HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError> {
        // Note: Parameters should already be substituted in the statement
        // by the template substitutor before reaching here
        let rows = sqlx::query(statement)
            .fetch_all(&self.pool)
            .await
            .map_err(|e| HyperterseError::QueryExecution(format!("PostgreSQL query failed: {}", e)))?;

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
            .map_err(|e| HyperterseError::Database(format!("PostgreSQL health check failed: {}", e)))?;
        Ok(())
    }

    fn connector_type(&self) -> &'static str {
        "postgres"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    #[ignore] // Requires a running PostgreSQL instance
    async fn test_postgres_connection() {
        let connector = PostgresConnector::new("postgres://localhost/test").await;
        assert!(connector.is_ok());
    }
}
