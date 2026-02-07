//! Redis connector implementation (redis::cmd passthrough)

use async_trait::async_trait;
use hyperterse_core::HyperterseError;
use redis::aio::ConnectionManager;
use redis::{Client, Value as RedisValue};
use std::collections::HashMap;

use super::traits::{Connector, ConnectorResult};

/// Redis connector: executes arbitrary Redis commands via redis::cmd().
pub struct RedisConnector {
    conn: ConnectionManager,
}

impl RedisConnector {
    /// Create a new Redis connector
    pub async fn new(url: &str) -> Result<Self, HyperterseError> {
        let client = Client::open(url)
            .map_err(|e| HyperterseError::Redis(format!("Redis client creation failed: {}", e)))?;
        let conn = ConnectionManager::new(client)
            .await
            .map_err(|e| HyperterseError::Redis(format!("Redis connection failed: {}", e)))?;
        Ok(Self { conn })
    }
}

fn redis_value_to_json(v: RedisValue) -> serde_json::Value {
    match v {
        RedisValue::Nil => serde_json::Value::Null,
        RedisValue::Int(i) => serde_json::json!(i),
        RedisValue::Data(d) => serde_json::Value::String(
            String::from_utf8(d).unwrap_or_else(|_| String::from("<binary>")),
        ),
        RedisValue::Bulk(bulk) => {
            serde_json::Value::Array(bulk.into_iter().map(redis_value_to_json).collect())
        }
        RedisValue::Status(s) => serde_json::Value::String(s),
        RedisValue::Okay => serde_json::Value::String("OK".to_string()),
    }
}

#[async_trait]
impl Connector for RedisConnector {
    async fn execute(
        &self,
        statement: &str,
        _params: &HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError> {
        let parts: Vec<&str> = statement.split_whitespace().collect();
        if parts.is_empty() {
            return Err(HyperterseError::Redis("Empty Redis command".to_string()));
        }
        let command = parts[0];
        let args = &parts[1..];

        let mut cmd = redis::cmd(command);
        for arg in args {
            cmd.arg(*arg);
        }

        let mut conn = self.conn.clone();
        let result: RedisValue = cmd
            .query_async(&mut conn)
            .await
            .map_err(|e| HyperterseError::Redis(format!("Redis command failed: {}", e)))?;

        let mut row = HashMap::new();
        row.insert("result".to_string(), redis_value_to_json(result));
        Ok(vec![row])
    }

    async fn close(&self) -> Result<(), HyperterseError> {
        Ok(())
    }

    async fn health_check(&self) -> Result<(), HyperterseError> {
        let mut conn = self.conn.clone();
        let result: Result<String, _> = redis::cmd("PING").query_async(&mut conn).await;
        match result {
            Ok(r) if r == "PONG" => Ok(()),
            Ok(r) => Err(HyperterseError::Redis(format!("Unexpected PING response: {}", r))),
            Err(e) => Err(HyperterseError::Redis(format!("Redis health check failed: {}", e))),
        }
    }

    fn connector_type(&self) -> &'static str {
        "redis"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    #[ignore] // Requires a running Redis instance
    async fn test_redis_connection() {
        let connector = RedisConnector::new("redis://localhost:6379").await;
        assert!(connector.is_ok());
    }
}
