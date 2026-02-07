//! Hidden command to generate shell completions.

use clap::{Args, CommandFactory};
use clap_complete::{generate, Shell};

/// Generate shell completion scripts.
///
/// This command is intentionally hidden from normal `--help` output because it
/// exists primarily for installers and packaging scripts.
#[derive(Args, Debug)]
pub struct CompletionCommand {
    /// Shell to generate completions for (e.g. bash, zsh)
    #[arg(value_enum)]
    pub shell: Shell,
}

impl CompletionCommand {
    pub fn execute(&self) {
        // Build the full CLI command definition and generate completion output.
        // Note: we use the lib crate's `Cli` type so the generated completions
        // stay consistent with the real CLI surface.
        let mut cmd = crate::Cli::command();
        generate(self.shell, &mut cmd, "hyperterse", &mut std::io::stdout());
    }
}

