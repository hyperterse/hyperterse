import { resolve, dirname } from "node:path";
import { mkdirSync, rmSync } from "node:fs";

const scriptDir = dirname(new URL(import.meta.url).pathname);
const projectRoot = resolve(scriptDir, "..");

const [connectorsProtoFile, primitivesProtoFile] = process.argv.slice(2);

if (!connectorsProtoFile || !primitivesProtoFile) {
  console.error(
    `Usage: bun run ${process.argv[1]} <connectors-proto-file> <primitives-proto-file>`
  );
  process.exit(1);
}

const connectorsContent = await Bun.file(resolve(projectRoot, connectorsProtoFile)).text();
const primitivesContent = await Bun.file(resolve(projectRoot, primitivesProtoFile)).text();

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

const schemaBaseURL =
  "https://raw.githubusercontent.com/hyperterse/hyperterse/refs/heads/main/schema";
const namePattern = "^[a-zA-Z][a-zA-Z0-9_-]*$";
const toolNamePattern = "^[a-z][a-z0-9_-]*$";

function inputSpecSchema(requireDefaultWhenOptional: boolean) {
  const typedDefaultRules: Array<Record<string, any>> = [
    {
      if: {
        properties: { type: { enum: ["string", "datetime"] } },
        required: ["type"],
      },
      then: { properties: { default: { type: "string" as const } } },
    },
    {
      if: {
        properties: { type: { const: "int" } },
        required: ["type"],
      },
      then: { properties: { default: { type: "integer" as const } } },
    },
    {
      if: {
        properties: { type: { const: "float" } },
        required: ["type"],
      },
      then: { properties: { default: { type: "number" as const } } },
    },
    {
      if: {
        properties: { type: { const: "boolean" } },
        required: ["type"],
      },
      then: { properties: { default: { type: "boolean" as const } } },
    },
  ];

  const allOf: Array<Record<string, any>> = [...typedDefaultRules];
  if (requireDefaultWhenOptional) {
    allOf.unshift({
      if: { properties: { optional: { const: true } } },
      then: { required: ["default"] },
    });
  }

  return {
    type: "object" as const,
    properties: {
      type: {
        type: "string" as const,
        enum: primitiveValues,
        description: "Input type",
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
        description: "Default value. Must match the configured input type if provided.",
      },
    },
    required: ["type"],
    allOf,
    additionalProperties: false,
  };
}

const rootSchema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  $id: `${schemaBaseURL}/root.terse.schema.json`,
  title: "HyperterseRootConfig",
  description:
    "Schema for root project configuration files (`.hyperterse`) that define project-level settings.",
  type: "object" as const,
  properties: {
    name: {
      type: "string" as const,
      description: "Project name.",
      pattern: toolNamePattern,
      minLength: 1,
    },
    version: {
      type: "string" as const,
      description: "Optional project version for observability metadata.",
      minLength: 1,
    },
    root: {
      type: "string" as const,
      description: "Base directory for discovery (defaults to `app`).",
      minLength: 1,
    },
    build: {
      type: "object" as const,
      description: "Optional build configuration.",
      properties: {
        out: {
          type: "string" as const,
          description: "Build output directory.",
          minLength: 1,
        },
        out_dir: {
          type: "string" as const,
          description: "Alias for build.out.",
          minLength: 1,
        },
        clean_dir: {
          type: "boolean" as const,
          description: "Clean output directory before build.",
        },
      },
      additionalProperties: false,
    },
    tools: {
      type: "object" as const,
      description: "Tool discovery and global tool cache settings.",
      properties: {
        directory: {
          type: "string" as const,
          description: "Tools directory relative to `root` (defaults to `tools`).",
          minLength: 1,
        },
        cache: {
          type: "object" as const,
          description: "Global/default tool cache configuration.",
          properties: {
            enabled: { type: "boolean" as const },
            ttl: { type: "integer" as const, minimum: 1 },
          },
          additionalProperties: false,
        },
      },
      additionalProperties: false,
    },
    adapters: {
      type: "object" as const,
      description: "Adapter discovery settings.",
      properties: {
        directory: {
          type: "string" as const,
          description: "Adapters directory relative to `root` (defaults to `adapters`).",
          minLength: 1,
        },
      },
      additionalProperties: false,
    },
    server: {
      type: "object" as const,
      description: "Runtime server configuration.",
      properties: {
        port: {
          description: "Server port (default: 8080).",
          oneOf: [
            { type: "integer" as const, minimum: 1, maximum: 65535 },
            { type: "string" as const, pattern: "^[0-9]{1,5}$" },
          ],
        },
        log_level: {
          type: "integer" as const,
          description: "Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG.",
          minimum: 1,
          maximum: 4,
        },
      },
      additionalProperties: false,
    },
  },
  required: ["name"],
  additionalProperties: false,
};

