//! MongoDB connector implementation

use async_trait::async_trait;
use bson::{doc, Bson, Document};
use hyperterse_core::HyperterseError;
use mongodb::{options::ClientOptions, Client, Database};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

use super::traits::{Connector, ConnectorResult};

/// MongoDB document database connector
pub struct MongoDbConnector {
    client: Client,
    default_db: Option<String>,
}

/// MongoDB statement structure
#[derive(Debug, Clone, Serialize, Deserialize)]
struct MongoStatement {
    database: Option<String>,
    collection: String,
    operation: String,
    #[serde(default)]
    filter: Option<serde_json::Value>,
    #[serde(default)]
    document: Option<serde_json::Value>,
    #[serde(default)]
    documents: Option<Vec<serde_json::Value>>,
    #[serde(default)]
    update: Option<serde_json::Value>,
    #[serde(default)]
    pipeline: Option<Vec<serde_json::Value>>,
    #[serde(default)]
    options: Option<MongoOptions>,
}

/// MongoDB operation options
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
struct MongoOptions {
    #[serde(default)]
    limit: Option<i64>,
    #[serde(default)]
    skip: Option<u64>,
    #[serde(default)]
    sort: Option<serde_json::Value>,
    #[serde(default)]
    projection: Option<serde_json::Value>,
    #[serde(default)]
    upsert: Option<bool>,
}

impl MongoDbConnector {
    /// Create a new MongoDB connector
    pub async fn new(url: &str) -> Result<Self, HyperterseError> {
        let mut options = ClientOptions::parse(url).await.map_err(|e| {
            HyperterseError::MongoDB(format!("MongoDB options parse failed: {}", e))
        })?;

        // Set default pool options if not specified
        options.min_pool_size = options.min_pool_size.or(Some(1));
        options.max_pool_size = options.max_pool_size.or(Some(10));

        let client = Client::with_options(options).map_err(|e| {
            HyperterseError::MongoDB(format!("MongoDB client creation failed: {}", e))
        })?;

        // Extract default database from URL if present
        let default_db = client.default_database().map(|db| db.name().to_string());

        Ok(Self { client, default_db })
    }

    /// Get a database reference
    fn get_database(&self, name: Option<&str>) -> Result<Database, HyperterseError> {
        match name.or(self.default_db.as_deref()) {
            Some(db_name) => Ok(self.client.database(db_name)),
            None => Err(HyperterseError::MongoDB(
                "No database specified and no default database in connection string".to_string(),
            )),
        }
    }

    /// Convert a JSON value to a BSON document
    fn json_to_bson(value: &serde_json::Value) -> Result<Bson, HyperterseError> {
        // Handle special $oid format for ObjectId
        if let Some(obj) = value.as_object() {
            if let Some(oid) = obj.get("$oid") {
                if let Some(oid_str) = oid.as_str() {
                    let object_id = bson::oid::ObjectId::parse_str(oid_str).map_err(|e| {
                        HyperterseError::MongoDB(format!("Invalid ObjectId: {}", e))
                    })?;
                    return Ok(Bson::ObjectId(object_id));
                }
            }
        }

        bson::to_bson(value)
            .map_err(|e| HyperterseError::MongoDB(format!("JSON to BSON conversion failed: {}", e)))
    }

    /// Convert a JSON value to a BSON document
    fn json_to_document(value: &serde_json::Value) -> Result<Document, HyperterseError> {
        match Self::json_to_bson(value)? {
            Bson::Document(doc) => Ok(doc),
            _ => Err(HyperterseError::MongoDB(
                "Expected a JSON object for BSON document".to_string(),
            )),
        }
    }

    /// Convert a BSON value to a JSON value
    fn bson_to_json(bson: Bson) -> serde_json::Value {
        match bson {
            Bson::ObjectId(oid) => serde_json::Value::String(oid.to_hex()),
            Bson::DateTime(dt) => serde_json::Value::String(
                chrono::DateTime::from_timestamp_millis(dt.timestamp_millis())
                    .map(|d| d.to_rfc3339())
                    .unwrap_or_else(|| dt.to_string()),
            ),
            Bson::Document(doc) => {
                let mut map = serde_json::Map::new();
                for (key, value) in doc {
                    map.insert(key, Self::bson_to_json(value));
                }
                serde_json::Value::Object(map)
            }
            Bson::Array(arr) => {
                serde_json::Value::Array(arr.into_iter().map(Self::bson_to_json).collect())
            }
            Bson::Decimal128(d) => serde_json::Value::String(d.to_string()),
            other => bson::from_bson(other).unwrap_or(serde_json::Value::Null),
        }
    }

