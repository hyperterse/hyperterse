//! Init command implementation

use clap::Args;
use hyperterse_core::HyperterseError;
use std::fs;
use std::path::Path;
use tracing::info;

/// Init command arguments
#[derive(Args, Debug)]
pub struct InitCommand {
    /// Project name
    #[arg(default_value = "my-api")]
    pub name: String,

    /// Output directory
    #[arg(short, long, default_value = ".")]
    pub output: String,

    /// Database connector type
    #[arg(short = 'd', long, default_value = "postgres")]
    pub connector: String,
}

impl InitCommand {
    /// Execute the init command
    pub async fn execute(&self) -> Result<(), HyperterseError> {
        info!("Initializing new Hyperterse project: {}", self.name);

        let output_dir = Path::new(&self.output);
        if !output_dir.exists() {
            fs::create_dir_all(output_dir)?;
        }

        // Generate config file
        let config_path = output_dir.join("config.terse");
        let config_content = self.generate_config();

        fs::write(&config_path, config_content)?;

        info!("Created: {}", config_path.display());

        // Generate .env.example file
        let env_path = output_dir.join(".env.example");
        let env_content = self.generate_env_example();

        fs::write(&env_path, env_content)?;

        info!("Created: {}", env_path.display());

        // Print instructions
        println!("\n✨ Hyperterse project initialized!");
        println!("\nNext steps:");
        println!("  1. Copy .env.example to .env and update the DATABASE_URL");
        println!("  2. Edit config.terse to add your queries");
        println!("  3. Run: hyperterse run -f config.terse");
        println!("\nFor more information, visit: https://github.com/hyperterse/hyperterse");

        Ok(())
    }

    /// Generate configuration file content
    fn generate_config(&self) -> String {
        let connector = match self.connector.to_lowercase().as_str() {
            "postgres" | "postgresql" => "postgres",
            "mysql" => "mysql",
            "redis" => "redis",
            "mongodb" | "mongo" => "mongodb",
            _ => "postgres",
        };

        let _db_url_example = match connector {
            "postgres" => "postgres://user:password@localhost:5432/database",
            "mysql" => "mysql://user:password@localhost:3306/database",
            "redis" => "redis://localhost:6379",
            "mongodb" => "mongodb://localhost:27017/database",
            _ => "postgres://user:password@localhost:5432/database",
        };

        let example_query = match connector {
            "postgres" | "mysql" => {
                r#"  get-users:
    use: main
    statement: |
      SELECT * FROM users LIMIT {{ inputs.limit }}
    description: "Get a list of users"
    inputs:
      limit:
        type: int
        optional: true
        default: 10
        description: "Maximum number of users to return""#
            }
            "redis" => {
                r#"  get-value:
    use: main
    statement: |
      GET {{ inputs.key }}
    description: "Get a value by key"
    inputs:
      key:
        type: string
        description: "The key to retrieve""#
            }
            "mongodb" => {
                r#"  find-users:
    use: main
    statement: |
      {
        "collection": "users",
        "operation": "find",
        "filter": {},
        "options": { "limit": {{ inputs.limit }} }
      }
    description: "Find users"
    inputs:
      limit:
        type: int
        optional: true
        default: 10
        description: "Maximum number of users to return""#
            }
            _ => "",
        };

        format!(
            r#"# Hyperterse Configuration (.terse)
# See https://docs.hyperterse.com/reference/configuration

name: {}

adapters:
  main:
    connector: {}
    connection_string: "{{{{ env.DATABASE_URL }}}}"

queries:
{}

server:
  port: 3000
  log_level: 1
"#,
            self.name, connector, example_query
        )
    }

    /// Generate .env.example content
    fn generate_env_example(&self) -> String {
        let db_url = match self.connector.to_lowercase().as_str() {
            "postgres" | "postgresql" => "postgres://user:password@localhost:5432/database",
            "mysql" => "mysql://user:password@localhost:3306/database",
            "redis" => "redis://localhost:6379",
            "mongodb" | "mongo" => "mongodb://localhost:27017/database",
            _ => "postgres://user:password@localhost:5432/database",
        };

        format!(
            r#"# Database connection URL
DATABASE_URL={}

# Add other environment variables here
"#,
            db_url
        )
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_config_postgres() {
        let cmd = InitCommand {
            name: "test-api".to_string(),
            output: ".".to_string(),
            connector: "postgres".to_string(),
        };

        let config = cmd.generate_config();
        assert!(config.contains("name: test-api"));
        assert!(config.contains("connector: postgres"));
    }

    #[test]
    fn test_generate_config_mongodb() {
        let cmd = InitCommand {
            name: "test-api".to_string(),
            output: ".".to_string(),
            connector: "mongodb".to_string(),
        };

        let config = cmd.generate_config();
        assert!(config.contains("connector: mongodb"));
    }
}
