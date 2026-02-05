#!/usr/bin/env bun
/**
 * Generate JSON Schema for `.terse` config files.
 *
 * This schema is used by editors (e.g., VS Code YAML) for validation/autocomplete.
 * It should be regenerated whenever the config structs/enums change.
 *
 * Usage:
 *   bun run scripts/generate-schema.ts
 *   bun run scripts/generate-schema.ts --out schema/terse.schema.json
 */

import { readFile, writeFile } from "fs/promises";
import { dirname, resolve } from "path";
import { mkdir } from "fs/promises";
import { parseArgs } from "util";

const CONNECTOR_RS = "src/modules/types/connector.rs";
const PRIMITIVE_RS = "src/modules/types/primitive.rs";

function extractEnumVariants(source: string, enumName: string): string[] {
  const enumStart = source.indexOf(`pub enum ${enumName}`);
  if (enumStart === -1) return [];

  const braceStart = source.indexOf("{", enumStart);
  if (braceStart === -1) return [];

  let i = braceStart + 1;
  let depth = 1;
  for (; i < source.length; i++) {
    const ch = source[i];
    if (ch === "{") depth++;
    else if (ch === "}") {
      depth--;
      if (depth === 0) break;
    }
  }
  const block = source.slice(braceStart + 1, i);

  const variants: string[] = [];
  for (const line of block.split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("///") || trimmed.startsWith("//")) continue;
    const m = trimmed.match(/^([A-Za-z_][A-Za-z0-9_]*)\s*,?\s*$/);
    if (m) variants.push(m[1]);
  }
  return variants;
}

function toLowercaseSerdeVariant(variant: string): string {
  // With `#[serde(rename_all = "lowercase")]`, variants map to lowercase strings.
  return variant.toLowerCase();
}

