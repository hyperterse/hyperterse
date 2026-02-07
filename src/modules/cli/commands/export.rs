//! Export command implementation
//!
//! Produces a self-contained launcher script (config + version embedded), Dockerfile,
//! docker-compose.yml, and a copy of the config file for deployment.

use base64::Engine;
use clap::Args;
use hyperterse_core::HyperterseError;
use hyperterse_parser::parse_file;
use std::fs;
use std::path::Path;
use tracing::{debug, info};

const GITHUB_REPO: &str = env!("CARGO_PKG_REPOSITORY");
const HYPERTERSE_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Export command arguments
#[derive(Args, Debug)]
pub struct ExportCommand {
    /// Output directory
    #[arg(short = 'o', long, default_value = "dist")]
    pub out: String,

    /// Clean output directory before exporting
    #[arg(long)]
    pub clean_dir: bool,
}

impl ExportCommand {
    /// Execute the export command
    pub async fn execute(&self, config_path: &str) -> Result<(), HyperterseError> {
        info!("Exporting configuration from: {}", config_path);

        let model = parse_file(config_path)?;
        let config_bytes = fs::read(config_path)
            .map_err(|e| HyperterseError::Config(format!("Failed to read config file: {}", e)))?;
        let config_content = String::from_utf8_lossy(&config_bytes);

        let out_dir = self
            .out
            .as_str()
            .strip_suffix('/')
            .unwrap_or(self.out.as_str());
        let out_path = Path::new(out_dir);
        let docker_out_path = out_path.join("container");

        if self.clean_dir && out_path.exists() {
            fs::remove_dir_all(out_path)?;
        }
        fs::create_dir_all(out_path)?;
        fs::create_dir_all(&docker_out_path)?;

        let script_name = Self::script_name(&model.name);
        let launcher_path = out_path.join(&script_name);
        let launcher = Self::generate_launcher_script(&config_content, &model.name);
        fs::write(&launcher_path, launcher)?;
        #[cfg(unix)]
        {
            use std::os::unix::fs::PermissionsExt;
            let mut perms = fs::metadata(&launcher_path)?.permissions();
            perms.set_mode(0o755);
            fs::set_permissions(&launcher_path, perms)?;
        }
        debug!("Wrote launcher: {}", launcher_path.display());

        let config_dest = out_path.join("config.terse");
        fs::write(&config_dest, &config_bytes)?;
        debug!("Wrote config: {}", config_dest.display());

        // Also copy config into container/ so Docker build context is self-contained
        let docker_config_dest = docker_out_path.join("config.terse");
        fs::write(&docker_config_dest, &config_bytes)?;

        let dockerfile_path = docker_out_path.join("Dockerfile");
        fs::write(&dockerfile_path, Self::generate_dockerfile(&model.name))?;
        debug!("Wrote Dockerfile: {}", dockerfile_path.display());

        let compose_path = docker_out_path.join("docker-compose.yml");
        fs::write(&compose_path, Self::generate_docker_compose(&model.name))?;
        debug!("Wrote docker-compose.yml: {}", compose_path.display());

        info!("✨ Export complete!");
        debug!("Files written to: {}", out_path.display());
        debug!(
            "  {} - launcher script (run with: ./{})",
            script_name, script_name
        );
        debug!("  config.terse");
        debug!("  container/config.terse");
        debug!("  container/Dockerfile");
        debug!("  container/docker-compose.yml");

        Ok(())
    }

    /// Sanitize config name for use as script filename (no path, no extension)
    pub(crate) fn script_name(name: &str) -> String {
        name.chars()
            .map(|c| {
                if c.is_alphanumeric() || c == '-' || c == '_' {
                    c
                } else {
                    '_'
                }
            })
            .collect::<String>()
    }

