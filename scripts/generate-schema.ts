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
const queryNamePattern = "^[a-z][a-z0-9_-]*$";

function inputSpecSchema(requireDefaultWhenOptional: boolean) {
  const typedDefaultRules = [
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

  const allOf = [...typedDefaultRules];
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
    "Schema for root project configuration files (`config.terse` or `.hyperterse`) that define project-level settings.",
  type: "object" as const,
  properties: {
    name: {
      type: "string" as const,
      description: "Project name.",
      pattern: queryNamePattern,
      minLength: 1,
    },
    version: {
      type: "string" as const,
      description: "Optional project version for observability metadata.",
      minLength: 1,
    },
    export: {
      type: "object" as const,
      description: "Optional export configuration.",
      properties: {
        out: {
          type: "string" as const,
          description: "Export output directory.",
          minLength: 1,
        },
        clean_dir: {
          type: "boolean" as const,
          description: "Clean output directory before export.",
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
        queries: {
          type: "object" as const,
          description: "Global/default query execution settings.",
          properties: {
            cache: {
              type: "object" as const,
              description: "Global/default query cache configuration.",
              properties: {
                enabled: { type: "boolean" as const },
                ttl: { type: "integer" as const, minimum: 1 },
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
    framework: {
      type: "object" as const,
      description: "Framework mode options.",
      properties: {
        app_dir: {
          type: "string" as const,
          description: "Base application directory used for route/adapters discovery.",
          minLength: 1,
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
};

const routeSchema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  $id: `${schemaBaseURL}/route.terse.schema.json`,
  title: "HyperterseRouteConfig",
  description:
    "Schema for route-level `.terse` files. This schema validates route configuration regardless of folder naming convention.",
  type: "object" as const,
  properties: {
    name: {
      type: "string" as const,
      description: "Optional explicit MCP tool name.",
      pattern: queryNamePattern,
      minLength: 1,
    },
    description: {
      type: "string" as const,
      description: "Tool description exposed through MCP tools/list.",
      minLength: 1,
    },
    use: {
      description: "Adapter binding for DB-backed route tools.",
      oneOf: [
        { type: "string" as const, minLength: 1 },
        {
          type: "array" as const,
          items: { type: "string" as const, minLength: 1 },
          minItems: 1,
        },
      ],
    },
    statement: {
      type: "string" as const,
      description: "Statement for DB-backed route tools.",
      minLength: 1,
    },
    scripts: {
      type: "object" as const,
      description: "Optional script hooks for route behavior.",
      properties: {
        handler: {
          type: "string" as const,
          description: "Custom route handler script path.",
          minLength: 1,
        },
        input_transform: {
          type: "string" as const,
          description: "Input transform script path.",
          minLength: 1,
        },
        output_transform: {
          type: "string" as const,
          description: "Output transform script path.",
          minLength: 1,
        },
      },
      additionalProperties: false,
    },
    auth: {
      type: "object" as const,
      description: "Optional route auth plugin configuration.",
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
    data: {
      type: "object" as const,
      patternProperties: {
        [namePattern]: {
          type: "object" as const,
          properties: {
            type: { type: "string" as const, enum: primitiveValues },
            description: { type: "string" as const },
            map_to: { type: "string" as const },
          },
          required: ["type"],
          additionalProperties: false,
        },
      },
      additionalProperties: false,
    },
  },
  allOf: [
    {
      anyOf: [
        { required: ["use"] },
        {
          required: ["scripts"],
          properties: {
            scripts: {
              type: "object" as const,
              required: ["handler"],
            },
          },
        },
      ],
    },
  ],
  additionalProperties: false,
};

const outputs = [
  { fileName: "root.terse.schema.json", schema: rootSchema },
  { fileName: "adapter.terse.schema.json", schema: adapterSchema },
  { fileName: "route.terse.schema.json", schema: routeSchema },
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
