<div align="center">

[![Hyperterse](docs/public/og.png)](https://docs.hyperterse.com)

# Hyperterse

**Connect your data to your agents.**

A declarative interface between your data and modern software. Turn database queries into stable APIs and AI-ready tools—without exposing SQL, writing boilerplate, or coupling your application to your database.

[Website](https://hyperterse.com) • [Documentation](https://docs.hyperterse.com) • [Quick Start](#quick-start) • [Features](#features) • [Examples](#examples)

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

## Features

Hyperterse comes several features that help you leverage accessing your data in a secure manner for your AI agents and LLMs.

### Declarative data interfaces

Define the shape and intent of data access once, and let Hyperterse handle execution, validation, and exposure.

### Agent-ready by design

Connect your data to AI agents through discoverable, callable tools—without exposing SQL, schemas, or credentials.

### Zero-boilerplate API

Turn queries into production-ready APIs with typed inputs, predictable outputs, and built-in documentation.

### Single source of truth

Generate REST endpoints, OpenAPI specs, LLM-readable docs, and MCP tools from one configuration file.

### Database independence

Work across PostgreSQL, MySQL, and Redis using a consistent, unified interface.

### Fast and iterative development

Update queries and schemas with immediate feedback during development.

### Portable deployment

Ship a self-contained runtime that moves cleanly from local development to production.

### Built to scale

Support everything from prototypes to multi-agent systems without changing your architecture.

## Quick Start

### Installation

Install Hyperterse with a single command:

```bash
curl -fsSL https://hyperterse.com/install | bash
```

**Supported platforms:**

* Linux (amd64, arm64, arm)
* macOS (Intel, Apple Silicon)
* Windows (amd64, arm64)

For more installation options, see the [installation guide](https://docs.hyperterse.com/getting-started/installation).

### Your First Query

Create a configuration file:

```yaml
name: my-api

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

## Documentation

📚 **[Read the docs →](https://docs.hyperterse.com)**

* **[Getting Started](https://docs.hyperterse.com/getting-started/quick-start)** - Quick start guide and installation
* **[CLI Reference](https://docs.hyperterse.com/reference/cli)** - Complete command-line interface reference
* **[Configuration Guide](https://docs.hyperterse.com/reference/configuration)** - Configuration file reference
* **[Guides](https://docs.hyperterse.com/guides)** - Practical guides for AI integration, OpenAPI, and MCP
* **[Concepts](https://docs.hyperterse.com/concepts)** - Core concepts: adapters, queries, and inputs
* **[Databases](https://docs.hyperterse.com/databases)** - Database-specific documentation
* **[Deployment](https://docs.hyperterse.com/deployment)** - Deployment guides for various platforms

### Runtime Endpoints

When running, Hyperterse exposes several endpoints:

* **OpenAPI Documentation**: `GET /docs` - Interactive API documentation
* **LLM Documentation**: `GET /llms.txt` - AI-friendly documentation format
* **MCP Protocol**: `POST /mcp` - Model Context Protocol JSON-RPC endpoint

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

## Configuration

Hyperterse uses a simple YAML-like configuration format (`.terse` files) to define your database adapters and queries.

**Supported input types:** `string`, `int`, `float`, `boolean`, `uuid`, `datetime`

**Template syntax:**

```sql
WHERE price <= {{ inputs.maxPrice }}
```

**Optional inputs:**

```yaml
optional: true
default: "20"
```

**Multiple databases:** Hyperterse supports multiple adapters in a single configuration file.

For complete configuration reference, see the [configuration guide](https://docs.hyperterse.com/reference/configuration).

## Security

Hyperterse is designed with security as a baseline:

* 🔒 **Credentials are never exposed** - Connection strings stay server-side
* 🛡️ **SQL is never returned to clients** - Raw queries remain hidden
* ✅ **Inputs are validated and escaped** - Strong typing prevents injection
* 🔇 **Errors are sanitized by default** - Internal details stay internal

For production deployments, place Hyperterse behind a reverse proxy for authentication, rate limiting, and TLS. See the [production security guide](https://docs.hyperterse.com/security/production) for best practices.

## Contributing

Contributions are welcome! We appreciate your help in making Hyperterse better.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for your changes
4. Ensure code follows Rust formatting (`cargo fmt`)
5. Run linting (`cargo clippy`)
6. Open a pull request

Please keep changes focused and well-tested. For major changes, open an issue first to discuss your proposal. See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development guidelines.

**Please note**: By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). We are committed to providing a welcoming and inclusive environment for all contributors.

## Support

* 🌐 **Website**: [hyperterse.com](https://hyperterse.com)
* 📖 **Documentation**: [docs.hyperterse.com](https://docs.hyperterse.com)
* 🐛 **Issues**: [GitHub Issues](https://github.com/hyperterse/hyperterse/issues)
* 💬 **Discussions**: [GitHub Discussions](https://github.com/hyperterse/hyperterse/discussions)


---

<div align="center">


Made with care by the Hyperterse team.

[Website](https://hyperterse.com) • [GitHub](https://github.com/hyperterse/hyperterse) • [Documentation](https://docs.hyperterse.com)

</div>
