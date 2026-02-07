//! MongoDB connector implementation (run_command passthrough)

use async_trait::async_trait;
use bson::{Bson, Document};
use hyperterse_core::HyperterseError;
use mongodb::{options::ClientOptions, Client, Database};
use std::collections::HashMap;

use super::traits::{Connector, ConnectorResult};

/// MongoDB connector: executes native command documents via Database::run_command.
pub struct MongoDbConnector {
    client: Client,
    default_db: Option<String>,
}

impl MongoDbConnector {
    /// Create a new MongoDB connector
    pub async fn new(url: &str) -> Result<Self, HyperterseError> {
        let mut options = ClientOptions::parse(url).await.map_err(|e| {
            HyperterseError::MongoDB(format!("MongoDB options parse failed: {}", e))
        })?;
        options.min_pool_size = options.min_pool_size.or(Some(1));
        options.max_pool_size = options.max_pool_size.or(Some(10));

        let client = Client::with_options(options).map_err(|e| {
            HyperterseError::MongoDB(format!("MongoDB client creation failed: {}", e))
        })?;
        let default_db = client.default_database().map(|db| db.name().to_string());

        Ok(Self { client, default_db })
    }

    fn get_database(&self, name: Option<&str>) -> Result<Database, HyperterseError> {
        match name.or(self.default_db.as_deref()) {
            Some(db_name) => Ok(self.client.database(db_name)),
            None => Err(HyperterseError::MongoDB(
                "No database specified and no default database in connection string".to_string(),
            )),
        }
    }

    fn document_to_row(doc: &Document) -> HashMap<String, serde_json::Value> {
        let mut map = HashMap::new();
        for (k, v) in doc {
            map.insert(k.clone(), bson_to_json(v.clone()));
        }
        map
    }
}

/// Convert a run_command result document to ConnectorResult (cursor.firstBatch or single row).
fn result_to_rows(result: Document) -> ConnectorResult {
    if let Some(cursor_bson) = result.get("cursor").and_then(|c| c.as_document()) {
        if let Some(Bson::Array(batch)) = cursor_bson.get("firstBatch") {
            return batch
                .iter()
                .filter_map(|v| v.as_document())
                .map(|d| MongoDbConnector::document_to_row(d))
                .collect();
        }
    }
    vec![MongoDbConnector::document_to_row(&result)]
}

fn bson_to_json(bson: Bson) -> serde_json::Value {
    match bson {
        Bson::ObjectId(oid) => serde_json::Value::String(oid.to_hex()),
        Bson::DateTime(dt) => serde_json::Value::String(
            chrono::DateTime::from_timestamp_millis(dt.timestamp_millis())
                .map(|d| d.to_rfc3339())
                .unwrap_or_else(|| dt.to_string()),
        ),
        Bson::Document(doc) => {
            let mut m = serde_json::Map::new();
            for (key, value) in doc {
                m.insert(key, bson_to_json(value));
            }
            serde_json::Value::Object(m)
        }
        Bson::Array(arr) => serde_json::Value::Array(arr.into_iter().map(bson_to_json).collect()),
        Bson::Decimal128(d) => serde_json::Value::String(d.to_string()),
        other => bson::from_bson(other).unwrap_or(serde_json::Value::Null),
    }
}

#[async_trait]
impl Connector for MongoDbConnector {
    async fn execute(
        &self,
        statement: &str,
        _params: &HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError> {
        let value: serde_json::Value = serde_json::from_str(statement).map_err(|e| {
            HyperterseError::MongoDB(format!("Invalid MongoDB statement JSON: {}", e))
        })?;

        let bson = bson::to_bson(&value).map_err(|e| {
            HyperterseError::MongoDB(format!("JSON to BSON failed: {}", e))
        })?;

        let mut doc = match bson {
            Bson::Document(d) => d,
            _ => {
                return Err(HyperterseError::MongoDB(
                    "MongoDB statement must be a JSON object".to_string(),
                ))
            }
        };

        let db_name = doc.remove("database").and_then(|v| {
            if let Bson::String(s) = v {
                Some(s)
            } else {
                None
            }
        });

        let db = self.get_database(db_name.as_deref())?;
        let result = db
            .run_command(doc, None)
            .await
            .map_err(|e| HyperterseError::MongoDB(format!("run_command failed: {}", e)))?;

        Ok(result_to_rows(result))
    }

    async fn close(&self) -> Result<(), HyperterseError> {
        Ok(())
    }

    async fn health_check(&self) -> Result<(), HyperterseError> {
        self.client
            .database("admin")
            .run_command(bson::doc! { "ping": 1 }, None)
            .await
            .map_err(|e| HyperterseError::MongoDB(format!("MongoDB health check failed: {}", e)))?;
        Ok(())
    }

    fn connector_type(&self) -> &'static str {
        "mongodb"
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    #[ignore] // Requires a running MongoDB instance
    async fn test_mongodb_connection() {
        let connector = MongoDbConnector::new("mongodb://localhost:27017/test").await;
        assert!(connector.is_ok());
    }

    #[test]
    fn test_result_to_rows_cursor_first_batch() {
        let result = bson::doc! {
            "cursor": {
                "firstBatch": [
                    { "a": 1, "x": "one" },
                    { "b": 2, "y": "two" }
                ]
            }
        };
        let rows = result_to_rows(result);
        assert_eq!(rows.len(), 2);
        assert_eq!(rows[0].get("a"), Some(&serde_json::json!(1)));
        assert_eq!(rows[0].get("x"), Some(&serde_json::Value::String("one".to_string())));
        assert_eq!(rows[1].get("b"), Some(&serde_json::json!(2)));
    }

    #[test]
    fn test_result_to_rows_no_cursor_single_row() {
        let result = bson::doc! { "n": 1, "ok": 1.0 };
        let rows = result_to_rows(result);
        assert_eq!(rows.len(), 1);
        assert_eq!(rows[0].get("n"), Some(&serde_json::json!(1)));
        assert_eq!(rows[0].get("ok"), Some(&serde_json::json!(1.0)));
    }

    #[test]
    fn test_database_extracted_from_statement() {
        let statement = r#"{ "database": "mydb", "find": "orders", "filter": {} }"#;
        let value: serde_json::Value = serde_json::from_str(statement).unwrap();
        let bson = bson::to_bson(&value).unwrap();
        let mut doc = match bson {
            Bson::Document(d) => d,
            _ => panic!("expected document"),
        };
        let db_name = doc.remove("database").and_then(|v| {
            if let Bson::String(s) = v {
                Some(s)
            } else {
                None
            }
        });
        assert_eq!(db_name.as_deref(), Some("mydb"));
        assert!(!doc.contains_key("database"));
        assert_eq!(doc.get("find"), Some(&Bson::String("orders".to_string())));
    }
}