const adapterSchema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  $id: `${schemaBaseURL}/adapter.terse.schema.json`,
  title: "HyperterseAdapterConfig",
  description:
    "Schema for adapter `.terse` files. This schema validates adapter configuration regardless of folder naming convention.",
  type: "object" as const,
  properties: {
    name: {
      type: "string" as const,
      description: "Optional adapter name override. Defaults to filename when omitted.",
      pattern: namePattern,
      minLength: 1,
    },
    connector: {
      type: "string" as const,
      description: "Connector type.",
      enum: connectorValues,
    },
    connection_string: {
      type: "string" as const,
      description: "Connection string for the connector.",
      minLength: 1,
    },
    options: {
      type: "object" as const,
      description: "Connector-specific options.",
      additionalProperties: {
        type: ["string", "boolean", "number"] as const,
      },
    },
  },
  required: ["connector", "connection_string"],
  additionalProperties: false,
};

const toolSchema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  $id: `${schemaBaseURL}/tool.terse.schema.json`,
  title: "HyperterseToolConfig",
  description:
    "Schema for tool-level `.terse` files. This schema validates tool configuration regardless of folder naming convention.",
  type: "object" as const,
  properties: {
    name: {
      type: "string" as const,
      description: "Optional explicit MCP tool name.",
      pattern: toolNamePattern,
      minLength: 1,
    },
    description: {
      type: "string" as const,
      description: "Tool description exposed through MCP tools/list.",
      minLength: 1,
    },
    use: {
      description: "Adapter binding for DB-backed tools.",
      type: "string" as const,
      minLength: 1,
    },
    statement: {
      type: "string" as const,
      description: "Statement for DB-backed tools.",
      minLength: 1,
    },
    handler: {
      type: "string" as const,
      description: "Custom tool handler script path.",
      minLength: 1,
    },
    mappers: {
      type: "object" as const,
      description: "Optional mapper script paths for tool input/output transforms.",
      properties: {
        input: {
          type: "string" as const,
          description: "Input mapper script path.",
          minLength: 1,
        },
        output: {
          type: "string" as const,
          description: "Output mapper script path.",
          minLength: 1,
        },
      },
      additionalProperties: false,
    },
    auth: {
      type: "object" as const,
      description: "Optional tool auth plugin configuration.",
      properties: {
        plugin: { type: "string" as const, minLength: 1 },
        policy: {
          type: "object" as const,
          additionalProperties: { type: "string" as const },
        },
      },
      additionalProperties: false,
    },
    inputs: {
      type: "object" as const,
      patternProperties: {
        [namePattern]: inputSpecSchema(false),
      },
      additionalProperties: false,
    },
  },
  oneOf: [{ required: ["use"] }, { required: ["handler"] }],
  additionalProperties: false,
};

const outputs = [
  { fileName: "root.terse.schema.json", schema: rootSchema },
  { fileName: "adapter.terse.schema.json", schema: adapterSchema },
  { fileName: "tool.terse.schema.json", schema: toolSchema },
];

for (const output of outputs) {
  const outputPath = resolve(projectRoot, "schema", output.fileName);
  mkdirSync(dirname(outputPath), { recursive: true });
  await Bun.write(outputPath, JSON.stringify(output.schema, null, "  "));
  console.log(`✓ Generated JSON schema: schema/${output.fileName}`);
}

const legacySchemaPath = resolve(projectRoot, "schema", "terse.schema.json");
rmSync(legacySchemaPath, { force: true });
console.log("✓ Removed legacy schema: schema/terse.schema.json");
