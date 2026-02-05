//! Database connector type definitions

use serde::{Deserialize, Serialize};
use std::fmt;
use std::str::FromStr;

/// Supported database connector types
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Connector {
    /// PostgreSQL database
    Postgres,
    /// MySQL database
    Mysql,
    /// Redis key-value store
    Redis,
    /// MongoDB document database
    Mongodb,
}

impl fmt::Display for Connector {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Connector::Postgres => write!(f, "postgres"),
            Connector::Mysql => write!(f, "mysql"),
            Connector::Redis => write!(f, "redis"),
            Connector::Mongodb => write!(f, "mongodb"),
        }
    }
}

impl FromStr for Connector {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "postgres" | "postgresql" => Ok(Connector::Postgres),
            "mysql" => Ok(Connector::Mysql),
            "redis" => Ok(Connector::Redis),
            "mongodb" | "mongo" => Ok(Connector::Mongodb),
            _ => Err(format!("Unknown connector type: {}", s)),
        }
    }
}

impl Connector {
    /// Returns all supported connector types
    pub fn all() -> &'static [Connector] {
        &[
            Connector::Postgres,
            Connector::Mysql,
            Connector::Redis,
            Connector::Mongodb,
        ]
    }

    /// Returns true if this connector uses SQL
    pub fn is_sql(&self) -> bool {
        matches!(self, Connector::Postgres | Connector::Mysql)
    }

    /// Returns true if this connector is a document database
    pub fn is_document(&self) -> bool {
        matches!(self, Connector::Mongodb)
    }

    /// Returns true if this connector is a key-value store
    pub fn is_key_value(&self) -> bool {
        matches!(self, Connector::Redis)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_connector_from_str() {
        assert_eq!(Connector::from_str("postgres").unwrap(), Connector::Postgres);
        assert_eq!(Connector::from_str("postgresql").unwrap(), Connector::Postgres);
        assert_eq!(Connector::from_str("mysql").unwrap(), Connector::Mysql);
        assert_eq!(Connector::from_str("redis").unwrap(), Connector::Redis);
        assert_eq!(Connector::from_str("mongodb").unwrap(), Connector::Mongodb);
        assert_eq!(Connector::from_str("mongo").unwrap(), Connector::Mongodb);
        assert!(Connector::from_str("unknown").is_err());
    }

    #[test]
    fn test_connector_display() {
        assert_eq!(Connector::Postgres.to_string(), "postgres");
        assert_eq!(Connector::Mysql.to_string(), "mysql");
        assert_eq!(Connector::Redis.to_string(), "redis");
        assert_eq!(Connector::Mongodb.to_string(), "mongodb");
    }

    #[test]
    fn test_connector_serde() {
        let json = serde_json::to_string(&Connector::Postgres).unwrap();
        assert_eq!(json, "\"postgres\"");

        let connector: Connector = serde_json::from_str("\"mysql\"").unwrap();
        assert_eq!(connector, Connector::Mysql);
    }
}
