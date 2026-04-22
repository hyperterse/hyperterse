<!-- Start of - Dont change this block -->
<div align="center">
  <picture>
    <img alt="Hyperterse — The agentic server framework." src="docs/assets/og.png" />
  </picture>
</div>
<br />
<div align="center">
  <h1>Hyperterse</h1>
</div>
<p align="center">
  <strong>The agentic server framework.</strong><br />
  <a href="https://hyperterse.com">Website</a>
  •
  <a href="https://docs.hyperterse.com">Documentation</a>
  •
  <a href="#quick-start">Quick Start</a>
  •
  <a href="#tool-examples">Examples</a>
  •
  <a href="https://github.com/hyperterse/hyperterse">GitHub</a>
</p>

---

<!-- End of - Dont change this block -->

**Hyperterse** is an **agentic server framework**: one build ships **agents** (A2A), **tools** (MCP), **prompts**, **resources**, database adapters, auth, caching, and observability from a single process. You declare surfaces in config; the compiler validates and bundles them. Clients use [MCP](https://modelcontextprotocol.io/) Streamable HTTP at `/mcp` for tools, prompts, and resources; [A2A](https://github.com/a2aproject/a2a-go)-style agent routes live at `/agent/{name}` when you define agents.

## Where to start

- **Agents** — Declarative agents and A2A: [Agents overview](https://docs.hyperterse.com/agents/overview), [Agents quickstart](https://docs.hyperterse.com/agents/quickstart).
- **Tools** — Callable MCP tools (DB or scripts): [Tools](https://docs.hyperterse.com/concepts/tools), [Scripts](https://docs.hyperterse.com/concepts/scripts), [Adapters](https://docs.hyperterse.com/concepts/adapters).
- **Resources** — Static context for clients: [Resources](https://docs.hyperterse.com/concepts/resources).
- **Prompts** — Reusable prompt templates: [Prompts](https://docs.hyperterse.com/concepts/prompts).

The [Quickstart](https://docs.hyperterse.com/quickstart) walks through install, scaffold, and run, then optional MCP tool checks.

## What Hyperterse is for

- Running **agents** alongside **MCP tools**, **prompts**, and **resources** in one deployable service
- Exposing database queries and custom logic as MCP tools with declarative config
- Production **Streamable HTTP** for MCP and **A2A** routes for agents
- TypeScript handlers and transforms where config alone is not enough

## Core capabilities

- **Agents**: declarative configs, tool-access policies, multi-provider models, per-agent A2A HTTP.
- **Filesystem discovery**: one MCP tool per tool definition; prompts and resources follow the same discover-and-compile model (see [Project structure](https://docs.hyperterse.com/concepts/project-structure)).
- **Execution models**: DB-backed tools (`use` + `statement`) or script-backed tools (`handler`).
- **Database adapters**: PostgreSQL, MySQL, SQLite, MongoDB, Redis.
- **Per-tool auth**: built-in `allow_all` and `api_key`, plus custom plugins.
- **In-memory caching**: global defaults + per-tool overrides.
- **Observability**: OpenTelemetry tracing/metrics + structured logging.

## Quick Start

### Install

```bash
curl -fsSL https://hyperterse.com/install | bash
```

### Initialize

```bash
hyperterse init
```

Generated starter structure:

```text
.
├── .hyperterse
├── .agents/
│   └── skills/
│       ├── hyperterse-docs/
│       │   └── SKILL.md
│       └── hyperterse-agents/
│           └── SKILL.md
└── app/
    └── tools/
        └── hello-world/
            ├── config.terse
            └── handler.ts
```

### Start

```bash
hyperterse start
```

With live reload:

```bash
hyperterse start --watch
```

### Verify health

```bash
curl http://localhost:8080/heartbeat
```

Expected response:

```json
{ "success": true }
```

### Optional: list MCP tools

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }' | jq
```

By design, Hyperterse exposes two transport-layer tools:

- `search` - discover project tools by natural language
- `execute` - execute a project tool by name

### Optional: discover project tools (search)

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "search",
      "arguments": {
        "query": "hello world greeting"
      }
    },
    "id": 2
  }' | jq
```

Search hits include `name`, `description`, `relevance_score`, and `inputs`.

### Execute a project tool

```bash
curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "execute",
      "arguments": {
        "tool": "hello-world",
        "inputs": { "name": "Hyperterse" }
      }
    },
    "id": 3
  }' | jq
```

### Validate and build

```bash
hyperterse validate
hyperterse build -o dist
hyperterse serve dist/
```

## Configuration model

### Project layout

```text
my-project/
├── .hyperterse
├── app/
│   ├── adapters/
│   │   └── primary-db.terse
│   └── tools/
│       ├── get-user/
│       │   ├── config.terse
│       │   ├── input.ts
│       │   └── output.ts
│       └── get-weather/
│           ├── config.terse
│           └── handler.ts
└── package.json
```

### Root config (`.hyperterse`)

```yaml
name: my-service
server:
  port: 8080
  log_level: 3
tools:
  search:
    limit: 10
  cache:
    enabled: true
    ttl: 60
```

## Tool Examples

### DB-backed tool

```yaml
description: "Get user by ID"
use: primary-db
statement: |
  SELECT id, name, email
  FROM users
  WHERE id = {{ inputs.user_id }}
inputs:
  user_id:
    type: int
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

### Script-backed tool

```yaml
description: "Get weather by city"
handler: "./handler.ts"
inputs:
  city:
    type: string
auth:
  plugin: allow_all
```

Supported input types:

- `string`
- `int`
- `float`
- `boolean`
- `datetime`

Each tool must define exactly one execution mode:

- `use` (adapter-backed), or
- `handler` (script-backed)

## Runtime model

**MCP** — Streamable HTTP at `/mcp` (JSON-RPC 2.0): tools, prompts, resources, completion, subscriptions. See [MCP transport](https://docs.hyperterse.com/runtime/mcp-transport).

**A2A** — JSON-RPC per agent at `/agent/{agentName}` (agent card, messaging, tasks, streaming). See [A2A transport](https://docs.hyperterse.com/runtime/a2a-transport) and [Agents](https://docs.hyperterse.com/agents/overview).

Execution pipeline:

1. Tool resolution
2. Authentication
3. Input transform (optional)
4. Execution (DB or handler)
5. Output transform (optional)
6. Response serialization

## MCP spec compliance

Hyperterse implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) specification **[2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25)**.

Compliance status by component:

| Spec component                                                                                                                | Status |
| ----------------------------------------------------------------------------------------------------------------------------- | :----: |
| [Base protocol](https://modelcontextprotocol.io/specification/2025-11-25/basic) (JSON-RPC 2.0)                                |   ✅   |
| [Lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle) (initialize/initialized)                |   ✅   |
| [Tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) (list, call, listChanged)                      |   ✅   |
| [Resources](https://modelcontextprotocol.io/specification/2025-11-25/server/resources) (list, read, subscribe, updated)       |   ✅   |
| [Prompts](https://modelcontextprotocol.io/specification/2025-11-25/server/prompts) (list, get, listChanged)                   |   ✅   |
| [Completion](https://modelcontextprotocol.io/specification/2025-11-25/server/utilities/completion) (ref/prompt, ref/resource) |   ✅   |
| [Pagination](https://modelcontextprotocol.io/specification/2025-11-25/server/utilities/pagination) (cursor/nextCursor)        |   ⚠️   |
| Tool result content types (image, audio, resource_link)                                                                       |   ⚠️   |

Text content for tool results is supported; image, audio, and resource links are optional. Pagination applies when tools, prompts, or resources exceed typical small-to-medium counts.

## Security notes

- Use `{{ env.VAR_NAME }}` for secrets and connection strings.
- `{{ inputs.field }}` statement substitution is textual; enforce strict input schemas and safe query patterns.
- Configure tool-level auth explicitly for production use.

## Documentation map

- [Documentation index (`llms.txt`)](https://docs.hyperterse.com/llms.txt)
- [Introduction](https://docs.hyperterse.com/introduction)
- [Quickstart](https://docs.hyperterse.com/quickstart)
- [Project structure](https://docs.hyperterse.com/concepts/project-structure)
- **Agents** — [Overview](https://docs.hyperterse.com/agents/overview), [Agents quickstart](https://docs.hyperterse.com/agents/quickstart), [Tool access](https://docs.hyperterse.com/agents/tool-access), [Runtime API](https://docs.hyperterse.com/agents/runtime-api), [Model providers](https://docs.hyperterse.com/agents/model-providers)
- **Tools** — [Tools](https://docs.hyperterse.com/concepts/tools), [Scripts](https://docs.hyperterse.com/concepts/scripts), [Adapters](https://docs.hyperterse.com/concepts/adapters)
- **Resources** — [Resources](https://docs.hyperterse.com/concepts/resources)
- **Prompts** — [Prompts](https://docs.hyperterse.com/concepts/prompts)
- [Authentication](https://docs.hyperterse.com/concepts/authentication)
- [MCP transport](https://docs.hyperterse.com/runtime/mcp-transport)
- [A2A transport](https://docs.hyperterse.com/runtime/a2a-transport)
- [Execution pipeline](https://docs.hyperterse.com/runtime/execution-pipeline)
- [CLI reference](https://docs.hyperterse.com/reference/cli)
- [Agent config](https://docs.hyperterse.com/reference/agent-config), [Prompt config](https://docs.hyperterse.com/reference/prompt-config), [Resource config](https://docs.hyperterse.com/reference/resource-config)
- [Configuration schemas](https://docs.hyperterse.com/reference/configuration-schemas)

## Contributing

1. Fork the repo.
2. Create a feature branch.
3. Add or update tests.
4. Run validation locally.
5. Open a PR.

See `CONTRIBUTING.md` and `CODE_OF_CONDUCT.md`.

---

<p align="center">
  Agentic server framework—agents, tools, prompts, resources, one engine.<br />
  <a href="https://hyperterse.com">Website</a>
  •
  <a href="https://github.com/hyperterse/hyperterse">GitHub</a>
  •
  <a href="https://docs.hyperterse.com">Documentation</a>
</p>
