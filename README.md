<div align="center">

# 🚀 Hyperterse

**Transform database queries into RESTful APIs and AI tools**

[Features](#-features) • [Quick Start](#-quick-start) • [Documentation](#-documentation) • [Examples](#-examples) • [Contributing](#-contributing)

</div>

---

## ✨ What is Hyperterse?

**Hyperterse** is a high-performance runtime server that transforms your database queries into RESTful API endpoints and MCP (Model Context Protocol) tools. Define queries in a simple configuration file, and Hyperterse automatically generates individual endpoints with full OpenAPI documentation, input validation, and AI integration.

### 🎯 Perfect For

- **API Gateway for Databases** - Quickly expose database queries as REST APIs without boilerplate
- **AI/LLM Integration** - Make database queries available to AI assistants via MCP protocol
- **Microservices** - Create lightweight query services without full ORM overhead
- **Rapid Prototyping** - Define queries in configuration files and immediately have working APIs
- **Data Access Layers** - Build secure, documented data APIs with automatic validation

---

## 🌟 Features

| Feature | Description |
|---------|-------------|
| 🚀 **Automatic Endpoint Generation** | Each query becomes its own REST endpoint at `POST /query/{query-name}` |
| 📚 **OpenAPI 3.0 Compliant** | Full OpenAPI specification with Swagger documentation at `GET /docs` |
| 🔧 **MCP Protocol Support** | Expose queries as MCP tools for AI assistants via JSON-RPC 2.0 |
| 🗄️ **Multi-Database Support** | PostgreSQL, MySQL, and Redis connectors out of the box |
| ✅ **Input Validation** | Automatic type checking and validation for all query inputs |
| 📖 **LLM Documentation** | Auto-generated markdown documentation at `GET /llms.txt` |
| 🔒 **Security First** | Connection strings and raw SQL never exposed to clients |
| 🔄 **Hot Reload** | Development mode with automatic reload on configuration changes |
| 🎨 **Multiple Config Formats** | Support for YAML (`.yaml`, `.yml`) and DSL (`.terse`, `.hyperterse`) |
| 📦 **Portable Deployment** | Export self-contained scripts with embedded configuration |

---

## 🚀 Quick Start

### Installation

```bash
curl -fsSL https://hyperterse.com/install | bash
```

**Supported Platforms:**
- Linux (amd64, arm64, arm)
- macOS (amd64 Intel, arm64 Apple Silicon)
- Windows (amd64, arm64)

### Your First Query

1. **Create a configuration file** (`config.terse`):

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

2. **Start the server**:

```bash
hyperterse run -f config.terse
```

3. **Execute your query**:

```bash
curl -X POST http://localhost:8080/query/get-user \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

**Response:**

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

## 📖 Documentation

- **[Full Documentation](HYPERTERSE.md)** - Comprehensive guide covering all features
- **[API Reference](HYPERTERSE.md#api-reference)** - Complete API documentation
- **[Configuration Guide](HYPERTERSE.md#configuration)** - Detailed configuration reference
- **[CLI Reference](HYPERTERSE.md#cli-reference)** - All available commands and flags
- **[Examples](#-examples)** - Real-world usage examples below

### Quick Links

- **OpenAPI Spec**: `GET http://localhost:8080/docs`
- **LLM Documentation**: `GET http://localhost:8080/llms.txt`
- **MCP Endpoint**: `POST http://localhost:8080/mcp`

---

## 💡 Examples

### Example 1: User Management

```yaml
adapters:
  user_db:
    connector: postgres
    connection_string: "postgresql://user:pass@localhost:5432/users"

queries:
  get-user-by-id:
    use: user_db
    description: "Retrieve user information by ID"
    statement: |
      SELECT id, name, email, created_at
      FROM users
      WHERE id = {{ inputs.userId }}
    inputs:
      userId:
        type: int
        description: "Unique user identifier"

  search-users:
    use: user_db
    description: "Search users with pagination"
    statement: |
      SELECT id, name, email
      FROM users
      WHERE name ILIKE {{ inputs.searchTerm }}
      ORDER BY created_at DESC
      LIMIT {{ inputs.limit }}
      OFFSET {{ inputs.offset }}
    inputs:
      searchTerm:
        type: string
        description: "Search term for user names"
      limit:
        type: int
        description: "Maximum number of results"
        optional: true
        default: "20"
      offset:
        type: int
        description: "Number of results to skip"
        optional: true
        default: "0"
```

### Example 2: Analytics Queries

```yaml
adapters:
  analytics:
    connector: mysql
    connection_string: "user:pass@tcp(localhost:3306)/analytics"

queries:
  daily-stats:
    use: analytics
    description: "Get daily statistics for a date range"
    statement: |
      SELECT
        DATE(created_at) as date,
        COUNT(*) as total_events,
        COUNT(DISTINCT user_id) as unique_users
      FROM events
      WHERE created_at BETWEEN {{ inputs.startDate }} AND {{ inputs.endDate }}
      GROUP BY DATE(created_at)
      ORDER BY date DESC
    inputs:
      startDate:
        type: datetime
        description: "Start date (ISO 8601 format)"
      endDate:
        type: datetime
        description: "End date (ISO 8601 format)"
```

### Example 3: Using MCP Protocol

**List available tools:**

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }'
```

**Execute a tool:**

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

## 🛠️ CLI Commands

### Run Server

```bash
# Start server with config file
hyperterse run -f config.terse

