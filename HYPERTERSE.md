# Hyperterse: Comprehensive Documentation

> **Last Updated:** 2026-01-07
> **Status:** Living Document - Always Updated

## Table of Contents

1. [What is Hyperterse?](#what-is-hyperterse)
2. [Core Concepts](#core-concepts)
3. [Architecture Overview](#architecture-overview)
4. [Key Features](#key-features)
5. [Configuration](#configuration)
6. [CLI Reference](#cli-reference)
7. [API Reference](#api-reference)
8. [Protocols & Standards](#protocols--standards)
9. [Security Model](#security-model)
10. [Development Guide](#development-guide)
11. [Extension Points](#extension-points)
12. [Troubleshooting](#troubleshooting)

---

## What is Hyperterse?

**Hyperterse** is a high-performance runtime server that transforms database queries into RESTful API endpoints and MCP (Model Context Protocol) tools. It acts as a **query gateway** that:

- **Exposes database queries as APIs**: Define queries in YAML or DSL, and Hyperterse automatically generates individual REST endpoints for each query
- **Provides AI integration**: Exposes queries as MCP tools for AI assistants and LLMs via JSON-RPC 2.0
- **Generates documentation**: Auto-generates OpenAPI 3.0 specifications and LLM-friendly markdown documentation
- **Validates inputs**: Automatic type checking and validation for all query inputs
- **Supports multiple databases**: PostgreSQL, MySQL, and Redis connectors out of the box
- **Maintains security**: Connection strings and raw SQL never exposed to clients
- **Hot reloading**: Development mode with automatic reload on configuration changes

### Use Cases

- **API Gateway for Databases**: Quickly expose database queries as REST APIs without writing boilerplate code
- **AI/LLM Integration**: Make database queries available to AI assistants via MCP protocol
- **Microservices**: Create lightweight query services without full ORM overhead
- **Data APIs**: Build data access layers with automatic documentation and validation
- **Rapid Prototyping**: Define queries in configuration files and immediately have working APIs

### What Hyperterse is NOT

- **Not an ORM**: Hyperterse doesn't abstract away SQL - you write SQL queries directly
- **Not a database migration tool**: It doesn't manage schema changes or migrations
- **Not a query builder**: You write raw SQL/commands, not query builder syntax
- **Not a full application framework**: It's focused solely on query execution and API exposure

---

## Core Concepts

### Adapters

**Adapters** define database connections. Each adapter specifies:

- **Name**: Unique identifier (lower-kebab-case or lower_snake_case)
- **Connector Type**: `postgres`, `mysql`, or `redis`
- **Connection String**: Database connection string (never exposed to clients)
- **Options**: Connector-specific configuration (optional)

**Example:**

```yaml
adapters:
  production_db:
    connector: postgres
    connection_string: "postgresql://user:pass@host:5432/dbname"
    options:
      max_connections: "10"
      ssl_mode: "require"
```

### Queries

**Queries** define database operations that become API endpoints. Each query specifies:

- **Name**: Query identifier (becomes endpoint name: `/query/{name}`)
- **Use**: Adapter(s) to use for execution
- **Description**: Human-readable description (used in documentation)
- **Statement**: SQL query or command with template variables (`{{ inputs.fieldName }}`)
- **Inputs**: Input parameter definitions (optional)
- **Data**: Output schema definition (optional, for documentation)

**Example:**

```yaml
queries:
  get-user-by-id:
    use: production_db
    description: "Retrieve a user by their unique ID"
    statement: |
      SELECT id, name, email, created_at
      FROM users
      WHERE id = {{ inputs.userId }}
    inputs:
      userId:
        type: int
        description: "Unique user identifier"
        optional: false
    data:
      id:
        type: int
        description: "User ID"
      name:
        type: string
        description: "User's full name"
      email:
        type: string
        description: "Email address"
      created_at:
        type: datetime
        description: "Account creation timestamp"
```

### Template Variables

Queries use template variables to inject input values:

- **Syntax**: `{{ inputs.fieldName }}`
- **Substitution**: Values are properly escaped and formatted for SQL
- **Validation**: All referenced inputs must be defined in the query's `inputs` section

**Example:**

```sql
SELECT * FROM products
WHERE category = {{ inputs.category }}
AND price <= {{ inputs.maxPrice }}
ORDER BY created_at DESC
LIMIT {{ inputs.limit }}
```

### Input Types

Supported primitive types:

- `string` - Text values
- `int` - Integer numbers (64-bit)
- `float` - Floating-point numbers (64-bit)
- `boolean` - True/false values
- `uuid` - UUID strings
- `datetime` - ISO 8601 datetime strings (RFC3339)

### Optional Inputs

Inputs can be marked as optional with default values:

```yaml
inputs:
  limit:
    type: int
    description: "Maximum number of results"
    optional: true
    default: "20"
```

---

## Architecture Overview

### High-Level Architecture

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

### Package Structure (Rust Cargo Workspace)

```
hyperterse/
├── Cargo.toml                        # Workspace root manifest
├── package.json                      # Build scripts (Bun Shell)
├── src/
│   └── modules/                      # Rust crates
│       ├── cli/                      # CLI crate (hyperterse-cli)
│       │   ├── Cargo.toml
│       │   ├── lib.rs                # Library entry point
│       │   ├── main.rs               # Binary entry point
│       │   └── commands/             # CLI commands
│       │       ├── mod.rs            # Commands module
│       │       ├── run.rs            # 'run' command - start server
│       │       ├── dev.rs            # 'dev' command - hot reload mode
│       │       ├── generate.rs       # 'generate' command (llms, skills)
│       │       ├── init.rs           # 'init' command - project scaffolding
│       │       ├── upgrade.rs        # 'upgrade' command
│       │       └── export.rs         # 'export' command
│       ├── core/                     # Core domain models (hyperterse-core)
│       │   ├── Cargo.toml
│       │   ├── lib.rs                # Library entry point
│       │   ├── error.rs              # Error types (HyperterseError)
│       │   └── domain/               # Domain models
│       │       ├── mod.rs
│       │       ├── adapter.rs        # Adapter model
│       │       ├── model.rs          # Model (root config)
│       │       ├── query.rs          # Query and Input models
│       │       └── types.rs          # ServerConfig, PoolConfig, etc.
│       ├── parser/                   # Configuration parsing (hyperterse-parser)
│       │   ├── Cargo.toml
│       │   ├── lib.rs                # Library entry point
│       │   ├── yaml.rs               # YAML parser (serde_yaml)
│       │   ├── validator.rs          # Configuration validation
│       │   └── env.rs                # Environment variable substitution
│       ├── runtime/                  # Runtime server (hyperterse-runtime)
│       │   ├── Cargo.toml
│       │   ├── lib.rs                # Library entry point
│       │   ├── server.rs             # HTTP server (Axum)
│       │   ├── connectors/           # Database connectors
│       │   │   ├── mod.rs
│       │   │   ├── traits.rs         # Connector trait
│       │   │   ├── manager.rs        # ConnectorManager
│       │   │   ├── postgres.rs       # PostgreSQL (sqlx)
│       │   │   ├── mysql.rs          # MySQL (sqlx)
│       │   │   ├── redis.rs          # Redis
│       │   │   └── mongodb.rs        # MongoDB
│       │   ├── executor/             # Query execution
│       │   │   ├── mod.rs
│       │   │   ├── substitutor.rs    # Template substitution
│       │   │   └── validator.rs      # Input validation
│       │   └── handlers/             # HTTP handlers
│       │       ├── mod.rs
│       │       ├── query.rs          # Query endpoint handler
│       │       ├── mcp.rs            # MCP JSON-RPC handler
│       │       ├── openapi.rs        # OpenAPI spec generation
│       │       └── llms.rs           # LLM documentation generation
│       └── types/                    # Shared types (hyperterse-types)
│           ├── Cargo.toml
│           ├── lib.rs                # Library entry point
│           ├── connector.rs          # Connector enum
│           ├── primitive.rs          # Primitive enum
│           └── runtime.rs            # Runtime response types
├── scripts/                          # Build scripts (Bun Shell)
│   ├── build.ts                      # Build script
│   ├── test.ts                       # Test runner
│   ├── setup.ts                      # Development setup
│   ├── version.ts                    # Version management
│   ├── release.ts                    # Release builds
│   ├── archive.ts                    # Binary archiving
│   ├── check.ts                      # Cargo check
│   ├── lint.ts                       # Clippy linting
│   ├── fmt.ts                        # Code formatting
│   ├── clean.ts                      # Clean build artifacts
│   └── help.ts                       # Show available commands
├── schema/                           # JSON Schema for config files
│   └── terse.schema.json
└── distributions/                    # Distribution packages
    ├── homebrew/
    │   └── hyperterse.rb
    └── npm/
        └── package.json
```

### Request Flow

1. **Client Request** → HTTP POST to `/query/{query-name}`
2. **Server** → Routes to query handler
3. **Handler** → Parses JSON body, extracts inputs
4. **Executor** → Validates inputs against query definition
5. **Substitutor** → Replaces `{{ inputs.x }}` in SQL statement
6. **Connector** → Executes SQL against database
7. **Response** → Formats results as JSON and returns

### Configuration Flow

1. **Parse Configuration** → `.terse` file parsed into a language model
2. **Validate** → Comprehensive validation of adapters, queries, inputs
3. **Initialize Connectors** → Create database connections for each adapter
4. **Register Endpoints** → Dynamically create HTTP endpoints for each query
5. **Start Server** → Begin listening on configured port

---

## Key Features

### 1. Automatic Endpoint Generation

Each query automatically becomes a REST endpoint:

- **Path**: `POST /query/{query-name}`
- **Request**: JSON body with input parameters
- **Response**: JSON with `success`, `error`, and `results` fields

### 2. OpenAPI 3.0 Compliance

- **Endpoint**: `GET /docs`
- **Specification**: Complete OpenAPI 3.0 spec with:
  - All query endpoints documented
  - Request/response schemas
  - Input validation rules
  - Example values
  - Error responses

### 3. MCP Protocol Support

- **JSON-RPC 2.0 Endpoint**: `POST /mcp` - MCP protocol endpoint using JSON-RPC 2.0
- **Methods**:
  - `tools/list` - Returns all queries as MCP tools
  - `tools/call` - Execute a query via MCP protocol
- **Tool Descriptions**: Queries exposed with descriptions and input schemas

### 4. LLM-Friendly Documentation

- **Endpoint**: `GET /llms.txt`
- **Format**: Markdown documentation optimized for AI consumption
- **Content**: Complete API reference with examples and usage patterns

### 5. Input Validation

- **Type Checking**: Automatic validation of input types
- **Required Fields**: Enforces required inputs
- **Default Values**: Applies defaults for optional inputs
- **Error Messages**: Clear, descriptive validation errors

### 6. Multi-Database Support

- **PostgreSQL**: Full SQL support via `sqlx` with `PgPool`
- **MySQL**: Full SQL support via `sqlx` with `MySqlPool`
- **Redis**: Command execution via `redis` crate with async support
- **MongoDB**: Full document operations via `mongodb` crate with async client

### 7. Security Features

- **Connection String Protection**: Never exposed in API responses or documentation
- **SQL Injection Prevention**: Proper escaping and parameterization
- **Error Message Sanitization**: No sensitive database information leaked
- **Input Validation**: Type checking prevents injection attacks

### 8. Development Mode with Hot Reload

- **Command**: `hyperterse dev -f config.terse`
- **Auto-reload**: Watches configuration file for changes
- **Debounced**: 500ms delay to handle rapid file saves
- **Graceful**: Maintains server uptime during reloads

---

## Configuration

### Configuration File Format

Hyperterse supports two configuration formats:

#### Terse format (`.terse`)

```yaml
name: my-api

# Optional server configuration
server:
  port: "8080"
  log_level: 3 # 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG

adapters:
  my_database:
    connector: postgres
    connection_string: "postgresql://user:pass@host:5432/db"
    options:
      max_connections: "10"

queries:
  get-user:
    use: my_database
    description: "Get user by email"
    statement: |
      SELECT id, name, email
      FROM users
      WHERE email = {{ inputs.email }}
    inputs:
      email:
        type: string
        description: "User email address"
    data:
      id:
        type: int
      name:
        type: string
      email:
        type: string
```

#### DSL Format (`.hyperterse`)

```
adapter my_database {
  connector: postgres
  connection_string: "postgresql://user:pass@host:5432/db"
  options: {
    max_connections: "10"
  }
}

query get-user {
  use: my_database
  description: "Get user by email"
  statement: "SELECT id, name, email FROM users WHERE email = {{ inputs.email }}"
  inputs: {
    email: {
      type: string
      description: "User email address"
    }
  }
  data: {
    id: {
      type: int
    }
    name: {
      type: string
    }
    email: {
      type: string
    }
  }
}
```

### Server Configuration

Server configuration can be specified in the config file or overridden via CLI flags:

```yaml
server:
  port: "8080" # Server port (default: 8080)
  log_level: 3 # 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG (default: 3)
```

**Priority order (highest to lowest):**

1. CLI flags (`-p`, `--log-level`, `-v`)
2. Config file (`server.port`, `server.log_level`)
3. Environment variables (`PORT`)
4. Defaults (`8080`, `INFO`)

### Naming Conventions

- **Adapter Names**: `lower-kebab-case` or `lower_snake_case`, must start with letter
- **Query Names**: `lower-kebab-case` or `lower_snake_case`, must start with letter
- **Input Names**: Any valid identifier
- **Data Names**: Any valid identifier

### Validation Rules

- At least one adapter required
- At least one query required
- Adapter names must be unique
- Query names must be unique
- Query `use` must reference valid adapter(s)
- All `{{ inputs.x }}` references must be defined in inputs
- Optional inputs must have default values
- All types must be valid primitives

---

## CLI Reference

### Global Flags

| Flag     | Short | Description                         |
| -------- | ----- | ----------------------------------- |
| `--file` | `-f`  | Path to configuration file (.terse) |

### Commands

#### `hyperterse` or `hyperterse run`

Run the Hyperterse server.

```bash
hyperterse -f config.terse
# or
hyperterse run -f config.terse
```

**Flags:**

| Flag          | Short | Description                                          |
| ------------- | ----- | ---------------------------------------------------- |
| `--port`      | `-p`  | Server port (overrides config file and PORT env var) |
| `--log-level` |       | Log level: 1=ERROR, 2=WARN, 3=INFO, 4=DEBUG          |
| `--verbose`   | `-v`  | Enable verbose logging (sets log level to DEBUG)     |

#### `hyperterse dev`

Run the server in development mode with hot reload.

```bash
hyperterse dev -f config.terse
```

The server will automatically reload when the configuration file changes.

**Flags:** Same as `run` command.

#### `hyperterse generate llms`

Generate an llms.txt documentation file.

```bash
hyperterse generate llms -f config.terse
hyperterse generate llms -f config.terse -o docs/llms.txt --base-url https://api.example.com
```

**Flags:**

| Flag         | Short | Default                 | Description                       |
| ------------ | ----- | ----------------------- | --------------------------------- |
| `--output`   | `-o`  | `llms.txt`              | Output path for the llms.txt file |
| `--base-url` |       | `http://localhost:8080` | Base URL for the API endpoints    |

#### `hyperterse generate skills`

Generate an Agent Skills compatible archive (.zip) for AI platforms.

```bash
hyperterse generate skills -f config.terse
hyperterse generate skills -f config.terse -o my-skill.zip --name my-api-skill
```

**Flags:**

| Flag       | Short | Default                        | Description                        |
| ---------- | ----- | ------------------------------ | ---------------------------------- |
| `--output` | `-o`  | `skill.zip`                    | Output path for the skills archive |
| `--name`   |       | (derived from config filename) | Skill name                         |

---

## API Reference

### Query Endpoints

#### Execute Query

**Endpoint:** `POST /query/{query-name}`

**Request:**

```json
{
  "input1": "value1",
  "input2": 42,
  "input3": true
}
```

**Response (Success):**

```json
{
  "success": true,
  "error": "",
  "results": [
    {
      "field1": "value1",
      "field2": 42,
      "field3": "2024-01-01T00:00:00Z"
    }
  ]
}
```

**Response (Error):**

```json
{
  "success": false,
  "error": "validation error for field 'userId': required input 'userId' is missing",
  "results": []
}
```

### Utility Endpoints

#### MCP JSON-RPC 2.0 Endpoint

**Endpoint:** `POST /mcp`

The MCP protocol uses JSON-RPC 2.0 messages over HTTP POST. All MCP requests are sent to this single endpoint.

##### List MCP Tools (`tools/list` method)

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "tools/list",
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "tools": [
      {
        "name": "get-user-by-id",
        "description": "Retrieve user by ID",
        "inputSchema": {
          "type": "object",
          "properties": {
            "userId": {
              "type": "int",
              "description": "User ID"
            }
          },
          "required": ["userId"]
        }
      }
    ]
  },
  "id": 1
}
```

##### Call MCP Tool (`tools/call` method)

**Request:**

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "get-user-by-id",
    "arguments": {
      "userId": "123"
    }
  },
  "id": 1
}
```

**Response:**

```json
{
  "jsonrpc": "2.0",
  "result": {
    "content": [
      {
        "type": "text",
        "text": "[{\"id\":123,\"name\":\"John Doe\",\"email\":\"john@example.com\"}]"
      }
    ],
    "isError": false
  },
  "id": 1
}
```

#### Get OpenAPI Specification

**Endpoint:** `GET /docs`

**Response:** Complete OpenAPI 3.0 JSON specification

#### Get LLM Documentation

**Endpoint:** `GET /llms.txt`

**Response:** Markdown documentation for LLMs

---

## Protocols & Standards

### REST API

- **Base URL**: `http://localhost:8080` (configurable via `PORT` env var, config file, or CLI flag)
- **Content-Type**: `application/json`
- **Methods**: `GET` (utility endpoints), `POST` (query endpoints)
- **Status Codes**: `200` (success), `400` (bad request), `500` (server error)

### MCP Protocol

Hyperterse implements the Model Context Protocol (MCP) for AI assistant integration:

- **JSON-RPC 2.0**: Uses JSON-RPC 2.0 messages over HTTP POST
- **Endpoint**: Single `/mcp` endpoint handles all MCP requests
- **Methods**:
  - `tools/list` - List all available tools (queries)
  - `tools/call` - Execute a tool (query)
- **Tool Descriptions**: Include input schemas and descriptions
- **Tool Execution**: Execute queries via MCP JSON-RPC interface

### OpenAPI 3.0

- **Version**: 3.0.0
- **Specification**: Auto-generated from query definitions
- **Validation**: Uses `libopenapi` for spec validation
- **Schemas**: Complete request/response schemas for all endpoints

---

## Security Model

### Connection String Protection

- **Never Exposed**: Connection strings are never included in:
  - API responses
  - OpenAPI documentation
  - MCP tool descriptions
  - LLM documentation
  - Error messages

### SQL Injection Prevention

- **Template Substitution**: Values are properly escaped before substitution
- **Type Validation**: Input types are validated before use
- **Parameterization**: Values are formatted according to their types
- **String Escaping**: Single quotes are escaped in string values

### Error Message Sanitization

- **No Sensitive Data**: Error messages don't leak:
  - Database connection details
  - Table/column names (in some cases)
  - Internal query structure
  - Stack traces (in production)

### Input Validation

- **Type Checking**: All inputs validated against declared types
- **Required Fields**: Missing required inputs rejected
- **Unknown Fields**: Extra fields rejected
- **Default Values**: Applied only for optional inputs

---

## Development Guide

### Prerequisites

- **Rust**: 1.75+ (install via [rustup](https://rustup.rs/))
- **Bun**: 1.0+ (install via [bun.sh](https://bun.sh/))

### Building

```bash
# Set up development environment (checks dependencies, installs components)
bun run setup

# Build debug version
bun run build

# Build release version
bun run build:release

# Build for all platforms (cross-compilation)
bun run build:all
```

### Running

```bash
# Run with `.terse` config (debug build)
cargo run -- run -f config.terse

# Run with custom port
cargo run -- run -f config.terse -p 3000

# Run with verbose logging
cargo run -- run -f config.terse -v

# Development mode with hot reload
cargo run -- dev -f config.terse

# Using release binary
./target/release/hyperterse run -f config.terse
```

### Available Scripts

```bash
# Show all available commands
bun run help

# Build commands
bun run build              # Build debug version
bun run build:release      # Build release version
bun run build:all          # Build for all targets

# Test commands
bun run test               # Run all tests
bun run test:unit          # Run unit tests only
bun run test:ignored       # Run ignored tests (integration)

# Code quality
bun run lint               # Run clippy linter
bun run lint:fix           # Auto-fix lint issues
bun run fmt                # Format code
bun run fmt:check          # Check formatting
bun run check              # Fast compilation check

# Version management
bun run version:patch      # Bump patch version
bun run version:minor      # Bump minor version
bun run version:major      # Bump major version

# Release
bun run release:build      # Build release for current platform
bun run clean              # Clean build artifacts
```

### Testing

```bash
# Run all tests
bun run test
# or
cargo test

# Run tests with coverage (requires cargo-llvm-cov)
bun run test --coverage

# Run specific test
cargo test test_name

# Run tests for specific crate
cargo test -p hyperterse-runtime
```

### Development Workflow

1. **Define Queries**: Create/update `.terse` configuration
2. **Run Dev Mode**: `hyperterse dev -f config.terse` for hot reload
3. **Test**: Use `curl` or HTTP client to test endpoints
4. **Documentation**: Check `/docs` and `/llms.txt` for generated docs
5. **Iterate**: Edit configuration, server reloads automatically

### Hot Reloading

Development mode (`dev` command) includes built-in hot reloading for configuration files:

```bash
# Start dev server
cargo run -- dev -f config.terse

# Edit config.terse - server reloads automatically
```

The `dev` command uses the `notify` crate to watch for file changes and automatically reloads the configuration. Changes are debounced to handle rapid file saves.

For source code changes during development, you can use `cargo watch`:

```bash
# Install cargo-watch
cargo install cargo-watch

# Run with automatic rebuild on source changes
cargo watch -x 'run -- run -f config.terse'
```

---

## Extension Points

### Adding New Connectors

1. **Define Connector Type**: Add variant to `Connector` enum in `hyperterse-types/src/connector.rs`
2. **Implement Connector Trait**: Create `hyperterse-runtime/src/connectors/{connector}.rs` implementing the `Connector` trait
3. **Export Module**: Add `pub mod {connector};` to `hyperterse-runtime/src/connectors/mod.rs`
4. **Update Manager**: Add case to `create_connector()` in `manager.rs`
5. **Add Dependencies**: Add required crate to `hyperterse-runtime/Cargo.toml`
6. **Test**: Add tests for the new connector

### Adding New Primitive Types

1. **Define Type**: Add variant to `Primitive` enum in `hyperterse-types/src/primitive.rs`
2. **Implement Traits**: Add `Display`, `FromStr`, and validation methods
3. **Update Validator**: Add type conversion in `hyperterse-runtime/src/executor/validator.rs`
4. **Update OpenAPI**: Add JSON Schema mapping in `hyperterse-runtime/src/handlers/openapi.rs`
5. **Test**: Add validation tests

### Adding New Parsers

1. **Create Parser**: Add parser module in `hyperterse-parser/src/`
2. **Export Module**: Add `pub mod {parser};` to `hyperterse-parser/src/lib.rs`
3. **Update Loader**: Add file extension detection in `hyperterse-cli/src/commands/`
4. **Test**: Add parser tests

### Custom Handlers

1. **Create Handler**: Implement Axum handler in `hyperterse-runtime/src/handlers/`
2. **Export Module**: Add handler module to `mod.rs`
3. **Register Route**: Add route in `server.rs` router builder
4. **Documentation**: Update OpenAPI spec generation if needed
5. **Test**: Add handler tests

---

## Troubleshooting

### Common Issues

#### Connection Errors

**Problem**: "failed to ping postgres database"

**Solutions**:

- Verify connection string format
- Check database is running and accessible
- Verify credentials
- Check firewall/network settings

#### Validation Errors

**Problem**: "validation error for field 'x'"

**Solutions**:

- Check input name matches query definition
- Verify input type matches expected type
- Ensure required inputs are provided
- Check default values for optional inputs

#### Template Substitution Errors

**Problem**: "input 'x' not found for substitution"

**Solutions**:

- Verify `{{ inputs.x }}` matches input name exactly
- Check input is defined in query's `inputs` section
- Ensure input name uses correct casing

#### Query Execution Errors

**Problem**: "query execution failed"

**Solutions**:

- Verify SQL syntax is correct
- Check table/column names exist
- Verify user has necessary permissions
- Check database logs for detailed errors

### Debugging

#### Log Levels

Use the `--log-level` flag or `-v` for verbose output:

```bash
# Debug level logging
./dist/hyperterse -f config.terse --log-level 4

# or use verbose shortcut
./dist/hyperterse -f config.terse -v
```

Log levels:

- `1` = ERROR - Only errors
- `2` = WARN - Warnings and errors
- `3` = INFO - General information (default)
- `4` = DEBUG - Detailed debugging information

#### Check Generated Documentation

- **OpenAPI**: `GET /docs` - Verify endpoint definitions
- **LLM Docs**: `GET /llms.txt` - Check query documentation

#### Test Queries Directly

Use `curl` to test endpoints:

```bash
curl -X POST http://localhost:8080/query/get-user \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

---

## Changelog

### Version History

**Current Version**: Development

**Notable Changes**:

- CLI with subcommands: `run`, `dev`, `generate llms`, `generate skills`
- Hot reload development mode
- YAML and DSL configuration support
- Server configuration in config files
- PostgreSQL, MySQL, Redis connectors
- OpenAPI 3.0 generation
- MCP protocol support via JSON-RPC 2.0
- LLM documentation generation
- Agent Skills archive generation
- Configurable log levels

---

## Contributing

### Code Style

- Follow Rust standard formatting (`cargo fmt`)
- Run clippy for linting (`cargo clippy`)
- Use meaningful variable names
- Add doc comments for public items (`///` documentation comments)
- Keep functions focused and small
- Prefer `Result` and `Option` over panics

### Testing

- Write tests for new features
- Maintain test coverage
- Test error cases
- Test edge cases
- Use `#[tokio::test]` for async tests

### Documentation

- Update this document when adding features
- Add doc comments for complex logic
- Update README.md for user-facing changes
- Keep examples up to date

### Pull Request Process

1. Run all checks: `bun run lint && bun run fmt:check && bun run test`
2. Update documentation if needed
3. Keep commits focused and well-described
4. Request review from maintainers

---

## License

[Add license information]

---

## Support

- **Documentation**: See this file and README.md
- **Issues**: [GitHub Issues](https://github.com/hyperterse/hyperterse/issues)
- **Discussions**: [GitHub Discussions](https://github.com/hyperterse/hyperterse/discussions)

---

**Note**: This is a living document. Update it whenever:

- New features are added
- Architecture changes
- API changes
- New connectors or types are added
- Security considerations change
