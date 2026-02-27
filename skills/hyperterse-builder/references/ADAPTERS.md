# Hyperterse Adapters Reference

Adapters connect Hyperterse to databases. Each adapter is a `.terse` file in `app/adapters/`.

The adapter name is derived from the filename (e.g., `main-db.terse` → `main-db`). You can optionally override with the `name` field.

## Supported Connectors

| Connector | Database | Statement Format |
|-----------|----------|------------------|
| `postgres` | PostgreSQL | SQL |
| `mysql` | MySQL/MariaDB | SQL |
| `mongodb` | MongoDB | JSON commands |
| `redis` | Redis | Commands |

---

## PostgreSQL


### Basic Configuration
```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
```

### Full Configuration

```yaml
connector: postgres
connection_string: postgresql://user:password@host:5432/database
options:
  sslmode: disable          # disable, require, verify-ca, verify-full
  connect_timeout: 10
  application_name: "hyperterse"
```

### SSL Modes

| Mode | Description |
|------|-------------|
| `disable` | No SSL (development only) |
| `require` | SSL required, no verification |
| `verify-ca` | SSL + verify CA certificate |
| `verify-full` | SSL + verify CA + hostname |

### Connection String Format
```
postgresql://[user[:password]@][host][:port][/database][?param=value]
```

### Examples

**Local development (`dev-db.terse`):**
```yaml
connector: postgres
connection_string: postgresql://dev:dev@localhost:5432/myapp_dev
options:
  sslmode: disable
```

**Production (`prod-db.terse`):**
```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
options:
  sslmode: require
  max_connections: "25"
```

**Supabase (`supabase.terse`):**
```yaml
connector: postgres
connection_string: "{{ env.SUPABASE_DB_URL }}"
options:
  sslmode: require
```

---

## MySQL

### Basic Configuration
```yaml
connector: mysql
connection_string: "{{ env.MYSQL_DSN }}"
```

### Full Configuration
```yaml
connector: mysql
connection_string: user:password@tcp(host:3306)/database
options:
  charset: utf8mb4
  parseTime: "true"
  loc: UTC
  timeout: 10s
  readTimeout: 30s
  writeTimeout: 30s
```

### Connection String Format
```
[user[:password]@][protocol[(address)]]/database[?param=value]
```

### Examples

**Local development (`dev-mysql.terse`):**
```yaml
connector: mysql
connection_string: root:password@tcp(localhost:3306)/myapp_dev
options:
  charset: utf8mb4
  parseTime: "true"
```

**PlanetScale (`planetscale.terse`):**
```yaml
connector: mysql
connection_string: "{{ env.PLANETSCALE_DSN }}"
options:
  tls: "true"
  charset: utf8mb4
```

---

## MongoDB

### Basic Configuration
```yaml
connector: mongodb
connection_string: "{{ env.MONGODB_URI }}"
```

### Full Configuration
```yaml
connector: mongodb
connection_string: mongodb://user:password@host:27017/database
options:
  authSource: admin
  replicaSet: rs0
  maxPoolSize: "100"
  minPoolSize: "10"
```

### Connection String Format
```
mongodb://[user:password@]host[:port][/database][?options]
mongodb+srv://[user:password@]host[/database][?options]
```

### Examples

**Local development (`dev-mongo.terse`):**
```yaml
connector: mongodb
connection_string: mongodb://localhost:27017/myapp_dev
```

**MongoDB Atlas (`atlas.terse`):**
```yaml
connector: mongodb
connection_string: "{{ env.MONGODB_ATLAS_URI }}"
options:
  retryWrites: "true"
  w: majority
```

### MongoDB Statements

MongoDB uses JSON command syntax:

```yaml
description: "Find user by email"
use: mongo-db
statement: |
  { "find": "users", "filter": { "email": "{{ inputs.email }}" } }
```

```yaml
description: "Insert document"
use: mongo-db
statement: |
  { "insert": "users", "documents": [{ "name": "{{ inputs.name }}", "email": "{{ inputs.email }}" }] }
```

---

## Redis

### Basic Configuration
```yaml
connector: redis
connection_string: "{{ env.REDIS_URL }}"
```

### Full Configuration
```yaml
connector: redis
connection_string: redis://user:password@host:6379/0
options:
  max_retries: "3"
  pool_size: "10"
  min_idle_conns: "5"
  dial_timeout: "5s"
  read_timeout: "3s"
  write_timeout: "3s"
```

### Connection String Format
```
redis://[user:password@]host[:port][/database]
```

### Examples

**Local development (`local-cache.terse`):**
```yaml
connector: redis
connection_string: redis://localhost:6379/0
```

**Production (`prod-cache.terse`):**
```yaml
connector: redis
connection_string: "{{ env.REDIS_URL }}"
options:
  pool_size: "50"
```

### Redis Statements

Redis uses command syntax:

```yaml
description: "Get cached value"
use: cache
statement: GET user:{{ inputs.userId }}
```

```yaml
description: "Set cached value"
use: cache
statement: SET session:{{ inputs.sessionId }} '{{ inputs.data }}'
```

---

## Environment Variables

Always use `{{ env.VAR_NAME }}` for sensitive values:

```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
```

Set environment variables before running:
```bash
export DATABASE_URL="postgresql://user:pass@localhost:5432/mydb"
hyperterse start
```

**Security note:** Never commit plaintext credentials. Always use environment variables or a secrets manager.

---

## Multiple Adapters

Projects can use multiple adapters for different purposes:

```
app/adapters/
├── main-db.terse       # Primary PostgreSQL
├── read-replica.terse  # PostgreSQL read replica
├── cache.terse         # Redis cache
└── documents.terse     # MongoDB for documents
```

Tools reference adapters by name:

```yaml
# Primary database
use: main-db
statement: SELECT * FROM users

# Cache lookup
use: cache
statement: GET session:{{ inputs.sessionId }}
```

---

## Adapter Initialization

- Adapters initialize concurrently at startup
- Connection failures cause immediate process exit
- Health checks run automatically
- Connection pools are managed per-adapter
