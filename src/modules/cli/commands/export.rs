//! Export command implementation

use clap::Args;
use hyperterse_core::HyperterseError;
use hyperterse_parser::parse_file;
use std::fs;
use std::path::Path;
use tracing::info;

/// Export command arguments
#[derive(Args, Debug)]
pub struct ExportCommand {
    /// Output directory
    #[arg(short, long, default_value = "./export")]
    pub output: String,

    /// Export format (json, yaml)
    #[arg(short, long, default_value = "json")]
    pub format: String,
}

impl ExportCommand {
    /// Execute the export command
    pub async fn execute(&self, config_path: &str) -> Result<(), HyperterseError> {
        info!("Exporting configuration from: {}", config_path);

        let model = parse_file(config_path)?;

        // Create output directory
        let output_dir = Path::new(&self.output);
        if !output_dir.exists() {
            fs::create_dir_all(output_dir)?;
        }

        // Export model configuration
        let model_file = match self.format.to_lowercase().as_str() {
            "yaml" | "yml" => {
                let path = output_dir.join("model.yaml");
                let content = serde_yaml::to_string(&model)
                    .map_err(|e| HyperterseError::Config(format!("Failed to serialize: {}", e)))?;
                fs::write(&path, content)?;
                path
            }
            _ => {
                let path = output_dir.join("model.json");
                let content = serde_json::to_string_pretty(&model)
                    .map_err(|e| HyperterseError::Config(format!("Failed to serialize: {}", e)))?;
                fs::write(&path, content)?;
                path
            }
        };

        info!("Exported model: {}", model_file.display());

        // Export OpenAPI spec
        let openapi_path = output_dir.join("openapi.json");
        let spec = Self::generate_openapi_spec(&model);
        fs::write(
            &openapi_path,
            serde_json::to_string_pretty(&spec).unwrap_or_default(),
        )?;

        info!("Exported OpenAPI: {}", openapi_path.display());

        // Export llms.txt
        let llms_path = output_dir.join("llms.txt");
        let llms_content = Self::generate_llms_txt(&model);
        fs::write(&llms_path, llms_content)?;

        info!("Exported llms.txt: {}", llms_path.display());

        println!("\n✨ Export complete!");
        println!("Files written to: {}", output_dir.display());

        Ok(())
    }

    /// Generate OpenAPI specification
    fn generate_openapi_spec(model: &hyperterse_core::Model) -> serde_json::Value {
        let mut paths = serde_json::Map::new();

        for query in &model.queries {
            let path = format!("/query/{}", query.name);

            let mut properties = serde_json::Map::new();
            let mut required = Vec::new();

            for input in &query.inputs {
                let type_str = match input.primitive_type {
                    hyperterse_types::Primitive::String => "string",
                    hyperterse_types::Primitive::Int => "integer",
                    hyperterse_types::Primitive::Float => "number",
                    hyperterse_types::Primitive::Boolean => "boolean",
                    hyperterse_types::Primitive::Uuid => "string",
                    hyperterse_types::Primitive::Datetime => "string",
                };

                let mut prop = serde_json::Map::new();
                prop.insert("type".to_string(), serde_json::json!(type_str));
                if let Some(desc) = &input.description {
                    prop.insert("description".to_string(), serde_json::json!(desc));
                }

                properties.insert(input.name.clone(), serde_json::Value::Object(prop));

                if input.required {
                    required.push(input.name.clone());
                }
            }

            paths.insert(
                path,
                serde_json::json!({
                    "post": {
                        "summary": query.description.as_deref().unwrap_or(&query.name),
                        "requestBody": {
                            "content": {
                                "application/json": {
                                    "schema": {
                                        "type": "object",
                                        "properties": {
                                            "inputs": {
                                                "type": "object",
                                                "properties": properties,
                                                "required": required
                                            }
                                        }
                                    }
                                }
                            }
                        },
                        "responses": {
                            "200": {
                                "description": "Successful response"
                            }
                        }
                    }
                }),
            );
        }

        serde_json::json!({
            "openapi": "3.0.3",
            "info": {
                "title": model.name,
                "version": "1.0.0"
            },
            "paths": paths
        })
    }

    /// Generate llms.txt content
    fn generate_llms_txt(model: &hyperterse_core::Model) -> String {
        let mut content = String::new();

        content.push_str(&format!("# {}\n\n", model.name));
        content.push_str("## Queries\n\n");

        for query in &model.queries {
            content.push_str(&format!("### {}\n", query.name));
            if let Some(desc) = &query.description {
                content.push_str(&format!("{}\n", desc));
            }
            content.push_str(&format!("POST /query/{}\n\n", query.name));
        }

        content
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_export_command_args() {
        let cmd = ExportCommand {
            output: "./out".to_string(),
            format: "yaml".to_string(),
        };
        assert_eq!(cmd.output, "./out");
        assert_eq!(cmd.format, "yaml");
    }
}
