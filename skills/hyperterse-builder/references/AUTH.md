# Hyperterse Authentication Reference

Authentication plugins control access to tools. Auth runs as the first stage of the execution pipeline.

## Built-in Plugins

| Plugin | Use Case |
|--------|----------|
| `allow_all` | Development, public tools, health checks |
| `api_key` | Production API protection |

---

## allow_all

Unconditionally allows every request. Use for development or intentionally public tools.

### Config
```yaml
auth:
  plugin: allow_all
```

### Use Cases
- Health check endpoints
- Public read-only data
- Development/testing
- Internal tools (with network-level security)

---

## api_key

Validates the `X-API-Key` HTTP header against a configured value.

### Config
```yaml
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

### How It Works
1. Client sends request with `X-API-Key: <key>` header
2. Plugin compares against `policy.value`
3. Match = allowed, mismatch = 401 Unauthorized

### Environment Variable
```yaml
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

```bash
export API_KEY="sk-your-secret-key"
hyperterse start
```

### Static Value (Not Recommended)
```yaml
auth:
  plugin: api_key
  policy:
    value: "sk-hardcoded-key"  # Don't do this in production
```

### Default Behavior
If `policy.value` is not set, falls back to `HYPERTERSE_API_KEY` environment variable:
```yaml
auth:
  plugin: api_key
  # Uses HYPERTERSE_API_KEY env var
```

```bash
export HYPERTERSE_API_KEY="sk-your-key"
```

---

## Tool-Level Auth

Each tool can have its own auth configuration:

```yaml
# app/tools/public-data/config.terse
description: "Public data endpoint"
use: main-db
statement: SELECT * FROM public_stats
auth:
  plugin: allow_all

# app/tools/admin-action/config.terse
description: "Admin-only action"
use: main-db
statement: DELETE FROM users WHERE id = {{ inputs.userId }}
inputs:
  userId:
    type: int
auth:
  plugin: api_key
  policy:
    value: "{{ env.ADMIN_API_KEY }}"
```

---

## No Auth (Unauthenticated)

Tools without an `auth` block are unauthenticated with no default protection.

```yaml
# WARNING: This tool has no authentication
description: "Unprotected endpoint"
use: main-db
statement: SELECT * FROM users
```

**Best Practice:** Always explicitly set auth, even if using `allow_all`.

---

## Auth in Execution Pipeline

Auth runs first in the tool execution pipeline:

1. **Auth** ← Validates request
2. Input Mapper
3. Execute (DB or Handler)
4. Output Mapper

If auth fails, the pipeline stops immediately with a 401 error.

---

## Security Best Practices

### Use Environment Variables
```yaml
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

Never commit secrets to version control.

### Different Keys Per Environment
```bash
# Development
export API_KEY="dev-key-12345"

# Production
export API_KEY="prod-sk-xxxxxxxxxxxxx"
```

### Rotate Keys Regularly
Update `API_KEY` environment variable and restart the server.

### Use Network-Level Security Too
Even with `api_key`, consider:
- VPC/private networking
- IP allowlisting
- Rate limiting
- TLS/HTTPS

---

## Example: Multi-Tier Auth

```
app/tools/
├── health/config.terse          # allow_all (public)
├── users/
│   ├── get/config.terse         # api_key (user key)
│   └── delete/config.terse      # api_key (admin key)
└── analytics/
    └── dashboard/config.terse   # api_key (analytics key)
```

**health/config.terse:**
```yaml
description: "Health check"
handler: "./health.ts"
auth:
  plugin: allow_all
```

**users/get/config.terse:**
```yaml
description: "Get user"
use: main-db
statement: SELECT * FROM users WHERE id = {{ inputs.id }}
inputs:
  id:
    type: int
auth:
  plugin: api_key
  policy:
    value: "{{ env.USER_API_KEY }}"
```

**users/delete/config.terse:**
```yaml
description: "Delete user (admin only)"
use: main-db
statement: DELETE FROM users WHERE id = {{ inputs.id }}
inputs:
  id:
    type: int
auth:
  plugin: api_key
  policy:
    value: "{{ env.ADMIN_API_KEY }}"
```

---

## Testing Auth

### Without Auth (allow_all)
```bash
curl http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"health"},"id":1}'
```

### With API Key
```bash
curl http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get-user","arguments":{"id":1}},"id":1}'
```

### Expected Errors
- Missing header: `401 Unauthorized`
- Wrong key: `401 Unauthorized`
- Valid key: Tool executes normally
