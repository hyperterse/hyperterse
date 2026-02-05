//! Error types for Hyperterse

use thiserror::Error;

/// Main error type for Hyperterse operations
#[derive(Error, Debug)]
pub enum HyperterseError {
    /// Configuration file parsing error
    #[error("Configuration error: {0}")]
    Config(String),

    /// Configuration validation error
    #[error("Validation error: {0}")]
    Validation(String),

    /// Database connection error
    #[error("Database error: {0}")]
    Database(String),

    /// Redis-specific error
    #[error("Redis error: {0}")]
    Redis(String),

    /// MongoDB-specific error
    #[error("MongoDB error: {0}")]
    MongoDB(String),

    /// Query execution error
    #[error("Query execution failed: {0}")]
    QueryExecution(String),

    /// Connector initialization error
    #[error("Connector error: {0}")]
    Connector(String),

    /// Input validation error
    #[error("Input validation error: {0}")]
    InputValidation(String),

    /// Template substitution error
    #[error("Template error: {0}")]
    Template(String),

    /// HTTP server error
    #[error("Server error: {0}")]
    Server(String),

    /// File system error
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    /// JSON serialization/deserialization error
    #[error("JSON error: {0}")]
    Json(#[from] serde_json::Error),

    /// Query not found
    #[error("Query not found: {0}")]
    QueryNotFound(String),

    /// Adapter not found
    #[error("Adapter not found: {0}")]
    AdapterNotFound(String),

    /// Missing required input
    #[error("Missing required input: {0}")]
    MissingInput(String),

    /// Invalid input type
    #[error("Invalid input type for '{0}': expected {1}")]
    InvalidInputType(String, String),

    /// Environment variable not found
    #[error("Environment variable not found: {0}")]
    EnvVarNotFound(String),

    /// Internal error (should not happen in normal operation)
    #[error("Internal error: {0}")]
    Internal(String),
}

impl HyperterseError {
    /// Returns true if this error should be logged at error level
    pub fn is_error(&self) -> bool {
        matches!(
            self,
            HyperterseError::Database(_)
                | HyperterseError::Redis(_)
                | HyperterseError::MongoDB(_)
                | HyperterseError::Connector(_)
                | HyperterseError::Server(_)
                | HyperterseError::Internal(_)
        )
    }

    /// Returns true if this error is a client error (4xx)
    pub fn is_client_error(&self) -> bool {
        matches!(
            self,
            HyperterseError::Validation(_)
                | HyperterseError::InputValidation(_)
                | HyperterseError::QueryNotFound(_)
                | HyperterseError::AdapterNotFound(_)
                | HyperterseError::MissingInput(_)
                | HyperterseError::InvalidInputType(_, _)
        )
    }

    /// Returns the appropriate HTTP status code for this error
    pub fn status_code(&self) -> u16 {
        match self {
            HyperterseError::QueryNotFound(_) | HyperterseError::AdapterNotFound(_) => 404,
            HyperterseError::Validation(_)
            | HyperterseError::InputValidation(_)
            | HyperterseError::MissingInput(_)
            | HyperterseError::InvalidInputType(_, _) => 400,
            HyperterseError::Config(_) | HyperterseError::Template(_) => 500,
            _ => 500,
        }
    }

    /// Sanitize the error message to avoid leaking sensitive information
    pub fn sanitized_message(&self) -> String {
        match self {
            // Don't expose connection details
            HyperterseError::Database(_)
            | HyperterseError::Redis(_)
            | HyperterseError::MongoDB(_)
            | HyperterseError::Connector(_) => "Database connection error".to_string(),

            // Don't expose internal details
            HyperterseError::Internal(_) => "Internal server error".to_string(),

            // Safe to expose
            HyperterseError::QueryNotFound(name) => format!("Query not found: {}", name),
            HyperterseError::AdapterNotFound(name) => format!("Adapter not found: {}", name),
            HyperterseError::MissingInput(name) => format!("Missing required input: {}", name),
            HyperterseError::InvalidInputType(name, expected) => {
                format!("Invalid input type for '{}': expected {}", name, expected)
            }
            HyperterseError::Validation(msg) => format!("Validation error: {}", msg),
            HyperterseError::InputValidation(msg) => format!("Input validation error: {}", msg),

            // Default: use the error message
            _ => self.to_string(),
        }
    }
}

/// Result type alias using HyperterseError
pub type Result<T> = std::result::Result<T, HyperterseError>;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_error_status_codes() {
        assert_eq!(HyperterseError::QueryNotFound("test".into()).status_code(), 404);
        assert_eq!(HyperterseError::MissingInput("id".into()).status_code(), 400);
        assert_eq!(HyperterseError::Database("err".into()).status_code(), 500);
    }

    #[test]
    fn test_error_sanitization() {
        let err = HyperterseError::Database("postgres://user:password@localhost".into());
        assert_eq!(err.sanitized_message(), "Database connection error");

        let err = HyperterseError::QueryNotFound("get-users".into());
        assert_eq!(err.sanitized_message(), "Query not found: get-users");
    }

    #[test]
    fn test_error_is_client_error() {
        assert!(HyperterseError::MissingInput("id".into()).is_client_error());
        assert!(HyperterseError::QueryNotFound("test".into()).is_client_error());
        assert!(!HyperterseError::Database("err".into()).is_client_error());
    }
}
