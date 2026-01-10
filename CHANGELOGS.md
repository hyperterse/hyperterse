# v1.0.0-alpha.1

Timestamp: 2026-01-10 11:20:55 UTC

🔄 **Developer Experience Improvements**

This release focuses on improving the development workflow with graceful server reloads and better signal handling.

## ✨ Features & Enhancements

### Dev Server

- **Graceful Reloads** — Dev server now supports graceful reloads on config changes, allowing the old server to continue running until the new one is ready
- **Graceful Shutdown** — Dev command now handles exit signals for graceful shutdown

### Documentation

- **LLM Types** — Updated `generate llms` command handler to generate proper types in docs

## 🐛 Bug Fixes

- **Enum Validation** — Fixed runtime input validator parsing of enums

## 🔧 Improvements

- **Build Consistency** — Refactored build command in `.air.toml` to use Makefile for improved consistency and clarity
- **Air Integration** — Updated run command when using air

## 📝 Maintenance

- Ignore system files
- Updated project documentation

---

_This is an alpha release of Hyperterse. We welcome feedback and contributions!_

# v1.0.0-alpha.0

Timestamp: 2026-01-10 11:20:55 UTC

🎉 **First Alpha Release**

Hyperterse is a high-performance runtime server that transforms database queries into RESTful API endpoints and MCP (Model Context Protocol) tools. Define queries in YAML, and Hyperterse automatically generates endpoints with full OpenAPI documentation.

## ✨ Features

### Core Runtime

- **Automatic Endpoint Generation** — Each query becomes its own REST endpoint at `POST /query/{query-name}`
- **Multi-Database Support** — PostgreSQL, MySQL, and Redis connectors out of the box
- **Input Validation** — Automatic type checking and validation for all query inputs
- **Template Variables** — Use `{{ inputs.parameterName }}` syntax in SQL statements

### Supported Data Types

- `string`, `int`, `float`, `boolean`, `uuid`, `datetime`
- Optional inputs with default values
- Required field enforcement

### AI & LLM Integration

- **MCP Protocol Support** — Expose queries as MCP tools via JSON-RPC 2.0 endpoint (`POST /mcp`)
- **LLM Documentation** — Auto-generated markdown documentation at `GET /llms.txt`
- **Agent Skills Export** — Generate Agent Skills compatible archives with `hyperterse generate skills`

### Documentation

- **OpenAPI 3.0 Specification** — Complete API documentation at `GET /docs`
- **Request/Response Schemas** — Auto-generated from query definitions
- **Example Values** — Included in OpenAPI spec

### Configuration

- **YAML Format** — Simple, readable configuration files (`.yaml`, `.yml`)
- **Comprehensive Validation** — Catches configuration errors before startup

### CLI Commands

- `hyperterse -f config.yaml` — Start the runtime server
- `hyperterse run -f config.yaml` — Start with explicit run command
- `hyperterse dev -f config.yaml` — Development mode with hot reload
- `hyperterse generate llms -f config.yaml` — Generate llms.txt documentation
- `hyperterse generate skills -f config.yaml` — Generate Agent Skills archive

### Security

- **Connection String Protection** — Never exposed in API responses or documentation
- **SQL Injection Prevention** — Proper escaping and type validation
- **Error Message Sanitization** — No sensitive database information leaked

## 📦 Installation

### Quick install

```bash
curl -fsSL https://github.com/hyperterse/hyperterse/releases/latest/download/install.sh | bash
```

Or download binary directly for your platform below.

### Supported platforms

| Platform | Architecture                         |
| -------- | ------------------------------------ |
| Linux    | amd64, arm64, arm                    |
| macOS    | amd64 (Intel), arm64 (Apple Silicon) |
| Windows  | amd64, arm64                         |

### ⚠️ Known Limitations

- This is an alpha release — APIs may change in future versions
- No built-in authentication/authorization (use a reverse proxy)
- No connection pooling configuration exposed yet

## 📚 Documentation

- [README](https://github.com/hyperterse/hyperterse#readme)
- [Full Documentation](https://github.com/hyperterse/hyperterse/blob/main/HYPERTERSE.md)

---

_This is the first public alpha release of Hyperterse. We welcome feedback and contributions!_