    /// Generate the self-contained launcher script.
    /// Cache path: /usr/local/hyperterse/cache/{version}/bin/hyperterse
    fn generate_launcher_script(config_content: &str, _name: &str) -> String {
        let encoded = base64::engine::general_purpose::STANDARD.encode(config_content.as_bytes());
        let version = HYPERTERSE_VERSION;
        let repo = GITHUB_REPO.trim_end_matches('/');
        let repo_owner_name = repo
            .strip_prefix("https://github.com/")
            .unwrap_or(repo)
            .trim_end_matches('/');

        format!(
            r#"#!/usr/bin/env bash
set -euo pipefail
VERSION="{version}"
REPO="{repo_owner_name}"
CACHE_DIR="/usr/local/hyperterse/cache/${{VERSION}}/bin"
BINARY="${{CACHE_DIR}}/hyperterse"

case "$(uname -s)" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $(uname -m)" >&2; exit 1 ;;
esac

if [ ! -x "$BINARY" ]; then
  mkdir -p "$CACHE_DIR"
  URL="https://github.com/{repo_owner_name}/releases/download/v${{VERSION}}/hyperterse-${{OS}}-${{ARCH}}"
  if command -v curl >/dev/null 2>&1; then
    curl -fSL -o "$BINARY" "$URL"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "$BINARY" "$URL"
  else
    echo "Need curl or wget to download Hyperterse binary" >&2
    exit 1
  fi
  chmod +x "$BINARY"
fi

CONFIG=$(base64 -d <<'__HYPERTERSE_CONFIG__'
{encoded}
__HYPERTERSE_CONFIG__
)
exec "$BINARY" run --source "$CONFIG" "$@"
"#,
            version = version,
            repo_owner_name = repo_owner_name,
            encoded = encoded,
        )
    }

    fn generate_dockerfile(_name: &str) -> String {
        let version = HYPERTERSE_VERSION;
        let repo = GITHUB_REPO
            .strip_prefix("https://github.com/")
            .unwrap_or("hyperterse/hyperterse")
            .trim_end_matches('/');

        format!(
            r#"# Multi-arch: build with docker buildx (e.g. --platform linux/amd64,linux/arm64)
ARG TARGETOS=linux
ARG TARGETARCH=amd64

FROM debian:bookworm-slim AS runner
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

ARG TARGETOS
ARG TARGETARCH
ENV HYPERTERSE_VERSION={version}
RUN ARCH=$(case "$TARGETARCH" in amd64) echo amd64;; arm64) echo arm64;; *) echo amd64;; esac) \
    && mkdir -p /usr/local/hyperterse/cache/$HYPERTERSE_VERSION/bin \
    && curl -fSL -o /usr/local/hyperterse/cache/$HYPERTERSE_VERSION/bin/hyperterse \
    "https://github.com/{repo}/releases/download/v$HYPERTERSE_VERSION/hyperterse-$TARGETOS-$ARCH" \
    && chmod +x /usr/local/hyperterse/cache/$HYPERTERSE_VERSION/bin/hyperterse

WORKDIR /app
COPY config.terse /app/config.terse
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/bin/sh", "-c", "exec /usr/local/hyperterse/cache/$HYPERTERSE_VERSION/bin/hyperterse run -f /app/config.terse \"$@\"", "--"]
CMD []
"#
        )
    }

    fn generate_docker_compose(name: &str) -> String {
        let service_name = name
            .chars()
            .map(|c| {
                if c.is_alphanumeric() || c == '-' || c == '_' {
                    c
                } else {
                    '_'
                }
            })
            .collect::<String>();
        format!(
            r#"name: {service_name}

services:
  hyperterse:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
    env_file:
      - required: false
        path: .env
    restart: unless-stopped
"#,
            service_name = service_name,
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_export_command_args() {
        let cmd = ExportCommand {
            out: "dist".to_string(),
            clean_dir: false,
        };
        assert_eq!(cmd.out, "dist");
        assert!(!cmd.clean_dir);
    }

    #[test]
    fn test_script_name_sanitize() {
        assert_eq!(ExportCommand::script_name("my-api"), "my-api");
        assert_eq!(ExportCommand::script_name("my_api"), "my_api");
        assert_eq!(ExportCommand::script_name("my api"), "my_api");
    }
}
