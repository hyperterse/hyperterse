import { resolve, dirname } from "node:path";
import { mkdirSync } from "node:fs";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
const projectRoot = resolve(scriptDir, "..");

const [connectorsProtoFile, primitivesProtoFile] = process.argv.slice(2);

if (!connectorsProtoFile || !primitivesProtoFile) {
  console.error(
    `Usage: bun run ${process.argv[1]} <connectors-proto-file> <primitives-proto-file>`
  );
  process.exit(1);
}

// Read proto files
const connectorsContent = await Bun.file(resolve(projectRoot, connectorsProtoFile)).text();
const primitivesContent = await Bun.file(resolve(projectRoot, primitivesProtoFile)).text();

// Parse enum values from proto file content
function parseEnumValues(content: string, enumName: string): string[] {
  const values: string[] = [];

  const enumPattern = new RegExp(`enum\\s+${enumName}\\s*\\{([^}]+)\\}`, "s");
  const match = content.match(enumPattern);
  if (!match?.[1]) return values;

  const enumBody = match[1];
  const valuePattern = /(\w+)\s*=\s*\d+\s*;/g;
  let valueMatch: RegExpExecArray | null;

  while ((valueMatch = valuePattern.exec(enumBody)) !== null) {
    const protoName = valueMatch[1];
    if (protoName.includes("UNSPECIFIED")) continue;

    // Convert CONNECTOR_POSTGRES -> postgres, PRIMITIVE_STRING -> string
    const prefix = enumName.toUpperCase() + "_";
    let stringVal = protoName;
    if (stringVal.startsWith(prefix)) {
      stringVal = stringVal.slice(prefix.length);
    }
    values.push(stringVal.toLowerCase());
  }

  return values;
}

const connectorValues = parseEnumValues(connectorsContent, "Connector");
const primitiveValues = parseEnumValues(primitivesContent, "Primitive");

// Name pattern: allows any casing (camelCase, PascalCase, snake_case, etc.)
const namePattern = "^[a-zA-Z][a-zA-Z0-9_-]*$";
// Query name pattern: must start with a letter, lowercase only
const queryNamePattern = "^[a-z][a-z0-9_-]*$";