# Custom port
hyperterse run -f config.terse -p 3000

# Verbose logging
hyperterse run -f config.terse -v
```

### Development Mode

```bash
# Hot reload on config changes
hyperterse dev -f config.terse
```

### Generate Documentation

```bash
# Generate LLM documentation
hyperterse generate llms -f config.terse -o docs/llms.txt

# Generate Agent Skills archive
hyperterse generate skills -f config.terse -o my-skill.zip
```

### Initialize Project

```bash
# Create a new configuration file
hyperterse init -o config.terse
```

### Upgrade

```bash
# Check for and install updates
hyperterse upgrade

# Upgrade to latest including pre-releases
hyperterse upgrade --prerelease
```

### Export

```bash
# Create portable deployment script
hyperterse export -f config.terse -o dist
```

---

## 🔧 Configuration

### Supported Data Types

- `string` - Text values
- `int` - Integer numbers (64-bit)
- `float` - Floating-point numbers (64-bit)
- `boolean` - True/false values
- `uuid` - UUID strings
- `datetime` - ISO 8601 datetime strings (RFC3339)

### Template Variables

Use `{{ inputs.parameterName }}` in your SQL statements:

```sql
SELECT * FROM products
WHERE category = {{ inputs.category }}
AND price <= {{ inputs.maxPrice }}
ORDER BY created_at DESC
LIMIT {{ inputs.limit }}
```

### Optional Inputs

```yaml
inputs:
  limit:
    type: int
    description: "Maximum number of results"
    optional: true
    default: "20"
```

### Multiple Databases

```yaml
adapters:
  production_db:
    connector: postgres
    connection_string: "postgresql://user:pass@prod-host:5432/prod_db"

  analytics_db:
    connector: mysql
    connection_string: "user:pass@tcp(analytics-host:3306)/analytics"

queries:
  cross-db-query:
    use:
      - production_db
      - analytics_db
    description: "Query spanning multiple databases"
    statement: |
      SELECT * FROM production_db.users
      UNION ALL
      SELECT * FROM analytics_db.users
```

---

## 🔒 Security

- ✅ **Connection strings** are never exposed in API responses or documentation
- ✅ **Raw SQL statements** are not included in MCP tool descriptions
- ✅ **Input validation** prevents SQL injection through type checking and parameterization
- ✅ **Error messages** do not leak sensitive database information
- ✅ **Proper escaping** of all input values before SQL execution

> **Note**: For production deployments, use a reverse proxy (nginx, Traefik, etc.) for authentication, rate limiting, and SSL termination.

---

## 🏗️ Architecture

```
┌─────────────┐
│   Client    │
│  (HTTP/MCP) │
└──────┬──────┘
       │
       ▼