function jsonSchema(): Record<string, unknown> {
  // Keep patterns aligned with `src/modules/parser/validator.rs`.
  const NAME_PATTERN = "^[a-z][a-z0-9]*(?:[-_][a-z0-9]+)*$";
  const IDENT_PATTERN = "^[A-Za-z_][A-Za-z0-9_-]*$";

  return {
    $schema: "http://json-schema.org/draft-07/schema#",
    $id: "https://raw.githubusercontent.com/hyperterse/hyperterse/refs/heads/main/schema/terse.schema.json",
    $comment:
      "Some validations (cross-references, placeholder checks) are enforced at runtime (see src/modules/parser/validator.rs).",
    title: "Terse",
    type: "object",
    additionalProperties: false,
    required: ["name"],
    properties: {
      name: {
        type: "string",
        minLength: 1,
        pattern: NAME_PATTERN,
        description: "Configuration name (lower-kebab-case or lower_snake_case).",
      },
      adapters: {
        description:
          "Database adapters. Supported in both legacy `.terse` (map) and current (array) formats.",
        oneOf: [
          {
            type: "array",
            default: [],
            items: { $ref: "#/$defs/adapter" },
          },
          {
            type: "object",
            default: {},
            propertyNames: { type: "string", pattern: NAME_PATTERN },
            additionalProperties: { $ref: "#/$defs/adapterInline" },
          },
        ],
      },
      queries: {
        description:
          "Query definitions. Supported in both legacy `.terse` (map) and current (array) formats.",
        oneOf: [
          {
            type: "array",
            default: [],
            items: { $ref: "#/$defs/query" },
          },
          {
            type: "object",
            default: {},
            propertyNames: { type: "string", pattern: NAME_PATTERN },
            additionalProperties: { $ref: "#/$defs/queryInline" },
          },
        ],
      },
      server: {
        type: "object",
        additionalProperties: false,
        description: "Optional server configuration.",
        properties: {
          port: {
            description:
              "Port to listen on (accepts number or string for env-substitution).",
            anyOf: [{ type: "integer", minimum: 1, maximum: 65535 }, { type: "string" }],
          },
          log_level: {
            type: "integer",
            minimum: 0,
            description:
              "Log level (implementation-defined). Not strictly validated by schema.",
          },
          pool: { $ref: "#/$defs/poolConfig" },
        },
      },
      export: {
        type: "object",
        additionalProperties: false,
        description: "Optional export configuration.",
        properties: {
          base_url: { type: "string" },
          output_dir: { type: "string" },
          out: {
            type: "string",
            description: "Legacy alias for output_dir used in older `.terse` configs.",
          },
        },
      },
    },
    $defs: {
      adapter: {
        type: "object",
        additionalProperties: false,
        required: ["name", "connector", "url"],
        properties: {
          name: {
            type: "string",
            minLength: 1,
            pattern: NAME_PATTERN,
            description: "Adapter name.",
          },
          connector: {
            type: "string",
            enum: [], // filled in at runtime
            description: "Connector type.",
          },
          url: {
            type: "string",
            minLength: 1,
            description: "Connection URL (supports {{ env.VAR }} substitution).",
          },
        },
      },
      adapterInline: {
        type: "object",
        additionalProperties: false,
        required: ["connector"],
        properties: {
          connector: {
            type: "string",
            enum: [], // filled in at runtime
            description: "Connector type.",
          },
          connection_string: {
            type: "string",
            minLength: 1,
            description:
              "Connection string (legacy field name, preferred in `.terse` examples).",
          },
          url: {
            type: "string",
            minLength: 1,
            description: "Connection URL (alias for connection_string).",
          },
          options: {
            description:
              "Connector-specific options (currently ignored by the Rust core model).",
          },
        },
        anyOf: [
          { required: ["connection_string"] },
          { required: ["url"] },
        ],
      },
      query: {
        type: "object",
        additionalProperties: false,
        required: ["name", "adapter", "statement"],
        properties: {
          name: {
            type: "string",
            minLength: 1,
            pattern: NAME_PATTERN,
            description: "Query name (becomes endpoint path).",
          },
          adapter: {
            type: "string",
            minLength: 1,
            pattern: NAME_PATTERN,
            description: "Adapter name to execute against.",
          },
          statement: {
            type: "string",
            minLength: 1,
            description: "SQL / Redis command / MongoDB JSON statement.",
          },
          description: { type: "string" },
          inputs: {
            description:
              "Input parameters. Supported in both legacy `.terse` (map) and current (array) formats.",
            oneOf: [
              {
                type: "array",
                default: [],
                items: { $ref: "#/$defs/input" },
              },
              {
                type: "object",
                default: {},
                propertyNames: { type: "string", pattern: IDENT_PATTERN },
                additionalProperties: { $ref: "#/$defs/inputInline" },
              },
            ],
          },
        },
      },
      queryInline: {
        type: "object",
        additionalProperties: false,
        required: ["statement"],
        properties: {
          use: {
            type: "string",
            minLength: 1,
            pattern: NAME_PATTERN,
            description: "Adapter name to execute against (legacy field name).",
          },
          adapter: {
            type: "string",
            minLength: 1,
            pattern: NAME_PATTERN,
            description: "Adapter name to execute against.",
          },
          statement: {
            type: "string",
            minLength: 1,
            description: "SQL / Redis command / MongoDB JSON statement.",
          },
          description: { type: "string" },
          inputs: {
            description:
              "Input parameters. Map form is recommended for `.terse` configs.",
            oneOf: [
              {
                type: "array",
                default: [],
                items: { $ref: "#/$defs/input" },
              },
              {
                type: "object",
                default: {},
                propertyNames: { type: "string", pattern: IDENT_PATTERN },
                additionalProperties: { $ref: "#/$defs/inputInline" },
              },
            ],
          },
        },
        anyOf: [{ required: ["use"] }, { required: ["adapter"] }],
      },
      input: {
        type: "object",
        additionalProperties: false,
        required: ["name", "type"],
        properties: {
          name: {
            type: "string",
            minLength: 1,
            pattern: IDENT_PATTERN,
            description: "Input parameter name (referenced as {{ inputs.<name> }}).",
          },
          type: {
            type: "string",
            enum: [], // filled in at runtime
            description: "Input primitive type.",
          },
          required: {
            type: "boolean",
            default: true,
            description: "Whether this input is required (default: true).",
          },
          default: {
            description: "Default value for optional inputs (type must match `type`).",
          },
          description: { type: "string" },
        },
        allOf: [
          {
            if: {
              properties: { required: { const: false } },
              required: ["required"],
            },
            then: {
              required: ["default"],
            },
          },
          {
            if: { properties: { type: { const: "string" } }, required: ["type"] },
            then: { properties: { default: { type: "string" } } },
          },
          {
            if: { properties: { type: { const: "datetime" } }, required: ["type"] },
            then: { properties: { default: { type: "string" } } },
          },
          {
            if: { properties: { type: { const: "uuid" } }, required: ["type"] },
            then: { properties: { default: { type: "string" } } },
          },
          {
            if: { properties: { type: { const: "int" } }, required: ["type"] },
            then: { properties: { default: { type: "integer" } } },
          },
          {
            if: { properties: { type: { const: "float" } }, required: ["type"] },
            then: { properties: { default: { type: "number" } } },
          },
          {
            if: { properties: { type: { const: "boolean" } }, required: ["type"] },
            then: { properties: { default: { type: "boolean" } } },
          },
        ],
      },
      inputInline: {
        type: "object",
        additionalProperties: false,
        required: ["type"],
        properties: {
          type: {
            type: "string",
            enum: [], // filled in at runtime
            description: "Input primitive type.",
          },
          description: { type: "string" },
          optional: {
            type: "boolean",
            description:
              "Legacy flag. If true, the input is considered optional (and must provide a default).",
          },
          required: {
            type: "boolean",
            description:
              "If set, overrides `optional`. If false, a `default` must be provided.",
          },
          default: {
            description:
              "Default value for optional inputs (type must match `type`).",
          },
        },
        allOf: [
          {
            if: {
              properties: { required: { const: false } },
              required: ["required"],
            },
            then: { required: ["default"] },
          },
          {
            if: {
              properties: { optional: { const: true } },
              required: ["optional"],
            },
            then: { required: ["default"] },
          },
        ],
      },
      poolConfig: {
        type: "object",
        additionalProperties: false,
        properties: {
          max_connections: { type: "integer", minimum: 1 },
          min_connections: { type: "integer", minimum: 0 },
          acquire_timeout_secs: { type: "integer", minimum: 0 },
          idle_timeout_secs: { type: "integer", minimum: 0 },
          max_lifetime_secs: { type: "integer", minimum: 0 },
        },
      },
    },
  };
}

