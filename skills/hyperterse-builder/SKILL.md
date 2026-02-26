---
name: hyperterse-builder
description: Build complete Hyperterse MCP server projects from natural language. Use when the user asks to create an API, backend, MCP server, database-backed tools, or mentions Hyperterse.
license: Apache-2.0
metadata:
  author: hyperterse
  version: "1.0.0"
  repository: https://github.com/hyperterse/hyperterse
---

# Hyperterse Builder

Generate production-ready Hyperterse MCP server projects from natural language descriptions.

## When to Use This Skill

Activate when the user:
- Asks to "build an API" or "create a backend"
- Mentions "MCP server" or "MCP tools"
- Wants "database-backed tools" or "data access layer"
- Says "Hyperterse" explicitly
- Needs CRUD operations exposed to AI agents
- Wants to connect databases to LLMs/agents

**Trigger phrases:** "build me a", "create an API for", "I need tools for", "set up a backend", "MCP server for", "Hyperterse project"

## Project Structure

Every Hyperterse project follows this structure:

```
project-name/
├── .hyperterse              # Root configuration
├── app/
│   ├── adapters/            # Database connections
│   │   └── *.terse
│   └── tools/               # Tool definitions
│       └── tool-name/
│           ├── config.terse # Tool config
│           └── *.ts         # Optional handlers/mappers
```

## Generation Workflow

When generating a Hyperterse project:

### Step 1: Understand Requirements
Ask clarifying questions if needed:
- What entities/resources? (users, products, orders, etc.)
- What operations? (CRUD, search, aggregations, custom)
- What database? (PostgreSQL default, MySQL, MongoDB, Redis)
- Authentication needs? (allow_all for dev, api_key for prod)

### Step 2: Create Root Config
Create `.hyperterse` with project metadata:

```yaml
name: project-name
version: 1.0.0

root: app

tools:
  directory: tools
  cache:
    enabled: true
    ttl: 60
  search:
    limit: 100

adapters:
  directory: adapters

server:
  port: 8080
  log_level: 3

build:
  out: dist
  clean_dir: true
```

**Note:** `tools.cache`, `tools.search`, and `build` are optional.

### Step 3: Create Adapters
Create `app/adapters/<name>.terse` for each database. The adapter name is derived from the filename (e.g., `main-db.terse` → adapter name `main-db`):

```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
options:
  sslmode: disable
```

### Step 4: Create Tools
For each operation, create `app/tools/<tool-name>/config.terse`:

**DB-backed tool:**
```yaml
description: "Get user by ID"
use: main-db
statement: |
  SELECT id, name, email, created_at
  FROM users
  WHERE id = {{ inputs.userId }}
inputs:
  userId:
    type: int
    description: "User ID to fetch"
auth:
  plugin: allow_all
```

**Custom handler tool:**
```yaml
description: "Calculate shipping cost"
handler: "./shipping-handler.ts"
auth:
  plugin: allow_all
```

### Step 5: Add Mappers (Optional)
For input validation or output transformation:

```yaml
mappers:
  input: "./validate-input.ts"
  output: "./transform-output.ts"
```

### Step 6: Validate and Run
```bash
hyperterse validate
hyperterse start --watch
```

## Quick Syntax Reference

### Input Types
| Type | Description | Example |
|------|-------------|---------|
| `string` | Text values | `"hello"` |
| `int` | Integers | `42` |
| `float` | Decimal numbers | `3.14` |
| `boolean` | True/false | `true` |
| `datetime` | ISO 8601 | `"2024-01-15T10:30:00Z"` |

### Input Options
```yaml
inputs:
  requiredField:
    type: string
    description: "Required field"
  optionalField:
    type: int
    description: "Optional with default"
    optional: true
    default: 10
```

### Connectors
| Connector | Database |
|-----------|----------|
| `postgres` | PostgreSQL |
| `mysql` | MySQL/MariaDB |
| `mongodb` | MongoDB |
| `redis` | Redis |

### Statement Templating
Use `{{ inputs.fieldName }}` for parameter substitution:
```yaml
statement: |
  SELECT * FROM users
  WHERE name LIKE '%{{ inputs.search }}%'
  LIMIT {{ inputs.limit }}
```

### Environment Variables
Use `{{ env.VAR_NAME }}` for secrets:
```yaml
connection_string: "{{ env.DATABASE_URL }}"
```

## Auth Plugins

| Plugin | Use Case |
|--------|----------|
| `allow_all` | Development, public tools |
| `api_key` | Production API protection |

