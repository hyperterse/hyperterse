# Hyperterse Syntax Reference

Complete reference for all `.terse` file formats and the root `.hyperterse` configuration.

## Root Configuration (.hyperterse)

The root configuration file defines project metadata and directory structure.

```yaml
# Required fields
name: my-service          # Project name (alphanumeric, hyphens)
version: 1.0.0            # Semantic version

# Directory configuration (optional, defaults shown)
root: app                 # Root directory for app files

tools:
  directory: tools        # Tools directory relative to root
  cache:                  # Global caching defaults (optional)
    enabled: true
    ttl: 60               # Cache TTL in seconds
  search:                 # Global search defaults (optional)
    limit: 100            # Default search result limit

adapters:
  directory: adapters     # Adapters directory relative to root

# Server configuration (optional)
server:
  port: 8080              # HTTP port
  log_level: 3            # 0=silent, 1=error, 2=warn, 3=info, 4=debug

# Build configuration (optional)
build:
  out: dist               # Output directory for builds
  clean_dir: true         # Clean output dir before build
```

### Minimal Configuration
```yaml
name: my-api
version: 1.0.0
```

Uses defaults: `app/tools/`, `app/adapters/`, port 8080, no caching.

---

## Adapter Configuration (*.terse)

Adapters define database connections. Place in `app/adapters/`. The adapter name is derived from the filename (e.g., `main-db.terse` → `main-db`).

### Structure
```yaml
connector: postgres             # Database type (required)
connection_string: "{{ env.DATABASE_URL }}"  # (required)
options:                        # (optional)
  key: value
```

### PostgreSQL (`main-db.terse`)
```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
options:
  sslmode: disable
  max_connections: "10"
```

### MySQL (`mysql-db.terse`)
```yaml
connector: mysql
connection_string: "{{ env.MYSQL_DSN }}"
options:
  charset: utf8mb4
  parseTime: "true"
```

### MongoDB (`mongo-db.terse`)
```yaml
connector: mongodb
connection_string: "{{ env.MONGODB_URI }}"
options:
  authSource: admin
```

### Redis (`cache.terse`)
```yaml
connector: redis
connection_string: "{{ env.REDIS_URL }}"
options:
  max_retries: "3"
```

---

## Tool Configuration (config.terse)

Tools define MCP-callable operations. Place in `app/tools/<tool-name>/config.terse`.

### DB-Backed Tool (Full Schema)

```yaml
# Required
description: "Human-readable description for AI agents"

# Database execution
use: adapter-name         # Reference to adapter name
statement: |              # SQL/query to execute
  SELECT * FROM table
  WHERE column = {{ inputs.paramName }}

# Input definitions
inputs:
  paramName:
    type: string          # string, int, float, boolean, datetime
    description: "Description for AI"
    optional: false       # default: false (required)
    default: "value"      # default value if optional

# Script hooks (optional)
mappers:
  input: "./input-validator.ts"
  output: "./output-mapper.ts"

# Authentication
auth:
  plugin: allow_all
  policy:
    key: value
```

### Handler-Only Tool

For tools that don't use a database:

```yaml
description: "Custom tool with TypeScript handler"
handler: "./handler.ts"

inputs:
  query:
    type: string
    description: "Search query"

auth:
  plugin: allow_all
```

### Minimal Examples

**Simplest DB tool:**
```yaml
description: "Get all users"
use: main-db
statement: SELECT * FROM users
```

**Simplest handler tool:**
```yaml
description: "Health check"
handler: "./health.ts"
```

---

## Input Types

| Type | Description | Example Values |
|------|-------------|----------------|
| `string` | Text/varchar | `"hello"`, `"user@email.com"` |
| `int` | Integer | `1`, `42`, `-10` |
| `float` | Decimal number | `3.14`, `0.5`, `-2.7` |
| `boolean` | True/false | `true`, `false` |
| `datetime` | ISO 8601 timestamp | `"2024-01-15T10:30:00Z"` |

### Input Definition Examples

```yaml
inputs:
  # Required string
  name:
    type: string
    description: "User's full name"

  # Required integer
  userId:
    type: int
    description: "Unique user identifier"

  # Optional with default
  limit:
    type: int
    description: "Maximum results to return"
    optional: true
    default: 10

  # Optional without default
  email:
    type: string
    description: "Email filter"
    optional: true

  # Boolean flag
  includeDeleted:
    type: boolean
    description: "Include soft-deleted records"
    optional: true
    default: false

  # Datetime
  since:
    type: datetime
    description: "Filter records after this timestamp"
```

---

## Templating

### Input Substitution
Use `{{ inputs.fieldName }}` for parameter values:

```yaml
statement: |
  SELECT id, name, email
  FROM users
  WHERE id = {{ inputs.userId }}
```

### Environment Variables
Use `{{ env.VAR_NAME }}` for secrets and configuration:

```yaml
connection_string: "{{ env.DATABASE_URL }}"
```

```yaml
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

### Multiple Parameters

```yaml
statement: |
  SELECT *
  FROM orders
  WHERE customer_id = {{ inputs.customerId }}
    AND status = '{{ inputs.status }}'
    AND created_at >= '{{ inputs.since }}'
  ORDER BY created_at DESC
  LIMIT {{ inputs.limit }}
```

---

## Named Exports

Target non-default TypeScript exports using `#` syntax:

```yaml
handler: "./script.ts#weatherHandler"
mappers:
  input: "./script.ts#validateInputs"
  output: "./script.ts#formatResults"
```

---

## File Naming Conventions

| File | Location | Purpose |
|------|----------|---------|
| `.hyperterse` | Project root | Root configuration |
| `*.terse` | `app/adapters/` | Database adapters |
| `config.terse` | `app/tools/<name>/` | Tool configuration |
| `*.ts` | `app/tools/<name>/` | TypeScript handlers/mappers |

### Tool Naming

Tool names are derived from directory paths:
- `app/tools/get-user/` → Tool name: `get-user`
- `app/tools/users/list/` → Tool name: `users-list`
- `app/tools/api/v2/search/` → Tool name: `api-v2-search`

Names can also be explicitly set via the `name` field in config.terse.

---

## Complete Example

```
my-api/
├── .hyperterse
├── app/
│   ├── adapters/
│   │   └── main-db.terse
│   └── tools/
│       ├── get-user/
│       │   ├── config.terse
│       │   └── output-mapper.ts
│       └── create-user/
│           ├── config.terse
│           └── input-validator.ts
```

**.hyperterse:**
```yaml
name: my-api
version: 1.0.0

server:
  port: 3000
  log_level: 3
```

**app/adapters/main-db.terse:**
```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
options:
  sslmode: disable
```

**app/tools/get-user/config.terse:**
```yaml
description: "Fetch a user by their unique ID"
use: main-db
statement: |
  SELECT id, name, email, created_at
  FROM users
  WHERE id = {{ inputs.userId }}
inputs:
  userId:
    type: int
    description: "The user's unique identifier"
mappers:
  output: "./output-mapper.ts"
auth:
  plugin: allow_all
```

**app/tools/get-user/output-mapper.ts:**
```typescript
export default function outputTransform(payload: { results?: any[]; tool?: string }) {
  const rows = payload?.results ?? [];
  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    email: row.email,
    createdAt: row.created_at,
  }));
}
```