async function main(): Promise<void> {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      out: { type: "string", default: "src/schema/terse.schema.json" },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Generate Hyperterse JSON Schema

Usage:
  bun run scripts/generate-schema.ts [--out src/schema/terse.schema.json]
`);
    return;
  }

  const connectorSrc = await readFile(CONNECTOR_RS, "utf8");
  const primitiveSrc = await readFile(PRIMITIVE_RS, "utf8");

  const connectors = extractEnumVariants(connectorSrc, "Connector").map(toLowercaseSerdeVariant);
  const primitives = extractEnumVariants(primitiveSrc, "Primitive").map(toLowercaseSerdeVariant);

  if (connectors.length === 0) {
    throw new Error(`Failed to extract Connector variants from ${CONNECTOR_RS}`);
  }
  if (primitives.length === 0) {
    throw new Error(`Failed to extract Primitive variants from ${PRIMITIVE_RS}`);
  }

  const schema = jsonSchema();
  // Fill enums
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (schema as any).$defs.adapter.properties.connector.enum = connectors;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (schema as any).$defs.adapterInline.properties.connector.enum = connectors;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (schema as any).$defs.input.properties.type.enum = primitives;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (schema as any).$defs.inputInline.properties.type.enum = primitives;

  const outPath = resolve(values.out);
  await mkdir(dirname(outPath), { recursive: true });
  await writeFile(outPath, JSON.stringify(schema, null, 2) + "\n", "utf8");
  console.log(`✓ Wrote ${outPath}`);
}

main().catch((err) => {
  console.error("❌ Schema generation failed:", err);
  process.exit(1);
});

