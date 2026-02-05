//! OpenAPI documentation handler

use axum::{extract::State, http::StatusCode, response::IntoResponse, Json};
use serde_json::json;
use std::sync::Arc;

use crate::executor::QueryExecutor;

/// Handler for OpenAPI documentation
pub struct OpenApiHandler;

impl OpenApiHandler {
    /// Handle GET /docs
    pub async fn handle(State(executor): State<Arc<QueryExecutor>>) -> impl IntoResponse {
        let model = executor.model();
        let spec = Self::generate_spec(model);
        (StatusCode::OK, Json(spec))
    }

    /// Generate OpenAPI 3.0 specification (static version for CLI use)
    pub fn generate_spec_static(model: &hyperterse_core::Model) -> serde_json::Value {
        Self::generate_spec(model)
    }

    /// Generate OpenAPI 3.0 specification
    fn generate_spec(model: &hyperterse_core::Model) -> serde_json::Value {
        let mut paths = serde_json::Map::new();

        for query in &model.queries {
            let path = format!("/query/{}", query.name);

            // Build request body schema
            let mut properties = serde_json::Map::new();
            let mut required: Vec<String> = Vec::new();

            for input in &query.inputs {
                let type_str = Self::primitive_to_openapi_type(input.primitive_type);
                let format = Self::primitive_to_openapi_format(input.primitive_type);

                let mut prop = serde_json::Map::new();
                prop.insert("type".to_string(), json!(type_str));
                if let Some(fmt) = format {
                    prop.insert("format".to_string(), json!(fmt));
                }
                if let Some(desc) = &input.description {
                    prop.insert("description".to_string(), json!(desc));
                }
                if let Some(default) = &input.default {
                    prop.insert("default".to_string(), default.clone());
                }

                properties.insert(input.name.clone(), serde_json::Value::Object(prop));

                if input.required {
                    required.push(input.name.clone());
                }
            }

            let operation = json!({
                "summary": query.description.as_deref().unwrap_or(&query.name),
                "operationId": query.name.replace('-', "_"),
                "tags": ["queries"],
                "requestBody": {
                    "required": true,
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "inputs": {
                                        "type": "object",
                                        "properties": properties,
                                        "required": required
                                    }
                                }
                            }
                        }
                    }
                },
                "responses": {
                    "200": {
                        "description": "Successful response",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/QueryResponse"
                                }
                            }
                        }
                    },
                    "400": {
                        "description": "Bad request - validation error",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/QueryResponse"
                                }
                            }
                        }
                    },
                    "404": {
                        "description": "Query not found",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/QueryResponse"
                                }
                            }
                        }
                    },
                    "500": {
                        "description": "Internal server error",
                        "content": {
                            "application/json": {
                                "schema": {
                                    "$ref": "#/components/schemas/QueryResponse"
                                }
                            }
                        }
                    }
                }
            });

            paths.insert(path, json!({ "post": operation }));
        }

        json!({
            "openapi": "3.0.3",
            "info": {
                "title": model.name,
                "version": "1.0.0",
                "description": format!("API generated by Hyperterse for {}", model.name)
            },
            "servers": [
                {
                    "url": format!("http://localhost:{}", model.port()),
                    "description": "Local development server"
                }
            ],
            "paths": paths,
            "components": {
                "schemas": {
                    "QueryResponse": {
                        "type": "object",
                        "properties": {
                            "success": {
                                "type": "boolean",
                                "description": "Whether the query succeeded"
                            },
                            "error": {
                                "type": "string",
                                "description": "Error message if the query failed"
                            },
                            "results": {
                                "type": "array",
                                "items": {
                                    "type": "object",
                                    "additionalProperties": true
                                },
                                "description": "Query results as an array of objects"
                            }
                        },
                        "required": ["success", "results"]
                    }
                }
            },
            "tags": [
                {
                    "name": "queries",
                    "description": "Query endpoints"
                }
            ]
        })
    }

    /// Convert primitive type to OpenAPI type
    fn primitive_to_openapi_type(primitive: hyperterse_types::Primitive) -> &'static str {
        match primitive {
            hyperterse_types::Primitive::String => "string",
            hyperterse_types::Primitive::Int => "integer",
            hyperterse_types::Primitive::Float => "number",
            hyperterse_types::Primitive::Boolean => "boolean",
            hyperterse_types::Primitive::Uuid => "string",
            hyperterse_types::Primitive::Datetime => "string",
        }
    }

    /// Convert primitive type to OpenAPI format (if applicable)
    fn primitive_to_openapi_format(primitive: hyperterse_types::Primitive) -> Option<&'static str> {
        match primitive {
            hyperterse_types::Primitive::Int => Some("int64"),
            hyperterse_types::Primitive::Float => Some("double"),
            hyperterse_types::Primitive::Uuid => Some("uuid"),
            hyperterse_types::Primitive::Datetime => Some("date-time"),
            _ => None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use hyperterse_core::{Adapter, Input, Model, Query};
    use hyperterse_types::{Connector, Primitive};

    fn create_test_model() -> Model {
        Model {
            name: "test-api".to_string(),
            adapters: vec![Adapter::new("db", Connector::Postgres, "postgres://localhost/test")],
            queries: vec![Query::new(
                "get-user",
                "db",
                "SELECT * FROM users WHERE id = {{ inputs.id }}",
            )
            .with_description("Get a user by ID")
            .with_input(Input::new("id", Primitive::Int))],
            server: None,
            export: None,
        }
    }

    #[test]
    fn test_generate_spec() {
        let model = create_test_model();
        let spec = OpenApiHandler::generate_spec(&model);

        assert_eq!(spec["openapi"], "3.0.3");
        assert_eq!(spec["info"]["title"], "test-api");
        assert!(spec["paths"]["/query/get-user"].is_object());
    }
}
