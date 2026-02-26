# Hyperterse Tools Reference

Tools are MCP-callable operations defined in `app/tools/<tool-name>/config.terse`.

## Tool Types

### DB-Backed Tools
Execute queries against database adapters:
```yaml
description: "Get user by ID"
use: main-db
statement: |
  SELECT * FROM users WHERE id = {{ inputs.userId }}
inputs:
  userId:
    type: int
    description: "User ID"
auth:
  plugin: allow_all
```

### Handler Tools
Delegate to TypeScript for custom logic:
```yaml
description: "Calculate shipping cost"
handler: "./shipping.ts"
inputs:
  weight:
    type: float
    description: "Package weight in kg"
auth:
  plugin: allow_all
```

### Hybrid Tools
Combine database execution with TypeScript mappers:
```yaml
description: "Get user with formatted output"
use: main-db
statement: |
  SELECT * FROM users WHERE id = {{ inputs.userId }}
inputs:
  userId:
    type: int
    description: "User ID"
mappers:
  input: "./validate.ts"
  output: "./format.ts"
auth:
  plugin: allow_all
```

---

## Execution Pipeline

Tools execute in this order:
1. **Auth** - Authentication plugin validates request
2. **Input Mapper** - Optional TypeScript validation/transformation
3. **Execute** - Database query or handler execution
4. **Output Mapper** - Optional TypeScript result transformation

---

## CRUD Patterns

### Get Single Record
```yaml
description: "Get user by ID"
use: main-db
statement: |
  SELECT id, name, email, created_at
  FROM users
  WHERE id = {{ inputs.id }}
inputs:
  id:
    type: int
    description: "User ID"
auth:
  plugin: allow_all
```

### List with Pagination
```yaml
description: "List users with pagination"
use: main-db
statement: |
  SELECT id, name, email, created_at
  FROM users
  ORDER BY created_at DESC
  LIMIT {{ inputs.limit }}
  OFFSET {{ inputs.offset }}
inputs:
  limit:
    type: int
    description: "Number of results"
    optional: true
    default: 20
  offset:
    type: int
    description: "Results to skip"
    optional: true
    default: 0
auth:
  plugin: allow_all
```

### Create Record
```yaml
description: "Create new user"
use: main-db
statement: |
  INSERT INTO users (name, email)
  VALUES ('{{ inputs.name }}', '{{ inputs.email }}')
  RETURNING id, name, email, created_at
inputs:
  name:
    type: string
    description: "User name"
  email:
    type: string
    description: "User email"
auth:
  plugin: allow_all
```

### Update Record
```yaml
description: "Update user"
use: main-db
statement: |
  UPDATE users
  SET name = '{{ inputs.name }}', email = '{{ inputs.email }}'
  WHERE id = {{ inputs.id }}
  RETURNING id, name, email, created_at
inputs:
  id:
    type: int
    description: "User ID"
  name:
    type: string
    description: "New name"
  email:
    type: string
    description: "New email"
auth:
  plugin: allow_all
```

### Delete Record
```yaml
description: "Delete user"
use: main-db
statement: |
  DELETE FROM users
  WHERE id = {{ inputs.id }}
  RETURNING id
inputs:
  id:
    type: int
    description: "User ID to delete"
auth:
  plugin: allow_all
```

---

## Search Patterns

### Text Search
```yaml
description: "Search users by name or email"
use: main-db
statement: |
  SELECT id, name, email
  FROM users
  WHERE name ILIKE '%{{ inputs.query }}%'
     OR email ILIKE '%{{ inputs.query }}%'
  ORDER BY name
  LIMIT {{ inputs.limit }}
inputs:
  query:
    type: string
    description: "Search term"
  limit:
    type: int
    optional: true
    default: 20
auth:
  plugin: allow_all
```

