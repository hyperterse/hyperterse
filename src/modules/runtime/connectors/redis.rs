//! Redis connector implementation

use async_trait::async_trait;
use hyperterse_core::HyperterseError;
use redis::aio::ConnectionManager;
use redis::{AsyncCommands, Client, RedisResult};
use std::collections::HashMap;

use super::traits::{Connector, ConnectorResult};

/// Redis key-value store connector
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

    /// Parse and execute a Redis command
    async fn execute_command(&self, statement: &str) -> Result<serde_json::Value, HyperterseError> {
        let parts: Vec<&str> = statement.split_whitespace().collect();
        if parts.is_empty() {
            return Err(HyperterseError::Redis("Empty Redis command".to_string()));
        }

        let command = parts[0].to_uppercase();
        let args = &parts[1..];
        let mut conn = self.conn.clone();

        match command.as_str() {
            "GET" => {
                if args.is_empty() {
                    return Err(HyperterseError::Redis("GET requires a key".to_string()));
                }
                let result: RedisResult<Option<String>> = conn.get(args[0]).await;
                match result {
                    Ok(Some(value)) => Ok(serde_json::Value::String(value)),
                    Ok(None) => Ok(serde_json::Value::Null),
                    Err(e) => Err(HyperterseError::Redis(format!("GET failed: {}", e))),
                }
            }
            "SET" => {
                if args.len() < 2 {
                    return Err(HyperterseError::Redis(
                        "SET requires key and value".to_string(),
                    ));
                }
                let result: RedisResult<()> = conn.set(args[0], args[1]).await;
                match result {
                    Ok(()) => Ok(serde_json::json!({"ok": true})),
                    Err(e) => Err(HyperterseError::Redis(format!("SET failed: {}", e))),
                }
            }
            "DEL" => {
                if args.is_empty() {
                    return Err(HyperterseError::Redis(
                        "DEL requires at least one key".to_string(),
                    ));
                }
                let keys: Vec<&str> = args.to_vec();
                let result: RedisResult<i64> = conn.del(&keys[..]).await;
                match result {
                    Ok(count) => Ok(serde_json::json!({"deleted": count})),
                    Err(e) => Err(HyperterseError::Redis(format!("DEL failed: {}", e))),
                }
            }
            "EXISTS" => {
                if args.is_empty() {
                    return Err(HyperterseError::Redis("EXISTS requires a key".to_string()));
                }
                let result: RedisResult<bool> = conn.exists(args[0]).await;
                match result {
                    Ok(exists) => Ok(serde_json::Value::Bool(exists)),
                    Err(e) => Err(HyperterseError::Redis(format!("EXISTS failed: {}", e))),
                }
            }
            "KEYS" => {
                let pattern = args.first().unwrap_or(&"*");
                let result: RedisResult<Vec<String>> = conn.keys(*pattern).await;
                match result {
                    Ok(keys) => Ok(serde_json::json!(keys)),
                    Err(e) => Err(HyperterseError::Redis(format!("KEYS failed: {}", e))),
                }
            }
            "HGET" => {
                if args.len() < 2 {
                    return Err(HyperterseError::Redis(
                        "HGET requires key and field".to_string(),
                    ));
                }
                let result: RedisResult<Option<String>> = conn.hget(args[0], args[1]).await;
                match result {
                    Ok(Some(value)) => Ok(serde_json::Value::String(value)),
                    Ok(None) => Ok(serde_json::Value::Null),
                    Err(e) => Err(HyperterseError::Redis(format!("HGET failed: {}", e))),
                }
            }
            "HSET" => {
                if args.len() < 3 {
                    return Err(HyperterseError::Redis(
                        "HSET requires key, field, and value".to_string(),
                    ));
                }
                let result: RedisResult<i64> = conn.hset(args[0], args[1], args[2]).await;
                match result {
                    Ok(count) => Ok(serde_json::json!({"new_fields": count})),
                    Err(e) => Err(HyperterseError::Redis(format!("HSET failed: {}", e))),
                }
            }
            "HGETALL" => {
                if args.is_empty() {
                    return Err(HyperterseError::Redis("HGETALL requires a key".to_string()));
                }
                let result: RedisResult<HashMap<String, String>> = conn.hgetall(args[0]).await;
                match result {
                    Ok(map) => Ok(serde_json::json!(map)),
                    Err(e) => Err(HyperterseError::Redis(format!("HGETALL failed: {}", e))),
                }
            }
            "LPUSH" | "RPUSH" => {
                if args.len() < 2 {
                    return Err(HyperterseError::Redis(format!(
                        "{} requires key and value(s)",
                        command
                    )));
                }
                let values: Vec<&str> = args[1..].to_vec();
                let result: RedisResult<i64> = if command == "LPUSH" {
                    conn.lpush(args[0], &values[..]).await
                } else {
                    conn.rpush(args[0], &values[..]).await
                };
                match result {
                    Ok(len) => Ok(serde_json::json!({"length": len})),
                    Err(e) => Err(HyperterseError::Redis(format!("{} failed: {}", command, e))),
                }
            }
            "LRANGE" => {
                if args.len() < 3 {
                    return Err(HyperterseError::Redis(
                        "LRANGE requires key, start, and stop".to_string(),
                    ));
                }
                let start: isize = args[1].parse().unwrap_or(0);
                let stop: isize = args[2].parse().unwrap_or(-1);
                let result: RedisResult<Vec<String>> = conn.lrange(args[0], start, stop).await;
                match result {
                    Ok(values) => Ok(serde_json::json!(values)),
                    Err(e) => Err(HyperterseError::Redis(format!("LRANGE failed: {}", e))),
                }
            }
            "INCR" => {
                if args.is_empty() {
                    return Err(HyperterseError::Redis("INCR requires a key".to_string()));
                }
                let result: RedisResult<i64> = conn.incr(args[0], 1).await;
                match result {
                    Ok(value) => Ok(serde_json::json!(value)),
                    Err(e) => Err(HyperterseError::Redis(format!("INCR failed: {}", e))),
                }
            }
            "DECR" => {
                if args.is_empty() {
                    return Err(HyperterseError::Redis("DECR requires a key".to_string()));
                }
                let result: RedisResult<i64> = conn.decr(args[0], 1).await;
                match result {
                    Ok(value) => Ok(serde_json::json!(value)),
                    Err(e) => Err(HyperterseError::Redis(format!("DECR failed: {}", e))),
                }
            }
            "EXPIRE" => {
                if args.len() < 2 {
                    return Err(HyperterseError::Redis(
                        "EXPIRE requires key and seconds".to_string(),
                    ));
                }
                let seconds: i64 = args[1].parse().unwrap_or(0);
                let result: RedisResult<bool> = conn.expire(args[0], seconds).await;
                match result {
                    Ok(set) => Ok(serde_json::json!({"set": set})),
                    Err(e) => Err(HyperterseError::Redis(format!("EXPIRE failed: {}", e))),
                }
            }
            "TTL" => {
                if args.is_empty() {
                    return Err(HyperterseError::Redis("TTL requires a key".to_string()));
                }
                let result: RedisResult<i64> = conn.ttl(args[0]).await;
                match result {
                    Ok(ttl) => Ok(serde_json::json!(ttl)),
                    Err(e) => Err(HyperterseError::Redis(format!("TTL failed: {}", e))),
                }
            }
            _ => Err(HyperterseError::Redis(format!(
                "Unsupported Redis command: {}",
                command
            ))),
        }
    }
}

#[async_trait]
impl Connector for RedisConnector {
    async fn execute(
        &self,
        statement: &str,
        _params: &HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError> {
        let result = self.execute_command(statement).await?;

        // Wrap the result in a single-row result set
        let mut row = HashMap::new();
        row.insert("result".to_string(), result);

        Ok(vec![row])
    }

    async fn close(&self) -> Result<(), HyperterseError> {
        // ConnectionManager handles connection cleanup automatically
        Ok(())
    }

    async fn health_check(&self) -> Result<(), HyperterseError> {
        let mut conn = self.conn.clone();
        let result: RedisResult<String> = redis::cmd("PING").query_async(&mut conn).await;
        match result {
            Ok(response) if response == "PONG" => Ok(()),
            Ok(response) => Err(HyperterseError::Redis(format!(
                "Unexpected PING response: {}",
                response
            ))),
            Err(e) => Err(HyperterseError::Redis(format!(
                "Redis health check failed: {}",
                e
            ))),
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
