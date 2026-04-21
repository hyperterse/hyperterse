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
      sqlite-db.terse
    tools/
      get-user/
        config.terse
        user-input-validator.ts
        user-data-mapper.ts
      get-weather/
        config.terse
        weather-handler.ts
      list-sqlite-notes/
        config.terse
    prompts/
      incident-update.terse
      summarize-release.terse
    agents/
      demo-concierge/
        config.terse
      weather-guide/
        config.terse
      notes-analyst/
        config.terse
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

- Adapter discovery from `app/adapters/*.terse` (including SQLite connector)
- Tool discovery from `app/tools/*/config.terse`
- Prompt discovery from `app/prompts/**/*.terse`
- Agent discovery from `app/agents/*/config.terse`
- Resource + resource-template discovery from `app/resources/**/config.terse`
- Concrete resources from both inline `text` and file-backed `file`
- Resource templates from both `text_template` and `file_template`
- Tool TS bundling
- Vendor bundling via npm dependencies (`dayjs`, `uuid`)
- Input transform + output transform flow (`get-user`)
- Custom handler flow (`get-weather`)
- Prompt argument completion (`completion/complete` with `ref/prompt`)
- Resource template argument completion (`completion/complete` with `ref/resource`)

## SQLite connector

The demo includes a SQLite adapter and tool that work without external services. To use them:

1. Seed the demo database (run from repo root):

```bash
sqlite3 demo.db < demo/seed-sqlite.sql
```

2. Start the server and invoke the tool:

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "method":"tools/call",
    "params":{"name":"execute","arguments":{"tool":"list-sqlite-notes","inputs":{}}},
    "id":16
  }' | jq
```

The SQLite adapter uses `file:./demo.db` by default; the database file is created in the current working directory when you run the server.

## Run

From repository root:

```bash
export OPENROUTER_API_KEY=your_openrouter_key
hyperterse start -f demo/.hyperterse
```

Or in dev mode:

```bash
export OPENROUTER_API_KEY=your_openrouter_key
hyperterse start --watch -f demo/.hyperterse
```

## Try demo agents

The demo now includes three A2A agents backed by an OpenRouter free model:

- `demo-concierge`
- `weather-guide`
- `notes-analyst`

Examples:

```bash
curl -s http://localhost:8080/agent/demo-concierge/.well-known/agent-card.json | jq

curl -s -X POST http://localhost:8080/agent/weather-guide \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc":"2.0",
    "id":1,
    "method":"SendMessage",
    "params":{"message":{"role":"user","parts":[{"text":"What is the weather in Austin in imperial units?"}]}}
  }' | jq
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
