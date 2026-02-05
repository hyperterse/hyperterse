//! Run command implementation

use clap::Args;
use hyperterse_core::HyperterseError;
use hyperterse_parser::parse_file;
use hyperterse_runtime::Runtime;
use tracing::info;

/// Run command arguments
#[derive(Args, Debug)]
pub struct RunCommand {
    /// Override server port
    #[arg(short, long)]
    pub port: Option<u16>,
}

impl RunCommand {
    /// Execute the run command
    pub async fn execute(&self, config_path: &str) -> Result<(), HyperterseError> {
        info!("Loading configuration from: {}", config_path);

        // Parse configuration
        let model = parse_file(config_path)?;

        // Create and run the runtime (port override handled by runtime)
        let runtime = Runtime::with_port_override(model, self.port).await?;
        runtime.run().await?;

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_run_command_args() {
        let cmd = RunCommand { port: Some(8080) };
        assert_eq!(cmd.port, Some(8080));
    }
}