// Generate JSON schema
const schema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  $id: "https://raw.githubusercontent.com/hyperterse/hyperterse/refs/heads/main/schema/terse.schema.json",
  title: "Terselang",
  description: "JSON schema for Hyperterse configuration files.",
  type: "object" as const,
  properties: {
    name: {
      type: "string" as const,
      description: "Configuration name (required) - must be lower-kebab-case or lower_snake_case",
      pattern: queryNamePattern,
      minLength: 1,
    },
    export: {
      type: "object" as const,
      description: "Optional export configuration",
      properties: {
        out: {
          type: "string" as const,
          description: "Output directory path (script filename uses config name)",
          minLength: 1,
        },
        clean_dir: {
          type: "boolean" as const,
          description: "Clean output directory before exporting (default: false)",
        },
      },
      additionalProperties: false,
    },
    server: {
      type: "object" as const,
      description: "Optional server configuration",
      properties: {
        port: {
          type: "integer" as const,
          description: "Server port (default: 8080)",
          minimum: 1,
          maximum: 65535,
        },
        log_level: {
          type: "integer" as const,
          description: "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (default: 3)",
          minimum: 1,
          maximum: 4,
        },
        queries: {
          type: "object" as const,
          description: "Optional query execution defaults",
          properties: {
            cache: {
              type: "object" as const,
              description: "Global/default query cache settings",
              properties: {
                enabled: {
                  type: "boolean" as const,
                  description: "Enable executor-level query caching",
                },
                ttl: {
                  type: "integer" as const,
                  description: "Cache TTL in seconds",
                  minimum: 1,
                },
              },
              required: ["enabled"],
              additionalProperties: false,
            },
          },
          additionalProperties: false,
        },
      },
      additionalProperties: false,
    },
    adapters: {
      type: "object" as const,
      description: "Adapter configurations (required, must have at least one entry)",
      minProperties: 1,
      patternProperties: {
        [namePattern]: {
          type: "object" as const,
          properties: {
            connector: {
              type: "string" as const,
              description: "Connector type (required)",
              enum: connectorValues,
            },
            connection_string: {
              type: "string" as const,
              description: "Database connection string (required)",
              minLength: 1,
            },
            options: {
              type: "object" as const,
              description: "Connector-specific options",
              additionalProperties: {
                oneOf: [
                  { type: "string" as const },
                  { type: "boolean" as const },
                  { type: "number" as const },
                ],
              },
            },
          },
          required: ["connector", "connection_string"],
          additionalProperties: false,
        },
      },
      additionalProperties: false,
    },
    queries: {
      type: "object" as const,
      description: "Query definitions (required, must have at least one entry)",
      minProperties: 1,
      patternProperties: {
        [queryNamePattern]: {
          type: "object" as const,
          properties: {
            use: {
              description: "References to adapter names (required, must reference valid adapters)",
              oneOf: [
                { type: "string" as const, minLength: 1 },
                {
                  type: "array" as const,
                  items: { type: "string" as const, minLength: 1 },
                  minItems: 1,
                },
              ],
            },
            description: {
              type: "string" as const,
              description: "Query description (required)",
              minLength: 1,
            },
            statement: {
              type: "string" as const,
              description: "SQL or command string (required)",
              minLength: 1,
            },
            cache: {
              type: "object" as const,
              description: "Optional per-query cache override",
              properties: {
                enabled: {
                  type: "boolean" as const,
                  description: "Enable or disable cache for this query",
                },
                ttl: {
                  type: "integer" as const,
                  description: "Per-query cache TTL in seconds",
                  minimum: 1,
                },
              },
              required: ["enabled"],
              additionalProperties: false,
            },
            inputs: {
              type: "object" as const,
              description: "Input parameter definitions",
              patternProperties: {
                [namePattern]: {
                  type: "object" as const,
                  properties: {
                    type: {
                      type: "string" as const,
                      description: "Input type (required)",
                      enum: primitiveValues,
                    },
                    description: {
                      type: "string" as const,
                      description: "Input description",
                    },
                    optional: {
                      type: "boolean" as const,
                      description: "Whether the input is optional",
                    },
                    default: {
                      description:
                        "Default value (required if optional=true, type depends on input type)",
                    },
                  },
                  required: ["type"],
                  allOf: [
                    {
                      if: {
                        properties: { optional: { const: true } },
                      },
                      then: {
                        required: ["default"],
                      },
                    },
                    {
                      if: {
                        properties: {
                          type: { enum: ["string", "datetime"] },
                        },
                        required: ["type"],
                      },
                      then: {
                        properties: {
                          default: {
                            description: "Default value (required if optional=true)",
                            type: "string" as const,
                          },
                        },
                      },
                    },
                    {
                      if: {
                        properties: { type: { const: "int" } },
                        required: ["type"],
                      },
                      then: {
                        properties: {
                          default: {
                            description: "Default value (required if optional=true)",
                            type: "integer" as const,
                          },
                        },
                      },
                    },
                    {
                      if: {
                        properties: { type: { const: "float" } },
                        required: ["type"],
                      },
                      then: {
                        properties: {
                          default: {
                            description: "Default value (required if optional=true)",
                            type: "number" as const,
                          },
                        },
                      },
                    },
                    {
                      if: {
                        properties: { type: { const: "boolean" } },
                        required: ["type"],
                      },
                      then: {
                        properties: {
                          default: {
                            description: "Default value (required if optional=true)",
                            type: "boolean" as const,
                          },
                        },
                      },
                    },
                  ],
                  additionalProperties: false,
                },
              },
              additionalProperties: false,
            },
          },
          required: ["use", "description", "statement"],
          additionalProperties: false,
        },
      },
      additionalProperties: false,
    },
  },
  required: ["name", "adapters", "queries"],
  additionalProperties: false,
  $comment:
    "Cross-reference validations: (1) The 'use' field in queries must reference valid adapter names defined in 'adapters'. (2) Statement input references ({{ inputs.x }}) must match defined inputs. These validations are performed by core/parser/validator.go. Standard JSON Schema Draft 07 doesn't support dynamic cross-references, but some validators (e.g., AJV with $data) can validate this.",
};

// Write to file
const outputPath = resolve(projectRoot, "schema", "terse.schema.json");
mkdirSync(dirname(outputPath), { recursive: true });
await Bun.write(outputPath, JSON.stringify(schema, null, "  "));

console.log(`✓ Generated JSON schema: schema/terse.schema.json`);
