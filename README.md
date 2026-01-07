# Hyperterse

**Hyperterse** is a high-performance runtime server that exposes database queries as RESTful API endpoints and MCP (Model Context Protocol) tools. Define your database queries in a simple YAML configuration file, and Hyperterse automatically generates individual endpoints for each query with full OpenAPI documentation.

## Features

- 🚀 **Automatic Endpoint Generation**: Each query becomes its own REST endpoint
- 📚 **OpenAPI Compliant**: Full OpenAPI 3.0 specification with Swagger documentation
- 🔧 **MCP Protocol Support**: Expose queries as MCP tools for AI assistants
- 🗄️ **Multi-Database Support**: PostgreSQL, MySQL, and Redis connectors
- ✅ **Input Validation**: Automatic type checking and validation
- 📖 **LLM Documentation**: Auto-generated markdown documentation for AI consumption
- 🔒 **Security First**: Connection strings and raw SQL never exposed to clients

## Installation

### Prerequisites

- **Go 1.25.1 or later** - [Download Go](https://go.dev/dl/)
- **Buf CLI** - Required for protobuf code generation ([Installation instructions](#installing-buf))
- Access to your database (PostgreSQL, MySQL, or Redis)

### Quick Setup

After cloning the repository, follow these steps to get started:

**Option 1: Using Make (Recommended)**

```bash
# 1. Clone the repository
git clone https://github.com/hyperterse/hyperterse.git
cd hyperterse

# 2. Complete setup (installs Buf, downloads deps, generates code)
make setup

# 3. Build the project
make build

# 4. Run the server
./hyperterse -file config.yaml
```

**Option 2: Using Setup Script**

```bash
# 1. Clone the repository
git clone https://github.com/hyperterse/hyperterse.git
cd hyperterse

# 2. Run the setup script
./scripts/setup.sh

# 3. Build the project
go build -o hyperterse

# 4. Run the server
./hyperterse -file config.yaml
```

**Option 3: Manual Setup**

```bash
# 1. Clone the repository
git clone https://github.com/hyperterse/hyperterse.git
cd hyperterse

# 2. Install protoc (if not already installed)
# macOS:
brew install protobuf

# Linux (Debian/Ubuntu):
sudo apt-get install protobuf-compiler

# Linux (RHEL/CentOS):
sudo yum install protobuf-compiler

# 3. Install protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# 4. Download Go dependencies
go mod download

# 5. Generate protobuf code (required before building)
make generate

# 6. Build the project
go build -o hyperterse

# 7. Run the server
./hyperterse -file config.yaml
```

**Available Make Targets:**

- `make setup` - Complete setup (install protoc, download deps, generate code)
- `make generate` - Generate protobuf files only
- `make build` - Generate code and build the project
- `make lint` - Lint proto files (optional, requires buf CLI)
- `make format` - Format proto files (optional, requires buf CLI)
- `make clean` - Remove generated files and binaries
- `make help` - Show all available targets

### Build from Source

If you've already completed the setup steps above:

```bash
go build -o hyperterse
```

## Quick Start

1. **Create a configuration file** (`config.yaml`):

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
    data:
      id:
        type: int
        description: "User ID"
      name:
        type: string
        description: "User's full name"
      email:
        type: string
        description: "User email address"
      created_at:
        type: datetime
        description: "Account creation timestamp"
```

2. **Start the server**:

```bash
./hyperterse -file config.yaml
```

3. **Execute your query**:

```bash
curl -X POST http://localhost:8080/query/get-user \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com"}'
```

## Configuration

### Configuration File Format

Hyperterse supports configuration by specifying it in a YAML file.

### Adapters

Adapters define database connections. Each adapter requires:

- **`name`**: Unique identifier for the adapter
- **`connector`**: Database type (`postgres`, `mysql`, or `redis`)
- **`connection_string`**: Database connection string (required)

Optional:

- **`options`**: Connector-specific options (optional, varies by connector)

#### PostgreSQL

```yaml
adapters:
  my_postgres:
    connector: postgres
    connection_string: "postgresql://user:password@host:5432/database?sslmode=disable"
    options: # Optional: PostgreSQL-specific options
      max_connections: "10"
      ssl_mode: "disable"
```

#### MySQL

```yaml
adapters:
  my_mysql:
    connector: mysql
    connection_string: "user:password@tcp(localhost:3306)/database"
    options: # Optional: MySQL-specific options
      max_connections: "10"
      charset: "utf8mb4"
```

#### Redis

```yaml
adapters:
  my_redis:
    connector: redis
    connection_string: "redis://localhost:6379/0"
    options: # Optional: Redis-specific options
      db: "0"
      password: "your-password"
```

**Note**: The `options` field is optional and contains connector-specific configuration. Each connector may support different options. The connection string should always be specified at the adapter level, not within options.

### Queries

Queries define the database operations exposed as API endpoints. Each query requires:

- **`name`**: Query identifier (becomes the endpoint name)
- **`use`**: Adapter name(s) to use (string or array)
- **`description`**: Human-readable description
- **`statement`**: SQL query or command with template variables
- **`inputs`**: Input parameters (optional)
- **`data`**: Output schema definition (optional)

#### Query Structure

```yaml
queries:
  query-name:
    use: adapter-name # Required: adapter to use
    description: "Query description" # Required: what this query does
    statement: | # Required: SQL/command with {{ inputs.name }} placeholders
      SELECT * FROM table WHERE id = {{ inputs.id }}
    inputs: # Optional: input parameters
      parameter-name:
        type: string # Required: data type
        description: "Parameter description" # Optional
        optional: false # Optional: default false
        default: "default-value" # Optional: default value
    data: # Optional: output schema
      field-name:
        type: string # Required: data type
        description: "Field description" # Optional
        map_to: "object.field" # Optional: mapping hint
```

#### Input Types

Supported input types:

- `string` - Text values
- `int` - Integer numbers
- `float` - Floating-point numbers
- `boolean` - True/false values
- `uuid` - UUID strings
- `datetime` - ISO 8601 datetime strings

#### Template Variables

Use `{{ inputs.parameterName }}` in your SQL statements to inject input values:

```yaml
statement: |
  SELECT * FROM users
  WHERE email = {{ inputs.email }}
  AND age > {{ inputs.minAge }}
```

#### Optional Inputs

Mark inputs as optional and provide defaults:

```yaml
inputs:
  limit:
    type: int
    description: "Maximum number of results"
    optional: true
    default: "10"
```

#### Multiple Adapters

A query can use multiple adapters (array format):

```yaml
queries:
  cross-db-query:
    use:
      - adapter1
      - adapter2
    description: "Query spanning multiple databases"
    statement: |
      SELECT * FROM adapter1.users
      UNION ALL
      SELECT * FROM adapter2.users
```

### Complete Example

```yaml
adapters:
  production_db:
    connector: postgres
    connection_string: "postgresql://user:pass@prod-host:5432/prod_db"

  analytics_db:
    connector: mysql
    connection_string: "user:pass@tcp(analytics-host:3306)/analytics"

queries:
  # Simple query with required input
  get-user-by-id:
    use: production_db
    description: "Retrieve user information by ID"
    statement: |
      SELECT id, name, email, created_at
      FROM users
      WHERE id = {{ inputs.userId }}
    inputs:
      userId:
        type: int
        description: "Unique user identifier"
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
        description: "Account creation date"

  # Query with optional inputs and defaults
  search-users:
    use: production_db
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
    data:
      id:
        type: int
      name:
        type: string
      email:
        type: string

  # Query without inputs
  get-active-users:
    use: production_db
    description: "Get all active users"
    statement: |
      SELECT id, name, email
      FROM users
      WHERE active = true
    data:
      id:
        type: int
      name:
        type: string
      email:
        type: string
```

## Usage

### Starting the Server

```bash
./hyperterse -file config.yaml
```

The server starts on port `8080` by default. Set the `PORT` environment variable to use a different port:

```bash
PORT=3000 ./hyperterse -file config.yaml
```

### API Endpoints

#### Query Endpoints

Each query becomes a `POST` endpoint at `/query/{query-name}`:

```bash
POST /query/get-user-by-id
Content-Type: application/json

{
  "userId": 123
}
```

**Response:**

```json
{
  "success": true,
  "error": "",
  "results": [
    {
      "id": "123",
      "name": "John Doe",
      "email": "john@example.com",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

#### Utility Endpoints

- **`POST /mcp`** - MCP JSON-RPC 2.0 endpoint (supports `tools/list` and `tools/call` methods)
- **`GET /llms.txt`** - Markdown documentation for LLMs
- **`GET /docs`** - OpenAPI 3.0 specification

### MCP Protocol

Hyperterse implements the Model Context Protocol (MCP) using JSON-RPC 2.0. All MCP requests are sent to the `/mcp` endpoint.

List available tools (`tools/list` method):

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/list",
    "id": 1
  }'
```

Execute a tool (`tools/call` method):

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

### OpenAPI Documentation

Access the OpenAPI specification:

```bash
curl http://localhost:8080/docs
```

The specification includes:

- Individual endpoints for each query
- Request/response schemas
- Input validation rules
- Example values

### LLM Documentation

Get markdown documentation optimized for AI consumption:

```bash
curl http://localhost:8080/llms.txt
```

## Input Validation

Hyperterse automatically validates all inputs:

- **Required inputs** must be provided
- **Type checking** ensures values match the declared type
- **Default values** are applied for optional inputs when not provided

### Validation Errors

If validation fails, the API returns a `400 Bad Request`:

```json
{
  "success": false,
  "error": "validation error for field 'userId': required input 'userId' is missing",
  "results": null
}
```

## Environment Variables

- **`PORT`** - Server port (default: `8080`)

## Error Handling

Hyperterse provides clear error messages:

- **400 Bad Request**: Invalid input parameters or validation errors
- **500 Internal Server Error**: Database connection issues or query execution errors

Error responses include a descriptive `error` field:

```json
{
  "success": false,
  "error": "query execution failed: connection refused",
  "results": null
}
```

## Security Considerations

- **Connection strings** are never exposed in API responses or documentation
- **Raw SQL statements** are not included in MCP tool descriptions
- **Input validation** prevents SQL injection through type checking and parameterization
- **Error messages** do not leak sensitive database information

## Examples

### Example 1: User Management

```yaml
adapters:
  user_db:
    connector: postgres
    connection_string: "postgresql://user:pass@localhost:5432/users"

queries:
  create-user:
    use: user_db
    description: "Create a new user account"
    statement: |
      INSERT INTO users (name, email, password_hash)
      VALUES ({{ inputs.name }}, {{ inputs.email }}, {{ inputs.passwordHash }})
      RETURNING id, name, email, created_at
    inputs:
      name:
        type: string
        description: "User's full name"
      email:
        type: string
        description: "Email address"
      passwordHash:
        type: string
        description: "Hashed password"
    data:
      id:
        type: int
      name:
        type: string
      email:
        type: string
      created_at:
        type: datetime

  update-user:
    use: user_db
    description: "Update user information"
    statement: |
      UPDATE users
      SET name = {{ inputs.name }}, email = {{ inputs.email }}
      WHERE id = {{ inputs.userId }}
      RETURNING id, name, email, updated_at
    inputs:
      userId:
        type: int
        description: "User ID to update"
      name:
        type: string
        optional: true
        description: "New name"
      email:
        type: string
        optional: true
        description: "New email"
    data:
      id:
        type: int
      name:
        type: string
      email:
        type: string
      updated_at:
        type: datetime
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
    data:
      date:
        type: datetime
      total_events:
        type: int
      unique_users:
        type: int
```

## Development

### Prerequisites

- Go 1.25.1 or later
- `protoc` (Protocol Buffers compiler) - [Installation guide](https://grpc.io/docs/protoc-installation/)

### Installing protoc

**macOS:**

```bash
brew install protobuf
```

**Linux (Debian/Ubuntu):**

```bash
sudo apt-get install protobuf-compiler
```

**Linux (RHEL/CentOS):**

```bash
sudo yum install protobuf-compiler
```

**Verify installation:**

```bash
protoc --version
```

### Generating Protobuf Files

We use `protoc` with `protoc-gen-go` to generate Go code from protobuf definitions.

**Generate protobuf files:**

```bash
make generate
```

**Note:** The `make setup` command will automatically install `protoc-gen-go` if it's not already installed.

**Optional: Linting and Formatting**

If you have [Buf CLI](https://buf.build/docs/installation) installed, you can use it for optional linting and formatting:

```bash
# Lint proto files (optional)
make lint

# Format proto files (optional)
make format
```

**Automatic Regeneration with Air:**

If you're using `air` for hot reloading (see [Development](#development)), proto files are automatically regenerated when you modify them. The `.air.toml` configuration:

- Watches `.proto` files and regenerates them before building
- Watches `myconfig.yaml` and restarts the server when it changes
- Excludes generated `pkg/pb` files from watching to prevent infinite loops

**Note:** The file move step is required because `paths=source_relative` generates files based on the proto file directory structure, but our Go code imports expect files directly in `pkg/pb/`. This is automated in the `generate-proto.sh` script and Makefile.

### Code Generation

The protobuf code generation uses:

- `protoc` - Protocol Buffers compiler
- `protoc-gen-go` - Go plugin for protoc (generates Go code from `.proto` files)

All generated code is placed in `pkg/pb/` directory.

### Proto File Structure

- `proto/hyperterse/hyperterse.proto` - Core domain models (Model, Adapter, Query, etc.)
- `proto/hyperterse/runtime/runtime.proto` - Runtime service definitions (QueryService, MCPService)

### Validation

We use a custom validator in `pkg/validator/` that performs comprehensive validation:

- Field-level validation (required fields, string patterns, etc.)
- Business logic validation (uniqueness, cross-references, etc.)

The validator ensures:

- Adapter names are unique and follow naming conventions
- Query names are unique and follow naming conventions
- Query `use` references valid adapters
- Statement input references match defined inputs
- All required fields are present
- Types are valid primitive types

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

[Add your license here]

## Support

- **Documentation**: [Full documentation](https://github.com/hyperterse/hyperterse/wiki)
- **Issues**: [GitHub Issues](https://github.com/hyperterse/hyperterse/issues)
- **Discussions**: [GitHub Discussions](https://github.com/hyperterse/hyperterse/discussions)
