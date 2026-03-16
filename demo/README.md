# Hyperterse Demo

This demo shows the full Hyperterse project layout with adapters, tools, prompts,
and resources.

```text
demo/
  .hyperterse
  app/
    adapters/
      my-adapter.terse
      my-other-adapter.terse
    tools/
      get-user/
        config.terse
        user-input-validator.ts
        user-data-mapper.ts
      get-weather/
        config.terse
        weather-handler.ts
    prompts/
      incident-update.terse
      summarize-release.terse
    resources/
      service-info/
        config.terse
      order-by-id/
        config.terse
      order-file-by-id/
        config.terse
        1001.json
        1002.json
      release-notes/
        config.terse
        release-notes.md
```

## What this demonstrates

- Adapter discovery from `app/adapters/*.terse`
- Tool discovery from `app/tools/*/config.terse`
- Prompt discovery from `app/prompts/**/*.terse`
- Resource + resource-template discovery from `app/resources/**/config.terse`
- Concrete resources from both inline `text` and file-backed `file`
- Resource templates from both `text_template` and `file_template`
- Tool TS bundling
- Vendor bundling via npm dependencies (`dayjs`, `uuid`)
- Input transform + output transform flow (`get-user`)
- Custom handler flow (`get-weather`)
- Prompt argument completion (`completion/complete` with `ref/prompt`)
- Resource template argument completion (`completion/complete` with `ref/resource`)

## Run

From repository root:

```bash
hyperterse start -f demo/.hyperterse
```

Or in dev mode:

```bash
hyperterse start --watch -f demo/.hyperterse
```

## Try MCP APIs

### List transport tools

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' | jq
```

### Discover and execute tools

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"tools/call",
    "params":{"name":"search","arguments":{"query":"weather by city"}},
    "id":2
  }' | jq

curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"tools/call",
    "params":{"name":"execute","arguments":{"tool":"get-weather","inputs":{"city":"Austin","units":"imperial"}}},
    "id":3
  }' | jq
```

### List prompts

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"prompts/list","id":4}' | jq
```

### Get a prompt

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"prompts/get",
    "params":{"name":"summarize-release","arguments":{"audience":"engineering","tone":"concise"}},
    "id":5
  }' | jq
```

### Prompt completion

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"completion/complete",
    "params":{
      "ref":{"type":"ref/prompt","name":"incident-update"},
      "argument":{"name":"severity","value":"sev"}
    },
    "id":6
  }' | jq
```

### List resources and templates

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/list","id":7}' | jq

curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/templates/list","id":8}' | jq
```

### Read concrete + template resources

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"memory://service/info"},"id":9}' | jq

curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"memory://release-notes/latest"},"id":10}' | jq

curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"memory://orders/1001"},"id":11}' | jq

curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/read","params":{"uri":"memory://orders-file/1002"},"id":12}' | jq
```

### Resource-template completion

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"completion/complete",
    "params":{
      "ref":{"type":"ref/resource","uri":"memory://orders-file/{id}"},
      "argument":{"name":"id","value":"10"}
    },
    "id":13
  }' | jq
```

### Subscribe and unsubscribe

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/subscribe","params":{"uri":"memory://release-notes/latest"},"id":14}' | jq

curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"resources/unsubscribe","params":{"uri":"memory://release-notes/latest"},"id":15}' | jq
```
