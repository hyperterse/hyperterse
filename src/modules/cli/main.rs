//! Hyperterse CLI
//!
//! Command-line interface for the Hyperterse type-safe query layer.

use clap::Parser;
use hyperterse_cli::{Cli, Commands};
use hyperterse_core::HyperterseError;
use tracing::Level;
use tracing_subscriber::{fmt, prelude::*, EnvFilter};

#[tokio::main]
async fn main() {
    if let Err(e) = run().await {
        eprintln!("Error: {}", e);
        std::process::exit(1);
    }
}

async fn run() -> Result<(), HyperterseError> {
    let cli = Cli::parse();
    let config_path = cli.config_path().to_string();

    // Initialize logging
    let log_level = if cli.verbose {
        Level::DEBUG
    } else {
        Level::INFO
    };
    let filter = EnvFilter::builder()
        .with_default_directive(log_level.into())
        .from_env_lossy();

    tracing_subscriber::registry()
        .with(fmt::layer())
        .with(filter)
        .init();

    // Execute command
    match cli.command {
        Commands::Run(cmd) => {
            cmd.execute(&config_path).await?;
        }
        Commands::Dev(cmd) => {
            cmd.execute(&config_path).await?;
        }
        Commands::Generate(cmd) => {
            cmd.execute(&config_path).await?;
        }
        Commands::Init(cmd) => {
            cmd.execute().await?;
        }
        Commands::Upgrade(cmd) => {
            cmd.execute().await?;
        }
        Commands::Export(cmd) => {
            cmd.execute(&config_path).await?;
        }
        Commands::Completion(cmd) => {
            cmd.execute();
        }
    }

    Ok(())
}
