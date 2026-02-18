# Hyperterse v2 Migration Guide

This release is a breaking framework shift from query-only configs to route-driven MCP tooling.

## What changed

- Legacy DSL format is no longer supported; `.hyperterse` is now YAML root config.
- Adapter configs now live under `app/adapters/*.terse`.
- Route tools now come from `app/routes/*/config.terse`.
- Root `.hyperterse` is now for shared config (server/framework options).
- TypeScript route scripts are bundled automatically.
- MCP route behavior is implicit from filesystem structure.

## Migration steps

1. Keep your current root `.hyperterse` and remove inline `queries` over time.
2. Create `app/adapters` and `app/routes` folders.
3. Move adapter definitions from root config into `app/adapters/*.terse`.
4. For each existing query, create one route `config.terse`:
   - copy `description`
   - copy `use`
   - copy `statement`
   - copy `inputs` and `data`
5. Add route scripts only where customization is needed:
   - `scripts.handler` for custom tool execution
   - `scripts.input_transform` / `scripts.output_transform` for data shaping

## Before (v1)

```yaml
name: my-api
adapters:
  main_db:
    connector: postgres
    connection_string: "postgresql://..."
queries:
  get-user:
    use: main_db
    description: "Get user by id"
    statement: "SELECT * FROM users WHERE id = {{ inputs.user_id }}"
    inputs:
      user_id:
        type: int
```

## After (v2)

Root `.hyperterse`:

```yaml
name: my-api
framework:
  app_dir: "app"
```

Adapter `app/adapters/main-db.terse`:

```yaml
connector: postgres
connection_string: "postgresql://..."
```

Route `app/routes/get-user/config.terse`:

```yaml
description: "Get user by id"
use: main-db
statement: "SELECT * FROM users WHERE id = {{ inputs.user_id }}"
inputs:
  user_id:
    type: int
```

## Validation and runtime commands

- Validate: `hyperterse validate -f .hyperterse`
- Run: `hyperterse start -f .hyperterse`
- Dev/hot reload: `hyperterse start --watch -f .hyperterse`

The dev watcher now tracks `.terse` and `.ts` changes under `app/`.
