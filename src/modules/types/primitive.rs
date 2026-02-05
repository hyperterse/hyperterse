//! Primitive type definitions for input validation

use serde::{Deserialize, Serialize};
use std::fmt;
use std::str::FromStr;

/// Supported primitive types for query inputs
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Primitive {
    /// String type
    String,
    /// Integer type (i64)
    Int,
    /// Floating point type (f64)
    Float,
    /// Boolean type
    Boolean,
    /// UUID type
    Uuid,
    /// DateTime type (ISO 8601)
    Datetime,
}

impl fmt::Display for Primitive {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Primitive::String => write!(f, "string"),
            Primitive::Int => write!(f, "int"),
            Primitive::Float => write!(f, "float"),
            Primitive::Boolean => write!(f, "boolean"),
            Primitive::Uuid => write!(f, "uuid"),
            Primitive::Datetime => write!(f, "datetime"),
        }
    }
}

impl FromStr for Primitive {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "string" | "str" => Ok(Primitive::String),
            "int" | "integer" | "i64" => Ok(Primitive::Int),
            "float" | "f64" | "double" => Ok(Primitive::Float),
            "boolean" | "bool" => Ok(Primitive::Boolean),
            "uuid" => Ok(Primitive::Uuid),
            "datetime" | "timestamp" => Ok(Primitive::Datetime),
            _ => Err(format!("Unknown primitive type: {}", s)),
        }
    }
}

impl Primitive {
    /// Returns all supported primitive types
    pub fn all() -> &'static [Primitive] {
        &[
            Primitive::String,
            Primitive::Int,
            Primitive::Float,
            Primitive::Boolean,
            Primitive::Uuid,
            Primitive::Datetime,
        ]
    }

    /// Validates a JSON value against this primitive type
    pub fn validate(&self, value: &serde_json::Value) -> bool {
        match self {
            Primitive::String => value.is_string(),
            Primitive::Int => value.is_i64() || value.is_u64(),
            Primitive::Float => value.is_f64() || value.is_i64() || value.is_u64(),
            Primitive::Boolean => value.is_boolean(),
            Primitive::Uuid => {
                value.as_str().map(|s| uuid::Uuid::parse_str(s).is_ok()).unwrap_or(false)
            }
            Primitive::Datetime => {
                value.as_str().map(|s| {
                    chrono::DateTime::parse_from_rfc3339(s).is_ok()
                        || chrono::NaiveDateTime::parse_from_str(s, "%Y-%m-%d %H:%M:%S").is_ok()
                }).unwrap_or(false)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn test_primitive_from_str() {
        assert_eq!(Primitive::from_str("string").unwrap(), Primitive::String);
        assert_eq!(Primitive::from_str("str").unwrap(), Primitive::String);
        assert_eq!(Primitive::from_str("int").unwrap(), Primitive::Int);
        assert_eq!(Primitive::from_str("integer").unwrap(), Primitive::Int);
        assert_eq!(Primitive::from_str("float").unwrap(), Primitive::Float);
        assert_eq!(Primitive::from_str("boolean").unwrap(), Primitive::Boolean);
        assert_eq!(Primitive::from_str("bool").unwrap(), Primitive::Boolean);
        assert_eq!(Primitive::from_str("uuid").unwrap(), Primitive::Uuid);
        assert_eq!(Primitive::from_str("datetime").unwrap(), Primitive::Datetime);
        assert!(Primitive::from_str("unknown").is_err());
    }

    #[test]
    fn test_primitive_display() {
        assert_eq!(Primitive::String.to_string(), "string");
        assert_eq!(Primitive::Int.to_string(), "int");
        assert_eq!(Primitive::Float.to_string(), "float");
        assert_eq!(Primitive::Boolean.to_string(), "boolean");
        assert_eq!(Primitive::Uuid.to_string(), "uuid");
        assert_eq!(Primitive::Datetime.to_string(), "datetime");
    }

    #[test]
    fn test_primitive_validate() {
        assert!(Primitive::String.validate(&json!("hello")));
        assert!(!Primitive::String.validate(&json!(123)));

        assert!(Primitive::Int.validate(&json!(42)));
        assert!(!Primitive::Int.validate(&json!("42")));

        assert!(Primitive::Float.validate(&json!(3.14)));
        assert!(Primitive::Float.validate(&json!(42))); // int is also valid as float

        assert!(Primitive::Boolean.validate(&json!(true)));
        assert!(!Primitive::Boolean.validate(&json!("true")));
    }
}
