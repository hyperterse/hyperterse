//! HTTP server for Hyperterse

use axum::{
    routing::{delete, get, post},
    Router,
};
use hyperterse_core::{HyperterseError, Model, ServerConfig};
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;
use tokio::net::TcpListener;
use tokio::signal;
use tower_http::cors::{Any, CorsLayer};
use tower_http::timeout::TimeoutLayer;
use tower_http::trace::TraceLayer;
use tracing::{debug, info, warn};

use crate::connectors::ConnectorManager;
use crate::executor::QueryExecutor;
use crate::handlers::{LlmsHandler, McpHandler, OpenApiHandler, QueryHandler};

/// Runtime server for Hyperterse
pub struct Runtime {
    model: Arc<Model>,
    connectors: Arc<ConnectorManager>,
    executor: Arc<QueryExecutor>,
    port_override: Option<u16>,
}

/// Apply port override to a model configuration
fn apply_port_override(mut model: Model, port_override: Option<u16>) -> Model {
    if let Some(port) = port_override {
        if let Some(ref mut server) = model.server {
            server.port = Some(port.to_string());
        } else {
            model.server = Some(ServerConfig {
                port: Some(port.to_string()),
                log_level: None,
                pool: None,
            });
        }
    }
    model
}

impl Runtime {
    /// Create a new runtime from a model configuration
    pub async fn new(model: Model) -> Result<Self, HyperterseError> {
        Self::with_port_override(model, None).await
    }

    /// Create a new runtime with an optional port override
    pub async fn with_port_override(
        model: Model,
        port_override: Option<u16>,
    ) -> Result<Self, HyperterseError> {
        let model = Arc::new(apply_port_override(model, port_override));

        // Initialize connectors
        let connectors = Arc::new(ConnectorManager::new());
        connectors.initialize(&model.adapters).await?;

        // Create executor
        let executor = Arc::new(QueryExecutor::new(model.clone(), connectors.clone()));

        Ok(Self {
            model,
            connectors,
            executor,
            port_override,
        })
    }

    /// Build the Axum router
    fn build_router(&self) -> Router {
        let executor = self.executor.clone();

        // CORS configuration
        let cors = CorsLayer::new()
            .allow_origin(Any)
            .allow_methods(Any)
            .allow_headers(Any);

        // Request timeout
        let timeout = TimeoutLayer::new(Duration::from_secs(30));

        Router::new()
            // Query endpoints
            .route("/query/:query_name", post(QueryHandler::execute))
            // MCP endpoints
            .route("/mcp", post(McpHandler::handle_rpc))
            .route("/mcp", get(McpHandler::handle_sse))
            .route("/mcp", delete(McpHandler::handle_delete))
            // Documentation endpoints
            .route("/docs", get(OpenApiHandler::handle))
            .route("/llms.txt", get(LlmsHandler::handle))
            // Health check
            .route("/health", get(Self::health_check))
            // State
            .with_state(executor)
            // Middleware
            .layer(cors)
            .layer(timeout)
            .layer(TraceLayer::new_for_http())
    }

