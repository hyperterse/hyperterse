# Configuration Schemas

This document is the source-of-truth checklist for config schema coverage in the repository.

## Schema Files

- `schema/root.terse.schema.json` (root-level `.hyperterse`)
- `schema/adapter.terse.schema.json` (adapter config files)
- `schema/tool.terse.schema.json` (tool config files)
- `schema/prompt.terse.schema.json` (prompt config files)
- `schema/resource.terse.schema.json` (resource and resource-template config files)

## Role Ownership

- **Root (`.hyperterse`)**
  - Owns service-wide settings and discovery configuration.
  - Fields: `name`, `version`, `root`, `build`, `server`, `tools.directory`, `tools.cache`, `tools.search`, `adapters.directory`, `prompts`, `resources`, `resource_templates`.
- **Adapter (`app/adapters/*.terse`)**
  - Owns connection-level runtime configuration.
  - Fields: `name`, `connector`, `connection_string`, `options`.
- **Route / Tool (`app/tools/**/config.terse`)**
  - Owns tool behavior exposed through MCP tools.
  - Fields: `name`, `description`, `use`, `statement`, `handler`, `mappers`, `auth`, `inputs`.
- **Prompt (`app/prompts/**/*.terse`)**
  - Owns prompt definitions exposed through MCP prompts.
  - Fields: `name`, `title`, `description`, `arguments`, `messages`.
- **Resource (`app/resources/**/config.terse`)**
  - Owns concrete resources and URI-template resources exposed through MCP resources.
  - Fields: `uri` / `uri_template`, metadata (`name`, `title`, `description`, `mime_type`), content (`text`/`file` or `text_template`/`file_template`), `arguments`.

## Required vs Optional and Constraints

- **Root**
  - Required: `name`.
  - Optional: all other root fields.
  - Constraints:
    - `name` matches `^[a-z][a-z0-9_-]*$`.
    - `tools.search.limit` is integer >= 1.
    - `tools.cache.ttl` is integer >= 1 when present.
    - `server.log_level` is integer 1..4.
    - `prompts` accepts either discovery object (`directory`) or inline prompt list.
    - `resources` accepts either discovery object (`directory`) or inline resource list.
    - `resource_templates` accepts inline template list.
- **Adapter**
  - Required: `connector`, `connection_string`.
  - Optional: `name`, `options`.
  - Constraints:
    - `connector` enum is derived from connector protobuf.
    - Current values: `postgres`, `redis`, `mysql`, `mongodb`, `sqlite`.
- **Tool**
  - Required: exactly one of `use` or `handler`.
  - Optional: `name`, `description`, `statement`, `mappers`, `auth`, `inputs`.
  - Constraints:
    - `name` matches `^[a-z][a-z0-9_-]*$` when present.
    - Input names match `^[a-zA-Z][a-zA-Z0-9_-]*$`.
    - Input `type` enum is derived from primitive protobuf.
- **Prompt**
  - Required: `messages`.
  - Optional: `name`, `title`, `description`, `arguments`.
  - Constraints:
    - `messages` must be non-empty.
    - Message role enum: `user`, `assistant`, `system`.
    - Argument names match `^[a-zA-Z][a-zA-Z0-9_-]*$`.
- **Resource**
  - Required:
    - one of `uri` or `uri_template`.
    - if `uri`: one of `text` or `file`.
    - if `uri_template`: one of `text_template` or `file_template`.
  - Optional: metadata fields and template `arguments`.
  - Constraints:
    - Template argument names match `^[a-zA-Z][a-zA-Z0-9_-]*$`.

## Defaults

- Root discovery defaults:
  - `root`: `app`
  - `tools.directory`: `tools`
  - `adapters.directory`: `adapters`
  - `prompts.directory`: `prompts`
  - `resources.directory`: `resources`
- MCP search default:
  - `tools.search.limit`: `10` (when unset)

## Regeneration

Regenerate schemas after config-shape changes:

```bash
bun run scripts/generate-schema.ts proto/connectors/connectors.proto proto/primitives/primitives.proto
```

## Editor Mapping (`.vscode/settings.json`)

`yaml.schemas` should include:

- `./schema/root.terse.schema.json` → `**/.hyperterse`
- `./schema/adapter.terse.schema.json` → `**/adapters/*.terse`
- `./schema/tool.terse.schema.json` → `**/tools/**/config.terse`
- `./schema/prompt.terse.schema.json` → `**/prompts/**/*.terse`
- `./schema/resource.terse.schema.json` → `**/resources/**/config.terse`
