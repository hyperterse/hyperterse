//! CLI commands

mod dev;
mod export;
mod generate;
mod init;
mod run;
mod upgrade;

pub use dev::DevCommand;
pub use export::ExportCommand;
pub use generate::{GenerateCommand, GenerateSubcommand};
pub use init::InitCommand;
pub use run::RunCommand;
pub use upgrade::UpgradeCommand;

use clap::{Parser, Subcommand};

/// Hyperterse - Type-safe query layer for databases
#[derive(Parser, Debug)]
#[command(name = "hyperterse")]
#[command(author, version, about, long_about = None)]
pub struct Cli {
    /// Configuration file path (`.terse`)
    ///
    /// This is a *global* option so it can be specified after subcommands,
    /// e.g. `hyperterse run -f config.terse`.
    #[arg(
        short = 'f',
        long = "file",
        global = true,
        default_value = "config.terse"
    )]
    pub config: String,

    /// Backwards/compat alias for `-f/--file`
    #[arg(short = 'c', long = "config", global = true, hide = true)]
    pub config_compat: Option<String>,

    /// Enable verbose logging
    #[arg(short, long, global = true)]
    pub verbose: bool,

    #[command(subcommand)]
    pub command: Commands,
}

/// Available commands
#[derive(Subcommand, Debug)]
pub enum Commands {
    /// Start the Hyperterse server
    Run(RunCommand),

    /// Start in development mode with hot reload
    Dev(DevCommand),

    /// Generate documentation or skills
    Generate(GenerateCommand),

    /// Initialize a new Hyperterse project
    Init(InitCommand),

    /// Upgrade Hyperterse to the latest version
    Upgrade(UpgradeCommand),

    /// Export configuration
    Export(ExportCommand),
}

impl Cli {
    /// Parse CLI arguments
    pub fn parse_args() -> Self {
        Self::parse()
    }

    /// Effective configuration path, accounting for compat flags.
    pub fn config_path(&self) -> &str {
        self.config_compat.as_deref().unwrap_or(&self.config)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_cli_parse() {
        // Test that CLI can parse basic commands
        let cli = Cli::try_parse_from(["hyperterse", "run"]);
        assert!(cli.is_ok());
    }

    #[test]
    fn test_cli_with_config() {
        // README-style
        let cli = Cli::try_parse_from(["hyperterse", "run", "-f", "custom.terse"]);
        assert!(cli.is_ok());
        let cli = cli.unwrap();
        assert_eq!(cli.config_path(), "custom.terse");
    }

    #[test]
    fn test_cli_with_config_compat() {
        // Compat alias
        let cli = Cli::try_parse_from(["hyperterse", "run", "-c", "custom.terse"]);
        assert!(cli.is_ok());
        let cli = cli.unwrap();
        assert_eq!(cli.config_path(), "custom.terse");
    }
}
