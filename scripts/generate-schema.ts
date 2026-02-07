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
        description: "Database adapters",
        type: "object",
        default: {},
        propertyNames: { type: "string", pattern: NAME_PATTERN },
        additionalProperties: { $ref: "#/$defs/adapterInline" },
      },
      queries: {
        description: "Query definitions",
        type: "object",
        default: {},
        propertyNames: { type: "string", pattern: NAME_PATTERN },
        additionalProperties: { $ref: "#/$defs/queryInline" },
      },
      server: {
        type: "object",
        additionalProperties: false,
        description: "Optional server configuration.",
        properties: {
          port: {
            description: "Port to listen on.",
            anyOf: [{ type: "integer", minimum: 1, maximum: 65535 }, { type: "string" }],
          },
          log_level: {
            type: "integer",
            minimum: 0,
            maximum: 4,
            description: "Log level.",
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
          out: {
            type: "string",
            description: "Output directory for export.",
          },
        },
      },
    },
    $defs: {
      adapterInline: {
        type: "object",
        additionalProperties: false,
        required: ["connector", "connection_string"],
        properties: {
          connector: {
            type: "string",
            enum: [], // filled in at runtime
            description: "Connector type.",
          },
          connection_string: {
            type: "string",
            minLength: 1,
            description: "Connection string.",
          },
          options: {
            type: "object",
            additionalProperties: { type: "string" },
            description: "Key-value pairs appended as query parameters to the connection string.",
          },
        },
      },
      queryInline: {
        type: "object",
        additionalProperties: false,
        required: ["use", "statement"],
        properties: {
          use: {
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
            description: "Input parameters.",
            type: "object",
            default: {},
            propertyNames: { type: "string", pattern: IDENT_PATTERN },
            additionalProperties: { $ref: "#/$defs/inputInline" },
          },
        },
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
            description: "Whether the input is optional.",
          },
          default: {
            description: "Default value for optional inputs.",
          },
        },
        allOf: [
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
  (schema as any).$defs.adapterInline.properties.connector.enum = connectors;
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
