//! Core domain logic for Hyperterse
//!
//! This crate contains the core domain models, business logic, and error types
//! for the Hyperterse query layer.

pub mod domain;
pub mod error;

pub use domain::*;
pub use error::HyperterseError;