    /// Convert a BSON document to a JSON-compatible map
    fn document_to_map(doc: Document) -> HashMap<String, serde_json::Value> {
        let mut map = HashMap::new();
        for (key, value) in doc {
            map.insert(key, Self::bson_to_json(value));
        }
        map
    }

    /// Execute a MongoDB operation
    async fn execute_operation(
        &self,
        stmt: &MongoStatement,
    ) -> Result<ConnectorResult, HyperterseError> {
        let db = self.get_database(stmt.database.as_deref())?;
        let collection = db.collection::<Document>(&stmt.collection);

        match stmt.operation.to_lowercase().as_str() {
            "find" => {
                let filter = stmt
                    .filter
                    .as_ref()
                    .map(Self::json_to_document)
                    .transpose()?
                    .unwrap_or_default();

                let mut options = mongodb::options::FindOptions::default();
                if let Some(opts) = &stmt.options {
                    options.limit = opts.limit;
                    options.skip = opts.skip;
                    if let Some(sort) = &opts.sort {
                        options.sort = Some(Self::json_to_document(sort)?);
                    }
                    if let Some(proj) = &opts.projection {
                        options.projection = Some(Self::json_to_document(proj)?);
                    }
                }

                let mut cursor = collection
                    .find(filter, options)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("find failed: {}", e)))?;

                let mut results = Vec::new();
                while cursor.advance().await.map_err(|e| {
                    HyperterseError::MongoDB(format!("cursor advance failed: {}", e))
                })? {
                    let doc = cursor.deserialize_current().map_err(|e| {
                        HyperterseError::MongoDB(format!("deserialize failed: {}", e))
                    })?;
                    results.push(Self::document_to_map(doc));
                }

                Ok(results)
            }
            "findone" => {
                let filter = stmt
                    .filter
                    .as_ref()
                    .map(Self::json_to_document)
                    .transpose()?
                    .unwrap_or_default();

                let mut options = mongodb::options::FindOneOptions::default();
                if let Some(opts) = &stmt.options {
                    if let Some(proj) = &opts.projection {
                        options.projection = Some(Self::json_to_document(proj)?);
                    }
                }

                let result = collection
                    .find_one(filter, options)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("findOne failed: {}", e)))?;

                match result {
                    Some(doc) => Ok(vec![Self::document_to_map(doc)]),
                    None => Ok(vec![]),
                }
            }
            "insertone" => {
                let document = stmt.document.as_ref().ok_or_else(|| {
                    HyperterseError::MongoDB("insertOne requires document".to_string())
                })?;

                let doc = Self::json_to_document(document)?;
                let result = collection
                    .insert_one(doc, None)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("insertOne failed: {}", e)))?;

                let mut map = HashMap::new();
                map.insert(
                    "insertedId".to_string(),
                    Self::bson_to_json(Bson::ObjectId(result.inserted_id.as_object_id().unwrap())),
                );
                Ok(vec![map])
            }
            "insertmany" => {
                let documents = stmt.documents.as_ref().ok_or_else(|| {
                    HyperterseError::MongoDB("insertMany requires documents".to_string())
                })?;

                let docs: Vec<Document> = documents
                    .iter()
                    .map(Self::json_to_document)
                    .collect::<Result<_, _>>()?;

                let result = collection
                    .insert_many(docs, None)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("insertMany failed: {}", e)))?;

                let inserted_ids: Vec<serde_json::Value> = result
                    .inserted_ids
                    .values()
                    .map(|id| Self::bson_to_json(id.clone()))
                    .collect();

                let mut map = HashMap::new();
                map.insert("insertedIds".to_string(), serde_json::json!(inserted_ids));
                Ok(vec![map])
            }
            "updateone" => {
                let filter = stmt
                    .filter
                    .as_ref()
                    .map(Self::json_to_document)
                    .transpose()?
                    .unwrap_or_default();

                let update = stmt.update.as_ref().ok_or_else(|| {
                    HyperterseError::MongoDB("updateOne requires update".to_string())
                })?;
                let update_doc = Self::json_to_document(update)?;

                let mut options = mongodb::options::UpdateOptions::default();
                if let Some(opts) = &stmt.options {
                    options.upsert = opts.upsert;
                }

                let result = collection
                    .update_one(filter, update_doc, options)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("updateOne failed: {}", e)))?;

                let mut map = HashMap::new();
                map.insert(
                    "matchedCount".to_string(),
                    serde_json::json!(result.matched_count),
                );
                map.insert(
                    "modifiedCount".to_string(),
                    serde_json::json!(result.modified_count),
                );
                if let Some(id) = result.upserted_id {
                    map.insert("upsertedId".to_string(), Self::bson_to_json(id));
                }
                Ok(vec![map])
            }
            "updatemany" => {
                let filter = stmt
                    .filter
                    .as_ref()
                    .map(Self::json_to_document)
                    .transpose()?
                    .unwrap_or_default();

                let update = stmt.update.as_ref().ok_or_else(|| {
                    HyperterseError::MongoDB("updateMany requires update".to_string())
                })?;
                let update_doc = Self::json_to_document(update)?;

                let mut options = mongodb::options::UpdateOptions::default();
                if let Some(opts) = &stmt.options {
                    options.upsert = opts.upsert;
                }

                let result = collection
                    .update_many(filter, update_doc, options)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("updateMany failed: {}", e)))?;

                let mut map = HashMap::new();
                map.insert(
                    "matchedCount".to_string(),
                    serde_json::json!(result.matched_count),
                );
                map.insert(
                    "modifiedCount".to_string(),
                    serde_json::json!(result.modified_count),
                );
                Ok(vec![map])
            }
            "deleteone" => {
                let filter = stmt
                    .filter
                    .as_ref()
                    .map(Self::json_to_document)
                    .transpose()?
                    .unwrap_or_default();

                let result = collection
                    .delete_one(filter, None)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("deleteOne failed: {}", e)))?;

                let mut map = HashMap::new();
                map.insert(
                    "deletedCount".to_string(),
                    serde_json::json!(result.deleted_count),
                );
                Ok(vec![map])
            }
            "deletemany" => {
                let filter = stmt
                    .filter
                    .as_ref()
                    .map(Self::json_to_document)
                    .transpose()?
                    .unwrap_or_default();

                let result = collection
                    .delete_many(filter, None)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("deleteMany failed: {}", e)))?;

                let mut map = HashMap::new();
                map.insert(
                    "deletedCount".to_string(),
                    serde_json::json!(result.deleted_count),
                );
                Ok(vec![map])
            }
            "aggregate" => {
                let pipeline = stmt.pipeline.as_ref().ok_or_else(|| {
                    HyperterseError::MongoDB("aggregate requires pipeline".to_string())
                })?;

                let pipeline_docs: Vec<Document> = pipeline
                    .iter()
                    .map(Self::json_to_document)
                    .collect::<Result<_, _>>()?;

                let mut cursor = collection
                    .aggregate(pipeline_docs, None)
                    .await
                    .map_err(|e| HyperterseError::MongoDB(format!("aggregate failed: {}", e)))?;

                let mut results = Vec::new();
                while cursor.advance().await.map_err(|e| {
                    HyperterseError::MongoDB(format!("cursor advance failed: {}", e))
                })? {
                    let doc = cursor.deserialize_current().map_err(|e| {
                        HyperterseError::MongoDB(format!("deserialize failed: {}", e))
                    })?;
                    results.push(Self::document_to_map(doc));
                }

                Ok(results)
            }
            "countdocuments" => {
                let filter = stmt
                    .filter
                    .as_ref()
                    .map(Self::json_to_document)
                    .transpose()?
                    .unwrap_or_default();

                let count = collection
                    .count_documents(filter, None)
                    .await
                    .map_err(|e| {
                        HyperterseError::MongoDB(format!("countDocuments failed: {}", e))
                    })?;

                let mut map = HashMap::new();
                map.insert("count".to_string(), serde_json::json!(count));
                Ok(vec![map])
            }
            _ => Err(HyperterseError::MongoDB(format!(
                "Unsupported MongoDB operation: {}",
                stmt.operation
            ))),
        }
    }
}

#[async_trait]
impl Connector for MongoDbConnector {
    async fn execute(
        &self,
        statement: &str,
        _params: &HashMap<String, serde_json::Value>,
    ) -> Result<ConnectorResult, HyperterseError> {
        // Parse the JSON statement
        let stmt: MongoStatement = serde_json::from_str(statement).map_err(|e| {
            HyperterseError::MongoDB(format!("Invalid MongoDB statement JSON: {}", e))
        })?;

        self.execute_operation(&stmt).await
    }

    async fn close(&self) -> Result<(), HyperterseError> {
        // MongoDB client handles connection cleanup automatically
        Ok(())
    }

    async fn health_check(&self) -> Result<(), HyperterseError> {
        self.client
            .database("admin")
            .run_command(doc! { "ping": 1 }, None)
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
    fn test_json_to_bson_objectid() {
        let json = serde_json::json!({"$oid": "507f1f77bcf86cd799439011"});
        let bson = MongoDbConnector::json_to_bson(&json).unwrap();
        assert!(matches!(bson, Bson::ObjectId(_)));
    }
}
