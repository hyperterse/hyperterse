# Hyperterse: Comprehensive Documentation

> **Last Updated:** 2024-12-19  
> **Status:** Living Document - Always Updated

## Table of Contents

1. [What is Hyperterse?](#what-is-hyperterse)
2. [Core Concepts](#core-concepts)
3. [Architecture Overview](#architecture-overview)
4. [Key Features](#key-features)
5. [Configuration](#configuration)
6. [API Reference](#api-reference)
7. [Protocols & Standards](#protocols--standards)
8. [Security Model](#security-model)
9. [Development Guide](#development-guide)
10. [Extension Points](#extension-points)
11. [Troubleshooting](#troubleshooting)

---

## What is Hyperterse?

**Hyperterse** is a high-performance runtime server that transforms database queries into RESTful API endpoints and MCP (Model Context Protocol) tools. It acts as a **query gateway** that:

- **Exposes database queries as APIs**: Define queries in YAML or DSL, and Hyperterse automatically generates individual REST endpoints for each query
- **Provides AI integration**: Exposes queries as MCP tools for AI assistants and LLMs
- **Generates documentation**: Auto-generates OpenAPI 3.0 specifications and LLM-friendly markdown documentation
- **Validates inputs**: Automatic type checking and validation for all query inputs
- **Supports multiple databases**: PostgreSQL, MySQL, and Redis connectors out of the box
- **Maintains security**: Connection strings and raw SQL never exposed to clients

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
┌─────────────────────────────────────┐
│      Hyperterse Runtime Server      │
│  ┌──────────────────────────────┐  │
│  │   HTTP Server (Port 8080)     │  │
│  │  - Query Endpoints             │  │
│  │  - MCP Endpoints               │  │
│  │  - Documentation Endpoints     │  │
│  └──────────────┬─────────────────┘  │
│                 │                     │
│  ┌──────────────▼─────────────────┐  │
│  │      Query Executor            │  │
│  │  - Input Validation            │  │
│  │  - Template Substitution       │  │
│  │  - Query Execution             │  │
│  └──────────────┬─────────────────┘  │
│                 │                     │
│  ┌──────────────▼─────────────────┐  │
│  │    Connector Layer             │  │
│  │  - PostgreSQL Connector       │  │
│  │  - MySQL Connector            │  │
│  │  - Redis Connector           │  │
│  └──────────────┬─────────────────┘  │
└─────────────────┼───────────────────┘
                   │
        ┌──────────┼──────────┐
        │          │          │
        ▼          ▼          ▼
   ┌────────┐ ┌────────┐ ┌────────┐
   │Postgres│ │ MySQL  │ │ Redis  │
   └────────┘ └────────┘ └────────┘
```

### Package Structure

```
hyperterse/
├── main.go                    # Application entry point
├── core/
│   ├── parser/                # Configuration parsing
│   │   ├── yaml.go            # YAML parser
│   │   ├── dsl.go             # DSL parser
│   │   └── validator.go       # Configuration validation
│   ├── runtime/               # Runtime server & execution
│   │   ├── server.go          # HTTP server
│   │   ├── handlers.go        # Request handlers
│   │   ├── executor.go        # Query executor
│   │   ├── substitutor.go    # Template substitution
│   │   ├── executor_validator.go  # Input validation
│   │   ├── connector.go       # Connector interface
│   │   ├── connector_factory.go  # Connector factory
│   │   ├── postgres.go        # PostgreSQL connector
│   │   ├── mysql.go           # MySQL connector
│   │   ├── redis.go           # Redis connector
│   │   ├── openapi_handler.go # OpenAPI generation
│   │   └── llms_txt_handler.go # LLM documentation
│   ├── types/                 # Type definitions
│   │   ├── connectors.go      # Connector types
│   │   └── primitives.go      # Primitive types
│   └── logger/                # Logging utilities
│       └── logger.go
├── pkg/
│   └── pb/                    # Generated protobuf code
└── proto/                     # Protocol buffer definitions
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

1. **Parse Configuration** → YAML or DSL file parsed into protobuf Model
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

- **PostgreSQL**: Full SQL support via `lib/pq` driver
- **MySQL**: Full SQL support via `go-sql-driver/mysql`
- **Redis**: Command execution via `go-redis/v9`

### 7. Security Features

- **Connection String Protection**: Never exposed in API responses or documentation
- **SQL Injection Prevention**: Proper escaping and parameterization
- **Error Message Sanitization**: No sensitive database information leaked
- **Input Validation**: Type checking prevents injection attacks

---

## Configuration

### Configuration File Format

Hyperterse supports two configuration formats:

#### YAML Format (`.yaml`, `.yml`)

```yaml
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
  "results": null
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

- **Base URL**: `http://localhost:8080` (configurable via `PORT` env var)
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

### Building

```bash
# Install dependencies
go mod download

# Generate protobuf code
make generate

# Build binary
make build
# or
go build -o hyperterse .
```

### Running

```bash
# Run with YAML config
./hyperterse -file config.yaml

# Run with DSL config
./hyperterse -file config.hyperterse

# Custom port
PORT=3000 ./hyperterse -file config.yaml
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for specific package
go test ./core/runtime/...
```

### Development Workflow

1. **Define Queries**: Create/update YAML or DSL configuration
2. **Validate**: Hyperterse validates configuration on startup
3. **Test**: Use `curl` or HTTP client to test endpoints
4. **Documentation**: Check `/docs` and `/llms.txt` for generated docs
5. **Iterate**: Update configuration and restart server

### Hot Reloading

For development, use `air` for hot reloading:
```bash
# Install air
go install github.com/air-verse/air@latest

# Run with air
air
```

---

## Extension Points

### Adding New Connectors

1. **Define Connector Type**: Add to `proto/hyperterse/hyperterse.proto` (Connector enum)
2. **Implement Interface**: Create `core/runtime/{connector}.go` implementing `Connector` interface
3. **Update Factory**: Add case to `NewConnector()` in `connector_factory.go`
4. **Update Types**: Regenerate `core/types/connectors.go` (or update manually)
5. **Test**: Add tests for new connector

### Adding New Primitive Types

1. **Define Type**: Add to `proto/hyperterse/hyperterse.proto` (Primitive enum)
2. **Add Conversion**: Implement conversion in `executor_validator.go`
3. **Update Types**: Regenerate `core/types/primitives.go`
4. **Update OpenAPI**: Add mapping in `openapi_handler.go`
5. **Test**: Add validation tests

### Adding New Parsers

1. **Create Parser**: Implement parser function in `core/parser/`
2. **File Detection**: Add extension detection in `main.go`
3. **Integration**: Call parser from main entry point
4. **Test**: Add parser tests

### Custom Handlers

1. **Create Handler**: Implement HTTP handler function
2. **Register Route**: Add route in `server.go` `Start()` method
3. **Documentation**: Update OpenAPI spec generation if needed
4. **Test**: Add handler tests

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

#### Enable Verbose Logging

Logging is built-in and shows:
- Configuration parsing
- Adapter initialization
- Query execution
- Error details

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
- Initial implementation
- YAML and DSL parser support
- PostgreSQL, MySQL, Redis connectors
- OpenAPI 3.0 generation
- MCP protocol support
- LLM documentation generation

---

## Contributing

### Code Style

- Follow Go standard formatting (`go fmt`)
- Use meaningful variable names
- Add comments for exported functions
- Keep functions focused and small

### Testing

- Write tests for new features
- Maintain test coverage
- Test error cases
- Test edge cases

### Documentation

- Update this document when adding features
- Add code comments for complex logic
- Update README.md for user-facing changes
- Keep examples up to date

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