```yaml
# Development
auth:
  plugin: allow_all

# Production
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

## Common Patterns

### CRUD Pattern
For a resource like `users`, generate these tools:

| Tool | Statement |
|------|-----------|
| `get-user` | `SELECT * FROM users WHERE id = {{ inputs.id }}` |
| `list-users` | `SELECT * FROM users LIMIT {{ inputs.limit }} OFFSET {{ inputs.offset }}` |
| `create-user` | `INSERT INTO users (name, email) VALUES ('{{ inputs.name }}', '{{ inputs.email }}') RETURNING *` |
| `update-user` | `UPDATE users SET name = '{{ inputs.name }}' WHERE id = {{ inputs.id }} RETURNING *` |
| `delete-user` | `DELETE FROM users WHERE id = {{ inputs.id }}` |

### Search Pattern
```yaml
description: "Search users by name or email"
use: main-db
statement: |
  SELECT id, name, email
  FROM users
  WHERE name ILIKE '%{{ inputs.query }}%'
     OR email ILIKE '%{{ inputs.query }}%'
  LIMIT {{ inputs.limit }}
inputs:
  query:
    type: string
    description: "Search term"
  limit:
    type: int
    optional: true
    default: 20
```

### Aggregation Pattern
```yaml
description: "Get order statistics by status"
use: main-db
statement: |
  SELECT status, COUNT(*) as count, SUM(total) as revenue
  FROM orders
  WHERE created_at >= '{{ inputs.since }}'
  GROUP BY status
inputs:
  since:
    type: datetime
    description: "Start date for aggregation"
```

## TypeScript Handlers

Hyperterse auto-discovers script files by naming convention:
- `handler.ts` or `*-handler.ts` → Custom handler
- `input.ts` or `*-validator.ts` → Input transform
- `output.ts` or `*-mapper.ts` → Output transform

### Output Mapper
Transform database results:
```typescript
export default function outputTransform(payload: { results?: any[]; tool?: string }) {
  const rows = payload?.results ?? [];
  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    createdAt: row.created_at,
  }));
}
```

### Input Validator
Validate inputs before execution:
```typescript
export default function inputTransform(payload: { inputs?: Record<string, any>; tool?: string }) {
  const inputs = payload?.inputs ?? {};
  if (!inputs.email?.includes('@')) {
    throw new Error("Invalid email format");
  }
  return inputs;
}
```

### Custom Handler
Full control (no database):
```typescript
export default function handler(payload: { inputs?: Record<string, any>; tool?: string }) {
  const inputs = payload?.inputs ?? {};
  return [{ result: "computed data", input: inputs.query }];
}
```

### Named Exports
Target specific exports with `#` syntax:
```yaml
handler: "./handlers.ts#weatherHandler"
mappers:
  input: "./validators.ts#validateEmail"
```

## Reference Documentation

For detailed documentation, see the `references/` directory:
- `SYNTAX.md` - Complete .terse file syntax
- `ADAPTERS.md` - Database adapter configurations
- `TOOLS.md` - Tool configuration patterns
- `HANDLERS.md` - TypeScript handler patterns
- `AUTH.md` - Authentication plugins
- `EXAMPLES.md` - Full project examples

## Validation Checklist

After generating a project, verify:
- [ ] `.hyperterse` exists with valid YAML
- [ ] At least one adapter in `app/adapters/`
- [ ] Tools reference existing adapters (unless using handlers)
- [ ] All `{{ inputs.x }}` have corresponding input definitions
- [ ] TypeScript files have default exports
- [ ] Run `hyperterse validate` passes

## Example: CRM API

User: "Build me a CRM API with contacts and companies"

Generated structure:
```
crm-api/
├── .hyperterse
├── app/
│   ├── adapters/
│   │   └── crm-db.terse
│   └── tools/
│       ├── contacts/
│       │   ├── get/config.terse
│       │   ├── list/config.terse
│       │   ├── create/config.terse
│       │   ├── update/config.terse
│       │   └── delete/config.terse
│       ├── companies/
│       │   ├── get/config.terse
│       │   ├── list/config.terse
│       │   ├── create/config.terse
│       │   ├── update/config.terse
│       │   └── delete/config.terse
│       └── search/
│           └── config.terse
```

## CLI Commands

After generating, users can run:
```bash
hyperterse validate      # Validate configuration
hyperterse start         # Start development server
hyperterse start --watch # Start with hot reload
hyperterse build -o dist # Build for production
hyperterse serve dist/   # Serve production build
```

## Installation

```bash
curl -fsSL https://hyperterse.com/install | bash
```