### Filter by Field
```yaml
description: "Get users by status"
use: main-db
statement: |
  SELECT id, name, email, status
  FROM users
  WHERE status = '{{ inputs.status }}'
  ORDER BY created_at DESC
  LIMIT {{ inputs.limit }}
inputs:
  status:
    type: string
    description: "User status (active, inactive, pending)"
  limit:
    type: int
    optional: true
    default: 50
auth:
  plugin: allow_all
```

### Date Range Filter
```yaml
description: "Get orders in date range"
use: main-db
statement: |
  SELECT id, customer_id, total, created_at
  FROM orders
  WHERE created_at >= '{{ inputs.startDate }}'
    AND created_at <= '{{ inputs.endDate }}'
  ORDER BY created_at DESC
inputs:
  startDate:
    type: datetime
    description: "Start of date range"
  endDate:
    type: datetime
    description: "End of date range"
auth:
  plugin: allow_all
```

---

## Aggregation Patterns

### Count
```yaml
description: "Count users by status"
use: main-db
statement: |
  SELECT status, COUNT(*) as count
  FROM users
  GROUP BY status
auth:
  plugin: allow_all
```

### Sum and Statistics
```yaml
description: "Get order statistics"
use: main-db
statement: |
  SELECT
    COUNT(*) as total_orders,
    SUM(total) as total_revenue,
    AVG(total) as avg_order_value
  FROM orders
  WHERE created_at >= '{{ inputs.since }}'
inputs:
  since:
    type: datetime
    description: "Start date for statistics"
auth:
  plugin: allow_all
```

### Group By
```yaml
description: "Revenue by product category"
use: main-db
statement: |
  SELECT
    p.category,
    COUNT(oi.id) as items_sold,
    SUM(oi.quantity * oi.price) as revenue
  FROM order_items oi
  JOIN products p ON oi.product_id = p.id
  WHERE oi.created_at >= '{{ inputs.since }}'
  GROUP BY p.category
  ORDER BY revenue DESC
inputs:
  since:
    type: datetime
    description: "Start date"
auth:
  plugin: allow_all
```

---

## Join Patterns

### Simple Join
```yaml
description: "Get order with customer info"
use: main-db
statement: |
  SELECT
    o.id as order_id,
    o.total,
    o.created_at,
    c.name as customer_name,
    c.email as customer_email
  FROM orders o
  JOIN customers c ON o.customer_id = c.id
  WHERE o.id = {{ inputs.orderId }}
inputs:
  orderId:
    type: int
    description: "Order ID"
auth:
  plugin: allow_all
```

### Multiple Joins
```yaml
description: "Get order details with items and products"
use: main-db
statement: |
  SELECT
    o.id as order_id,
    o.total,
    oi.quantity,
    p.name as product_name,
    p.price
  FROM orders o
  JOIN order_items oi ON o.id = oi.order_id
  JOIN products p ON oi.product_id = p.id
  WHERE o.id = {{ inputs.orderId }}
inputs:
  orderId:
    type: int
    description: "Order ID"
auth:
  plugin: allow_all
```

---

## Tool Directory Structure

Tools are organized in directories:

```
app/tools/
├── users/
│   ├── get/config.terse        # users-get
│   ├── list/config.terse       # users-list
│   ├── create/config.terse     # users-create
│   ├── update/config.terse     # users-update
│   └── delete/config.terse     # users-delete
├── orders/
│   ├── get/config.terse        # orders-get
│   └── list/config.terse       # orders-list
└── analytics/
    └── dashboard/config.terse  # analytics-dashboard
```

Tool names are derived from paths:
- `users/get` → `users-get`
- `analytics/dashboard` → `analytics-dashboard`

---

## Best Practices

1. **Descriptive descriptions** - AI agents rely on these to choose tools
2. **Validate inputs** - Use input mappers for complex validation
3. **Limit results** - Always include pagination for list operations
4. **Use RETURNING** - Return affected rows for INSERT/UPDATE/DELETE
5. **Consistent naming** - Use `resource-action` pattern (e.g., `user-get`, `order-create`)
