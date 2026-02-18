# Hyperterse v2 Framework

Hyperterse v2 introduces an app-router style framework for MCP tools.

## Project layout

```text
my-project/
  .hyperterse
  app/
    adapters/
      my-adapter.terse
      my-other-adapter.terse
    routes/
      get-user/
        config.terse
        user-input-validator.ts
        user-data-mapper.ts
      get-weather/
        config.terse
        weather-handler.ts
```

- Adapter configs live under `app/adapters/*.terse`.
- Tool routes live under `app/routes/<tool>/config.terse`.
- Route scripts are TypeScript and bundled at runtime startup.
- MCP route behavior is implicit from filesystem structure (no route type configuration).

## Root config (`.hyperterse`)

The root `.hyperterse` file contains shared server/framework settings. Adapter and tool definitions can be omitted because they are discovered from `app/adapters` and `app/routes`.

```yaml
name: my-framework
server:
  port: 8080
  log_level: 3
framework:
  app_dir: "app"
```

## Route config (`app/routes/*/config.terse`)

```yaml
description: "Get user by id"
use: main_db
statement: |
  SELECT id, name, email
  FROM users
  WHERE id = {{ inputs.user_id }}
inputs:
  user_id:
    type: int
    description: "User id"
scripts:
  input_transform: "./input.ts"
  output_transform: "./output.ts"
auth:
  plugin: api_key
  policy:
    value: "dev-key"
```

For custom non-DB MCP behavior, use a route handler:

```yaml
description: "Custom MCP functionality"
scripts:
  handler: "./handler.ts"
```

Script conventions are also supported if `scripts` fields are omitted:

- `*handler.ts` -> handler
- `*input*validator*.ts` -> input transform
- `*data*mapper*.ts` -> output transform

## Script contracts

- `handler.ts` export: `handler(payload)`
- `input_transform` export: `inputTransform(payload)`
- `output_transform` export: `outputTransform(payload)`

Current payload shape:

- handler: `{ inputs, route }`
- input transform: `{ inputs, route }`
- output transform: `{ results, route }`

## Bundling behavior

- Backend: native `esbuild` Go API.
- Shared dependency artifact: `<build.out>/build/vendor.js` (defaults to `dist/build/vendor.js`).
- Route bundles are emitted under `<build.out>/build/routes/<tool>/` (defaults to `dist/build/routes/<tool>/`).
- External package imports are rewritten to direct `vendor.js` registry references in each route script.

## Auth plugins

Built-in plugins:

- `allow_all`
- `api_key` (`X-API-Key` header, compares with route policy `value` or `HYPERTERSE_API_KEY`)

You can register custom auth plugins at runtime through `framework.RegisterAuthPlugin(...)`.
