# Hyperterse

Framework for building MCP tools from declarative data routes.

Hyperterse turns route configs in `app/routes/*/config.terse` into callable tools, with:

- filesystem-based tool routing,
- pluggable adapters (Postgres, MySQL, MongoDB, Redis),
- optional scripts for transforms and handlers,
- MCP runtime exposure over Streamable HTTP.

## What Hyperterse is

- **Tool-first MCP framework**: each route compiles into a tool.
- **Declarative runtime**: root config + adapter files + route files.
- **Extensible execution pipeline**: auth -> input transform -> execute -> output transform.
- **Embedded scripting**: script hooks run in a sandboxed runtime; bundled at compile time.

## Runtime surface

Current server routes registered by runtime:

- `GET/POST/DELETE /mcp`
- `GET /heartbeat`

## Quick start

### Install

```bash
curl -fsSL https://hyperterse.com/install | bash
```

### Initialize

```bash
hyperterse init
```

This scaffolds:

- `.hyperterse`
- `app/adapters/my-database.terse`
- `app/routes/health/config.terse`
- `app/routes/health/handler.ts`

### Run

```bash
hyperterse start
```

With hot reload:

```bash
hyperterse start --watch
```

### Test

```bash
curl http://localhost:8080/heartbeat
```

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }'
```

## Project structure

```text
my-project/
  .hyperterse
  app/
    adapters/
      main-db.terse
    routes/
      get-user/
        config.terse
        input.ts
        output.ts
      get-weather/
        config.terse
        handler.ts
```

## Route examples

### DB-backed route

```yaml
description: "Get user by id"
use: main-db
statement: |
  SELECT id, name, email
  FROM users
  WHERE id = {{ inputs.user_id }}
inputs:
  user_id:
    type: int
```

### Script-backed route

```yaml
description: "Get weather"
scripts:
  handler: "./weather-handler.ts"
```

## CLI commands

- `start` - run runtime from config
- `serve` - run from built manifest (`model.bin`)
- `build` - package runtime binary + manifest + bundles
- `validate` - validate config and route scripts
- `init` - scaffold starter project
- `upgrade` - upgrade installed binary
- `completion` - shell completion helper

## Build and deploy

```bash
hyperterse validate
hyperterse build -o dist
cd dist
./hyperterse serve
```

Build output includes:

- `model.bin`
- runtime binary
- `build/vendor.js`
- `build/routes/...` bundles

## Configuration highlights

- root config: `.hyperterse`
- adapter files: `app/adapters/*.terse`
- route files: `app/routes/*/config.terse`

Supported primitive types:

- `string`
- `int`
- `float`
- `boolean`
- `datetime`

## Security note

Hyperterse validates typed inputs, but statement placeholder substitution (`{{ inputs.x }}`) is raw string replacement. Use strict route input constraints and safe query patterns for production.

## Documentation

Docs are Mintlify-native in `docs/`:

- `docs/docs.json` (navigation + site config)
- `docs/**/*.mdx` (content pages)

Run docs locally:

```bash
cd docs
bun install
bun run dev
```

## Contributing

1. Fork the repo
2. Create a feature branch
3. Add or update tests
4. Run validation/lint/test locally
5. Open a PR

See `CONTRIBUTING.md` and `CODE_OF_CONDUCT.md`.
