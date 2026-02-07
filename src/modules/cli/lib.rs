//! Hyperterse CLI
//!
//! This crate provides the command-line interface for Hyperterse including:
//! - run: Start the server
//! - dev: Start in development mode with hot reload
//! - generate: Generate documentation and skills
//! - init: Initialize a new Hyperterse project
//! - upgrade: Upgrade Hyperterse
//! - export: Export configuration

pub mod commands;

pub use commands::{Cli, Commands};
