<div align="center">

# Hyperterse

**A declarative interface between your data and modern software.**

Turn database queries into stable APIs and AI-ready tools—without exposing SQL, writing boilerplate, or coupling your application to your database.

[Website](https://hyperterse.com) • [Features](#features) • [Quick Start](#quick-start) • [Documentation](#documentation) • [Examples](#examples) • [Contributing](#contributing)

</div>

---

## What is Hyperterse?

Hyperterse is a high-performance runtime server that transforms database queries into REST endpoints and MCP (Model Context Protocol) tools.

You describe your queries once, in a simple configuration file. Hyperterse does the rest:

* Generates individual, typed endpoints
* Validates inputs automatically
* Produces OpenAPI documentation
* Exposes queries safely to AI systems

No ORMs. No boilerplate. No exposed SQL.

---

## Designed for Modern Systems

### AI & LLM Applications

Hyperterse is built with AI in mind.

* **AI agents and assistants** - Safely query databases through MCP without exposing raw SQL.
* **LLM tool calling** - Let models discover and invoke database operations autonomously.
* **Retrieval-augmented generation (RAG)** - Use structured database queries as reliable context.
* **Conversational interfaces** - Power chatbots that access live business data.
* **AI-driven analytics** - Enable models to generate insights through validated queries.
* **Multi-agent systems** - Share consistent database access across agents.
* **Natural language to SQL pipelines** - Bridge human input and databases using tool calls.
* **AI dashboards** - Query and visualize data dynamically.

### Traditional Use Cases

* **Database-backed APIs** without boilerplate
* **Lightweight microservices** without ORM overhead
* **Rapid prototyping** with configuration-first workflows

---

## Features

| | |
|--- | --- |
| **Declarative Data Interfaces** | Define the shape and intent of data access once, and let Hyperterse handle execution, validation, and exposure. |
| **Agent-Ready by Design** | Connect your data to AI agents through discoverable, callable tools—without exposing SQL, schemas, or credentials. |
| **Zero-Boilerplate APIs** | Turn queries into production-ready APIs with typed inputs, predictable outputs, and built-in documentation. |
| **Single Source of Truth** | Generate REST endpoints, OpenAPI specs, LLM-readable docs, and MCP tools from one configuration file. |
| **Security as a Baseline** | Keep raw SQL, connection strings, and internal errors fully contained within the runtime. |
| **Database Independence** | Work across PostgreSQL, MySQL, and Redis using a consistent, unified interface. |
| **Fast Iteration** | Update queries and schemas with immediate feedback during development. |
| **Portable Deployment** | Ship a self-contained runtime that moves cleanly from local development to production. |
| **Built to Scale** | Support everything from prototypes to multi-agent systems without changing your architecture. |
<!--

|                                   |                                         |
| --------------------------------- | --------------------------------------- |
| **Automatic endpoint generation** | Each query becomes `POST /query/{name}` |
| **OpenAPI 3.0 support**           | Swagger UI available at `/docs`         |
| **MCP protocol support**          | JSON-RPC 2.0 tools for AI assistants    |
| **Multiple databases**            | PostgreSQL, MySQL, Redis                |
| **Input validation**              | Strong typing and validation by default |
| **LLM-friendly docs**             | Auto-generated at `/llms.txt`           |
| **Secure by design**              | No exposed SQL or credentials           |
| **Hot reload**                    | Instant feedback during development     |
| **Portable exports**              | Ship self-contained deployments         | -->

---

## Quick Start

### Install

Install Hyperterse with a single command:

```bash
curl -fsSL https://hyperterse.com/install | bash
```

**Supported platforms**

* Linux (amd64, arm64, arm)
* macOS (Intel, Apple Silicon)
* Windows (amd64, arm64)

---

### Your First Query

Create a configuration file:

```yaml
adapters:
  my_database:
    connector: postgres
    connection_string: "postgresql://user:password@localhost:5432/mydb"

queries:
  get-user:
    use: my_database
    description: "Retrieve a user by email address"
    statement: |
      SELECT id, name, email, created_at
      FROM users
      WHERE email = {{ inputs.email }}
    inputs:
      email:
        type: string
        description: "User email address"
```

Start the server:

```bash
hyperterse run -f config.terse
```

Call the endpoint:

```bash
curl -X POST http://localhost:8080/query/get-user \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

Response:

```json
{
  "success": true,
  "error": "",
  "results": [
    {
      "id": 123,
      "name": "John Doe",
      "email": "user@example.com",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

---

## Documentation

* **Full documentation**
  [https://docs.hyperterse.com](https://docs.hyperterse.com)

* **CLI reference**
  [https://docs.hyperterse.com/cli](https://docs.hyperterse.com/cli)

* **Configuration guide**
  [https://docs.hyperterse.com/configuration](https://docs.hyperterse.com/configuration)

* **Practical guides**
  [https://docs.hyperterse.com/guides](https://docs.hyperterse.com/guides)

### Runtime Endpoints

* OpenAPI: `GET /docs`
* LLM docs: `GET /llms.txt`
* MCP: `POST /mcp`

---

## Examples

### MCP Protocol

List available tools:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }'
```

Invoke a tool:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "get-user-by-id",
      "arguments": {
        "userId": "123"
      }
    },
    "id": 1
  }'
```

---

### User Management

```yaml
queries:
  get-user-by-id:
    use: user_db
    description: "Retrieve user information by ID"
    statement: |
      SELECT id, name, email, created_at
      FROM users
      WHERE id = {{ inputs.userId }}
```

---

### Analytics

```yaml
queries:
  daily-stats:
    description: "Daily statistics over a date range"
    statement: |
      SELECT
        DATE(created_at) AS date,
        COUNT(*) AS total_events,
        COUNT(DISTINCT user_id) AS unique_users
      FROM events
      WHERE created_at BETWEEN {{ inputs.startDate }} AND {{ inputs.endDate }}
      GROUP BY DATE(created_at)
      ORDER BY date DESC
```

---

## CLI

Run:

```bash
hyperterse run -f config.terse
```

Development mode (hot reload):

```bash
hyperterse dev -f config.terse
```

Generate artifacts:

```bash
hyperterse generate llms -f config.terse
hyperterse generate skills -f config.terse
```

Initialize:

```bash
hyperterse init
```

Upgrade:

```bash
hyperterse upgrade
```

Export:

```bash
hyperterse export -f config.terse -o dist
```

---

## Configuration

### Supported Types

`string`, `int`, `float`, `boolean`, `uuid`, `datetime`

### Templates

```sql
WHERE price <= {{ inputs.maxPrice }}
```

### Optional Inputs

```yaml
optional: true
default: "20"
```

### Multiple Databases

Hyperterse supports multiple adapters in a single configuration.

---

## Security

* Credentials are never exposed
* SQL is never returned to clients
* Inputs are validated and escaped
* Errors are sanitized by default

For production deployments, place Hyperterse behind a reverse proxy for authentication, rate limiting, and TLS.

---

## Development

Requirements:

* Go 1.25.1+
* `protoc`
* `protoc-gen-go`

Setup:

```bash
make setup
make build
go test ./...
```

---

## Contributing

Contributions are welcome.

1. Fork the repository
2. Create a feature branch
3. Add tests
4. Open a pull request

Follow standard Go formatting and keep changes focused.

---

## Support

* Website: [https://hyperterse.com](https://hyperterse.com)
* Docs: [https://docs.hyperterse.com](https://docs.hyperterse.com)
* Issues: [https://github.com/hyperterse/hyperterse/issues](https://github.com/hyperterse/hyperterse/issues)
* Discussions: [https://github.com/hyperterse/hyperterse/discussions](https://github.com/hyperterse/hyperterse/discussions)

---

<div align="center">

Made with care by the Hyperterse team.

[Website](https://hyperterse.com) • [GitHub](https://github.com/hyperterse/hyperterse) • [Docs](https://docs.hyperterse.com)

</div>
