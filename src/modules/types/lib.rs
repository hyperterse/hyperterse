//! Type definitions for Hyperterse
//!
//! This crate contains shared type definitions used across the Hyperterse codebase,
//! including connector types, primitive types, and runtime types.

pub mod connector;
pub mod primitive;
pub mod runtime;

pub use connector::Connector;
pub use primitive::Primitive;