┌──────────────────────────────────────┐
│       Hyperterse Runtime Server      │
│  ┌────────────────────────────────┐  │
│  │   HTTP Server (Port 8080)      │  │
│  │  - Query Endpoints             │  │
│  │  - MCP JSON-RPC 2.0 Endpoint   │  │
│  │  - Documentation Endpoints     │  │
│  └──────────────┬─────────────────┘  │
│                 │                    │
│  ┌──────────────▼─────────────────┐  │
│  │      Query Executor            │  │
│  │  - Input Validation            │  │
│  │  - Template Substitution       │  │
│  │  - Query Execution             │  │
│  └──────────────┬─────────────────┘  │
│                 │                    │
│  ┌──────────────▼─────────────────┐  │
│  │    Connector Layer             │  │
│  │  - PostgreSQL Connector        │  │
│  │  - MySQL Connector             │  │
│  │  - Redis Connector             │  │
│  └──────────────┬─────────────────┘  │
└─────────────────┼────────────────────┘
                   │
        ┌──────────┼──────────┐
        │          │          │
        ▼          ▼          ▼
   ┌────────┐ ┌────────┐ ┌────────┐
   │Postgres│ │ MySQL  │ │ Redis  │
   └────────┘ └────────┘ └────────┘
```

---

## 🧪 Development

### Prerequisites

- Go 1.25.1 or later
- `protoc` (Protocol Buffers compiler)
- `protoc-gen-go` (Go protobuf plugin)

### Setup

```bash
# Clone repository
git clone https://github.com/hyperterse/hyperterse.git
cd hyperterse

# Complete setup (installs dependencies, generates code)
make setup

# Build project
make build

# Run tests
go test ./...
```

### Available Make Targets

```bash
make help        # Show all available targets
make setup       # Complete setup (install deps, generate code)
make generate    # Generate protobuf files only
make build       # Generate code and build the project
make run         # Build and run (requires CONFIG_FILE env var)
```

### Hot Reloading

For development with hot reloading:

```bash
# Install air (hot reload tool)
go install github.com/air-verse/air@latest

# Run with air
air
```

Or use the built-in dev mode:

```bash
hyperterse dev -f config.terse
```

---

## 📝 Contributing

We welcome contributions! Here's how you can help:

1. **Fork the repository**
2. **Create a feature branch** (`git checkout -b feature/amazing-feature`)
3. **Make your changes**
4. **Add tests** for new functionality
5. **Ensure all tests pass** (`go test ./...`)
6. **Commit your changes** (`git commit -m 'Add amazing feature'`)
7. **Push to the branch** (`git push origin feature/amazing-feature`)
8. **Open a Pull Request**

### Code Style

- Follow Go standard formatting (`go fmt`)
- Use meaningful variable names
- Add comments for exported functions
- Keep functions focused and small
- Write tests for new features

### Reporting Issues

Found a bug? Have a feature request? Please [open an issue](https://github.com/hyperterse/hyperterse/issues) with:
- Clear description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)

---

## 🤝 Support

- **📚 Documentation**: [Full Documentation](HYPERTERSE.md) | [API Reference](HYPERTERSE.md#api-reference)
- **🐛 Issues**: [GitHub Issues](https://github.com/hyperterse/hyperterse/issues)
- **💬 Discussions**: [GitHub Discussions](https://github.com/hyperterse/hyperterse/discussions)
- **📦 Releases**: [GitHub Releases](https://github.com/hyperterse/hyperterse/releases)

---

<div align="center">

**Made with ❤️ by the Hyperterse team**

[⭐ Star us on GitHub](https://github.com/hyperterse/hyperterse) • [📖 Read the Docs](HYPERTERSE.md) • [🐛 Report a Bug](https://github.com/hyperterse/hyperterse/issues)

</div>
