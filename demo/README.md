# Hyperterse v2 Demo

This demo follows the v2 framework layout:

```text
demo/
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

## What this demonstrates

- Adapter discovery from `app/adapters/*.terse`
- MCP route discovery from `app/routes/*/config.terse`
- Route TS bundling
- Vendor bundling via npm dependencies (`dayjs`, `uuid`)
- Input transform + output transform flow (`get-user`)
- Custom handler flow (`get-weather`)

## Run

From repository root:

```bash
hyperterse start -f demo/.hyperterse
```

Or in dev mode:

```bash
hyperterse start --watch -f demo/.hyperterse
```