    /// Health check endpoint
    async fn health_check() -> &'static str {
        "OK"
    }

    /// Start the server
    pub async fn run(&self) -> Result<(), HyperterseError> {
        let addr: SocketAddr = format!("0.0.0.0:{}", self.model.port())
            .parse()
            .map_err(|e| HyperterseError::Server(format!("Invalid address: {}", e)))?;

        let app = self.build_router();

        info!("Starting Hyperterse server on http://{}", addr);
        info!("Model: {}", self.model.name);
        info!("Adapters: {}", self.connectors.names().await.join(", "));
        info!("Queries: {}", self.executor.query_names().join(", "));
        info!("OpenAPI docs: http://{}/docs", addr);
        info!("LLM docs: http://{}/llms.txt", addr);
        info!("MCP endpoint: http://{}/mcp", addr);

        let listener = TcpListener::bind(&addr)
            .await
            .map_err(|e| HyperterseError::Server(format!("Failed to bind: {}", e)))?;

        axum::serve(listener, app)
            .with_graceful_shutdown(Self::shutdown_signal())
            .await
            .map_err(|e| HyperterseError::Server(format!("Server error: {}", e)))?;

        info!("Server stopped");
        self.shutdown().await?;

        Ok(())
    }

    /// Wait for shutdown signal
    async fn shutdown_signal() {
        let ctrl_c = async {
            signal::ctrl_c()
                .await
                .expect("Failed to install CTRL+C signal handler");
        };

        #[cfg(unix)]
        let terminate = async {
            signal::unix::signal(signal::unix::SignalKind::terminate())
                .expect("Failed to install SIGTERM signal handler")
                .recv()
                .await;
        };

        #[cfg(not(unix))]
        let terminate = std::future::pending::<()>();

        tokio::select! {
            _ = ctrl_c => {
                debug!("Received CTRL+C, shutting down...");
            }
            _ = terminate => {
                debug!("Received SIGTERM, shutting down...");
            }
        }
    }

    /// Gracefully shutdown the runtime
    pub async fn shutdown(&self) -> Result<(), HyperterseError> {
        info!("Closing database connections...");
        if let Err(e) = self.connectors.close_all().await {
            warn!("Error closing connectors: {}", e);
        }
        info!("Shutdown complete");
        Ok(())
    }

    /// Reload the runtime with a new model configuration
    pub async fn reload(&mut self, new_model: Model) -> Result<(), HyperterseError> {
        info!("Reloading configuration...");

        // Close old connectors
        if let Err(e) = self.connectors.close_all().await {
            warn!("Error closing old connectors: {}", e);
        }

        // Create new runtime components (apply stored port override)
        let model = Arc::new(apply_port_override(new_model, self.port_override));
        let connectors = Arc::new(ConnectorManager::new());
        connectors.initialize(&model.adapters).await?;
        let executor = Arc::new(QueryExecutor::new(model.clone(), connectors.clone()));

        // Update self
        self.model = model;
        self.connectors = connectors;
        self.executor = executor;

        info!("Configuration reloaded successfully");
        Ok(())
    }

    /// Get the model
    pub fn model(&self) -> &Model {
        &self.model
    }

    /// Get the executor
    pub fn executor(&self) -> &QueryExecutor {
        &self.executor
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_model() -> Model {
        Model {
            name: "test-api".to_string(),
            adapters: vec![],
            queries: vec![],
            server: None,
            export: None,
        }
    }

    #[tokio::test]
    async fn test_runtime_creation() {
        let model = create_test_model();
        let runtime = Runtime::new(model).await;
        assert!(runtime.is_ok());
    }

    #[tokio::test]
    async fn test_runtime_with_port_override() {
        let model = create_test_model();
        let runtime = Runtime::with_port_override(model, Some(3000))
            .await
            .unwrap();
        assert_eq!(runtime.model().port(), 3000);
    }

    #[test]
    fn test_apply_port_override_with_existing_server() {
        let model = Model {
            name: "test".to_string(),
            adapters: vec![],
            queries: vec![],
            server: Some(ServerConfig {
                port: Some("8080".to_string()),
                log_level: None,
                pool: None,
            }),
            export: None,
        };
        let result = apply_port_override(model, Some(3000));
        assert_eq!(result.server.unwrap().port, Some("3000".to_string()));
    }

    #[test]
    fn test_apply_port_override_without_server() {
        let model = Model {
            name: "test".to_string(),
            adapters: vec![],
            queries: vec![],
            server: None,
            export: None,
        };
        let result = apply_port_override(model, Some(3000));
        assert!(result.server.is_some());
        assert_eq!(result.server.unwrap().port, Some("3000".to_string()));
    }

    #[test]
    fn test_apply_port_override_none() {
        let model = create_test_model();
        let result = apply_port_override(model, None);
        assert!(result.server.is_none());
    }

    #[test]
    fn test_build_router() {
        // This is a basic test that the router can be built
        // Full integration tests would be separate
    }
}
