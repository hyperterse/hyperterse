//! Shared runtime application state (HTTP handlers)

use crate::executor::QueryExecutor;
use serde_json::Value;
use std::collections::HashMap;
use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};
use tokio::sync::{broadcast, RwLock};
use uuid::Uuid;

/// `MCP-Session-Id` header name (Streamable HTTP transport).
pub const MCP_SESSION_ID_HEADER: &str = "mcp-session-id";

/// `MCP-Protocol-Version` header name (Streamable HTTP transport).
pub const MCP_PROTOCOL_VERSION_HEADER: &str = "mcp-protocol-version";

/// Latest protocol version supported by this server (per MCP "latest").
pub const MCP_LATEST_PROTOCOL_VERSION: &str = "2025-11-25";

/// Protocol version assumed by MCP when header is absent.
pub const MCP_DEFAULT_PROTOCOL_VERSION: &str = "2025-03-26";

/// Application state shared across handlers.
#[derive(Clone)]
pub struct AppState {
    pub executor: Arc<QueryExecutor>,
    pub mcp_sessions: Arc<McpSessions>,
}

impl AppState {
    pub fn new(executor: Arc<QueryExecutor>) -> Self {
        Self {
            executor,
            mcp_sessions: Arc::new(McpSessions::new()),
        }
    }
}

pub struct McpSession {
    pub tx: broadcast::Sender<Value>,
    counter: AtomicU64,
}

impl McpSession {
    fn new() -> Self {
        // Small buffer; if clients are slow, they can lag and reconnect.
        let (tx, _) = broadcast::channel::<Value>(128);
        Self {
            tx,
            counter: AtomicU64::new(1),
        }
    }

    pub fn next_event_seq(&self) -> u64 {
        self.counter.fetch_add(1, Ordering::Relaxed)
    }
}

/// In-memory MCP session store.
pub struct McpSessions {
    sessions: RwLock<HashMap<String, Arc<McpSession>>>,
}

impl McpSessions {
    pub fn new() -> Self {
        Self {
            sessions: RwLock::new(HashMap::new()),
        }
    }

    pub async fn create(&self) -> String {
        let id = Uuid::new_v4().to_string();
        let mut guard = self.sessions.write().await;
        guard.insert(id.clone(), Arc::new(McpSession::new()));
        id
    }

    pub async fn get(&self, id: &str) -> Option<Arc<McpSession>> {
        let guard = self.sessions.read().await;
        guard.get(id).cloned()
    }

    pub async fn remove(&self, id: &str) -> bool {
        let mut guard = self.sessions.write().await;
        guard.remove(id).is_some()
    }
}

