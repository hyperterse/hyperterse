//! Dev command implementation with hot reload

use clap::Args;
use hyperterse_core::HyperterseError;
use hyperterse_parser::parse_file;
use hyperterse_runtime::Runtime;
use notify::{Config, Event, RecommendedWatcher, RecursiveMode, Watcher};
use std::path::Path;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::mpsc;
use tokio::sync::RwLock;
use tracing::{error, info, warn};

/// Dev command arguments
#[derive(Args, Debug)]
pub struct DevCommand {
    /// Override server port
    #[arg(short, long)]
    pub port: Option<u16>,

    /// Debounce delay in milliseconds for file changes
    #[arg(long, default_value = "500")]
    pub debounce: u64,
}

impl DevCommand {
    /// Execute the dev command with file watching
    pub async fn execute(&self, config_path: &str) -> Result<(), HyperterseError> {
        info!("Starting development mode with hot reload");
        info!("Watching: {}", config_path);

        // Parse initial configuration
        let model = parse_file(config_path)?;

        // Create runtime (port override handled internally, preserved across reloads)
        let runtime = Arc::new(RwLock::new(
            Runtime::with_port_override(model, self.port).await?,
        ));

        // Set up file watcher
        let (tx, mut rx) = mpsc::channel::<()>(10);
        let config_path_owned = config_path.to_string();
        let debounce_ms = self.debounce;

        // Spawn file watcher
        let watcher_handle = tokio::task::spawn_blocking(move || {
            let rt = tokio::runtime::Handle::current();
            let tx = tx.clone();

            let mut watcher = match RecommendedWatcher::new(
                move |result: Result<Event, notify::Error>| {
                    if let Ok(event) = result {
                        if event.kind.is_modify() {
                            let _ = rt.block_on(async { tx.send(()).await });
                        }
                    }
                },
                Config::default().with_poll_interval(Duration::from_millis(500)),
            ) {
                Ok(w) => w,
                Err(e) => {
                    error!("Failed to create file watcher: {}", e);
                    return;
                }
            };

            if let Err(e) =
                watcher.watch(Path::new(&config_path_owned), RecursiveMode::NonRecursive)
            {
                error!("Failed to watch file: {}", e);
                return;
            }

            // Keep watcher alive
            loop {
                std::thread::sleep(Duration::from_secs(1));
            }
        });

        // Spawn reload handler
        let runtime_clone = runtime.clone();
        let config_path_reload = config_path.to_string();
        let reload_handle = tokio::spawn(async move {
            let mut last_reload = std::time::Instant::now();

            while rx.recv().await.is_some() {
                // Debounce
                if last_reload.elapsed() < Duration::from_millis(debounce_ms) {
                    continue;
                }

                info!("Configuration file changed, reloading...");

                match parse_file(&config_path_reload) {
                    Ok(model) => {
                        // Port override is preserved internally by Runtime
                        let mut runtime = runtime_clone.write().await;
                        if let Err(e) = runtime.reload(model).await {
                            error!("Failed to reload configuration: {}", e);
                        } else {
                            info!("Configuration reloaded successfully");
                            last_reload = std::time::Instant::now();
                        }
                    }
                    Err(e) => {
                        warn!("Failed to parse configuration: {}", e);
                        warn!("Server continues with previous configuration");
                    }
                }
            }
        });

        // Run the server
        {
            let runtime = runtime.read().await;
            runtime.run().await?;
        }

        // Cleanup
        watcher_handle.abort();
        reload_handle.abort();

        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_dev_command_args() {
        let cmd = DevCommand {
            port: Some(3000),
            debounce: 1000,
        };
        assert_eq!(cmd.port, Some(3000));
        assert_eq!(cmd.debounce, 1000);
    }
}
