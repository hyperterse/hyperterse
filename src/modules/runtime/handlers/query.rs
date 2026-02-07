//! Query execution handler

use axum::{
    extract::{Path, State},
    http::StatusCode,
    response::IntoResponse,
    Json,
};
use hyperterse_types::runtime::{QueryRequest, QueryResponse};
use tracing::{error, info};

use crate::state::AppState;

/// Handler for query execution requests
pub struct QueryHandler;

impl QueryHandler {
    /// Handle POST /query/{query_name}
    pub async fn execute(
        State(state): State<AppState>,
        Path(query_name): Path<String>,
        Json(request): Json<QueryRequest>,
    ) -> impl IntoResponse {
        info!("Executing query: {}", query_name);

        match state.executor.execute(&query_name, request.inputs).await {
            Ok(results) => {
                info!(
                    "Query '{}' executed successfully, {} rows returned",
                    query_name,
                    results.len()
                );
                (StatusCode::OK, Json(QueryResponse::success(results)))
            }
            Err(e) => {
                error!("Query '{}' failed: {}", query_name, e);
                let status = match e.status_code() {
                    404 => StatusCode::NOT_FOUND,
                    400 => StatusCode::BAD_REQUEST,
                    _ => StatusCode::INTERNAL_SERVER_ERROR,
                };
                (status, Json(QueryResponse::error(e.sanitized_message())))
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_query_response_success() {
        let response = QueryResponse::success(vec![]);
        assert!(response.success);
        assert!(response.error.is_empty());
    }

    #[test]
    fn test_query_response_error() {
        let response = QueryResponse::error("Something went wrong");
        assert!(!response.success);
        assert_eq!(response.error, "Something went wrong");
    }
}
