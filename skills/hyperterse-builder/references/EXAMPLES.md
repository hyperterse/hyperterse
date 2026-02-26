# Hyperterse Examples

Complete project examples for common use cases.

## CRM API

A customer relationship management API with contacts and companies.

### Structure
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
│       │   ├── delete/config.terse
│       │   └── search/config.terse
│       ├── companies/
│       │   ├── get/config.terse
│       │   ├── list/config.terse
│       │   ├── create/config.terse
│       │   └── update/config.terse
│       └── deals/
│           ├── get/config.terse
│           ├── list/config.terse
│           └── create/config.terse
```

### .hyperterse
```yaml
name: crm-api
version: 1.0.0

server:
  port: 8080
  log_level: 3
```

### app/adapters/crm-db.terse
```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
options:
  sslmode: require
```

### app/tools/contacts/get/config.terse
```yaml
description: "Get a contact by ID with company information"
use: crm-db
statement: |
  SELECT
    c.id, c.first_name, c.last_name, c.email, c.phone,
    c.created_at, c.updated_at,
    co.id as company_id, co.name as company_name
  FROM contacts c
  LEFT JOIN companies co ON c.company_id = co.id
  WHERE c.id = {{ inputs.id }}
inputs:
  id:
    type: int
    description: "Contact ID"
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

### app/tools/contacts/list/config.terse
```yaml
description: "List contacts with pagination and optional company filter"
use: crm-db
statement: |
  SELECT
    c.id, c.first_name, c.last_name, c.email, c.phone,
    co.name as company_name
  FROM contacts c
  LEFT JOIN companies co ON c.company_id = co.id
  ORDER BY c.created_at DESC
  LIMIT {{ inputs.limit }}
  OFFSET {{ inputs.offset }}
inputs:
  limit:
    type: int
    optional: true
    default: 20
  offset:
    type: int
    optional: true
    default: 0
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

### app/tools/contacts/create/config.terse
```yaml
description: "Create a new contact"
use: crm-db
statement: |
  INSERT INTO contacts (first_name, last_name, email, phone, company_id)
  VALUES (
    '{{ inputs.firstName }}',
    '{{ inputs.lastName }}',
    '{{ inputs.email }}',
    '{{ inputs.phone }}',
    {{ inputs.companyId }}
  )
  RETURNING id, first_name, last_name, email, phone, created_at
inputs:
  firstName:
    type: string
    description: "Contact first name"
  lastName:
    type: string
    description: "Contact last name"
  email:
    type: string
    description: "Contact email address"
  phone:
    type: string
    description: "Contact phone number"
    optional: true
  companyId:
    type: int
    description: "Associated company ID"
    optional: true
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

### app/tools/contacts/search/config.terse
```yaml
description: "Search contacts by name or email"
use: crm-db
statement: |
  SELECT id, first_name, last_name, email, phone
  FROM contacts
  WHERE first_name ILIKE '%{{ inputs.query }}%'
     OR last_name ILIKE '%{{ inputs.query }}%'
     OR email ILIKE '%{{ inputs.query }}%'
  ORDER BY last_name, first_name
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
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

---

## Blog API

A blog platform with posts, authors, and comments.

### Structure
```
blog-api/
├── .hyperterse
├── app/
│   ├── adapters/
│   │   └── blog-db.terse
│   └── tools/
│       ├── posts/
│       │   ├── get/config.terse
│       │   ├── list/config.terse
│       │   ├── create/config.terse
│       │   └── publish/config.terse
│       ├── authors/
│       │   ├── get/config.terse
│       │   └── list/config.terse
│       └── comments/
│           ├── list/config.terse
│           └── create/config.terse
```

### app/tools/posts/get/config.terse
```yaml
description: "Get a blog post by slug with author info"
use: blog-db
statement: |
  SELECT
    p.id, p.title, p.slug, p.content, p.excerpt,
    p.published_at, p.created_at, p.updated_at,
    a.id as author_id, a.name as author_name, a.bio as author_bio
  FROM posts p
  JOIN authors a ON p.author_id = a.id
  WHERE p.slug = '{{ inputs.slug }}'
    AND p.published_at IS NOT NULL
inputs:
  slug:
    type: string
    description: "Post URL slug"
auth:
  plugin: allow_all
```

### app/tools/posts/list/config.terse
```yaml
description: "List published blog posts"
use: blog-db
statement: |
  SELECT
    p.id, p.title, p.slug, p.excerpt, p.published_at,
    a.name as author_name
  FROM posts p
  JOIN authors a ON p.author_id = a.id
  WHERE p.published_at IS NOT NULL
  ORDER BY p.published_at DESC
  LIMIT {{ inputs.limit }}
  OFFSET {{ inputs.offset }}
inputs:
  limit:
    type: int
    optional: true
    default: 10
  offset:
    type: int
    optional: true
    default: 0
auth:
  plugin: allow_all
```

### app/tools/comments/create/config.terse
```yaml
description: "Add a comment to a post"
use: blog-db
statement: |
  INSERT INTO comments (post_id, author_name, author_email, content)
  VALUES (
    {{ inputs.postId }},
    '{{ inputs.authorName }}',
    '{{ inputs.authorEmail }}',
    '{{ inputs.content }}'
  )
  RETURNING id, author_name, content, created_at
inputs:
  postId:
    type: int
    description: "Post ID to comment on"
  authorName:
    type: string
    description: "Commenter name"
  authorEmail:
    type: string
    description: "Commenter email"
  content:
    type: string
    description: "Comment text"
auth:
  plugin: allow_all
```

---

## E-Commerce API

An online store with products, orders, and inventory.

### Structure
```
ecommerce-api/
├── .hyperterse
├── app/
│   ├── adapters/
│   │   ├── store-db.terse
│   │   └── cache.terse
│   └── tools/
│       ├── products/
│       │   ├── get/config.terse
│       │   ├── list/config.terse
│       │   ├── search/config.terse
│       │   └── by-category/config.terse
│       ├── orders/
│       │   ├── get/config.terse
│       │   ├── create/config.terse
│       │   └── list-by-customer/config.terse
│       ├── cart/
│       │   ├── get/
│       │   │   ├── config.terse
│       │   │   └── handler.ts
│       │   └── update/
│       │       ├── config.terse
│       │       └── handler.ts
│       └── analytics/
│           └── sales/config.terse
```

### app/adapters/store-db.terse
```yaml
connector: postgres
connection_string: "{{ env.DATABASE_URL }}"
options:
  sslmode: require
  max_connections: "20"
```

### app/adapters/cache.terse
```yaml
connector: redis
connection_string: "{{ env.REDIS_URL }}"
```

### app/tools/products/search/config.terse
```yaml
description: "Search products by name or description"
use: store-db
statement: |
  SELECT
    p.id, p.name, p.description, p.price, p.stock_quantity,
    c.name as category_name,
    COALESCE(
      (SELECT url FROM product_images WHERE product_id = p.id LIMIT 1),
      '/placeholder.jpg'
    ) as image_url
  FROM products p
  JOIN categories c ON p.category_id = c.id
  WHERE p.active = true
    AND (
      p.name ILIKE '%{{ inputs.query }}%'
      OR p.description ILIKE '%{{ inputs.query }}%'
    )
  ORDER BY p.name
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

### app/tools/orders/create/config.terse
```yaml
description: "Create a new order"
use: store-db
statement: |
  INSERT INTO orders (customer_id, status, total, shipping_address)
  VALUES (
    {{ inputs.customerId }},
    'pending',
    {{ inputs.total }},
    '{{ inputs.shippingAddress }}'
  )
  RETURNING id, status, total, created_at
inputs:
  customerId:
    type: int
    description: "Customer ID"
  total:
    type: float
    description: "Order total"
  shippingAddress:
    type: string
    description: "Shipping address"
auth:
  plugin: api_key
  policy:
    value: "{{ env.API_KEY }}"
```

### app/tools/analytics/sales/config.terse
```yaml
description: "Get sales analytics for a date range"
use: store-db
statement: |
  SELECT
    DATE(created_at) as date,
    COUNT(*) as order_count,
    SUM(total) as revenue,
    AVG(total) as avg_order_value
  FROM orders
  WHERE status = 'completed'
    AND created_at >= '{{ inputs.startDate }}'
    AND created_at <= '{{ inputs.endDate }}'
  GROUP BY DATE(created_at)
  ORDER BY date DESC
inputs:
  startDate:
    type: datetime
    description: "Start date"
  endDate:
    type: datetime
    description: "End date"
auth:
  plugin: api_key
  policy:
    value: "{{ env.ADMIN_API_KEY }}"
```

---

## Analytics Dashboard

Real-time analytics with aggregations and time series.

### app/tools/metrics/realtime/config.terse
```yaml
description: "Get real-time metrics for the last hour"
use: analytics-db
statement: |
  SELECT
    DATE_TRUNC('minute', timestamp) as minute,
    COUNT(*) as events,
    COUNT(DISTINCT user_id) as unique_users,
    AVG(duration_ms) as avg_duration
  FROM events
  WHERE timestamp >= NOW() - INTERVAL '1 hour'
  GROUP BY DATE_TRUNC('minute', timestamp)
  ORDER BY minute DESC
auth:
  plugin: api_key
  policy:
    value: "{{ env.ANALYTICS_KEY }}"
```

### app/tools/metrics/top-pages/config.terse
```yaml
description: "Get top pages by views"
use: analytics-db
statement: |
  SELECT
    page_path,
    COUNT(*) as views,
    COUNT(DISTINCT session_id) as unique_sessions,
    AVG(time_on_page) as avg_time_seconds
  FROM page_views
  WHERE timestamp >= '{{ inputs.since }}'
  GROUP BY page_path
  ORDER BY views DESC
  LIMIT {{ inputs.limit }}
inputs:
  since:
    type: datetime
    description: "Start timestamp"
  limit:
    type: int
    optional: true
    default: 10
auth:
  plugin: api_key
  policy:
    value: "{{ env.ANALYTICS_KEY }}"
```

---

## Schema Reference

### CRM Schema
```sql
CREATE TABLE companies (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  website VARCHAR(255),
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE contacts (
  id SERIAL PRIMARY KEY,
  first_name VARCHAR(100) NOT NULL,
  last_name VARCHAR(100) NOT NULL,
  email VARCHAR(255) NOT NULL,
  phone VARCHAR(50),
  company_id INTEGER REFERENCES companies(id),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE deals (
  id SERIAL PRIMARY KEY,
  title VARCHAR(255) NOT NULL,
  value DECIMAL(12,2),
  stage VARCHAR(50) DEFAULT 'lead',
  contact_id INTEGER REFERENCES contacts(id),
  created_at TIMESTAMP DEFAULT NOW()
);
```

### Blog Schema
```sql
CREATE TABLE authors (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  bio TEXT
);

CREATE TABLE posts (
  id SERIAL PRIMARY KEY,
  title VARCHAR(255) NOT NULL,
  slug VARCHAR(255) UNIQUE NOT NULL,
  content TEXT,
  excerpt VARCHAR(500),
  author_id INTEGER REFERENCES authors(id),
  published_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE comments (
  id SERIAL PRIMARY KEY,
  post_id INTEGER REFERENCES posts(id),
  author_name VARCHAR(100) NOT NULL,
  author_email VARCHAR(255) NOT NULL,
  content TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);
```

### E-Commerce Schema
```sql
CREATE TABLE categories (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  slug VARCHAR(100) UNIQUE NOT NULL
);

CREATE TABLE products (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  price DECIMAL(10,2) NOT NULL,
  stock_quantity INTEGER DEFAULT 0,
  category_id INTEGER REFERENCES categories(id),
  active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE customers (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE orders (
  id SERIAL PRIMARY KEY,
  customer_id INTEGER REFERENCES customers(id),
  status VARCHAR(50) DEFAULT 'pending',
  total DECIMAL(12,2) NOT NULL,
  shipping_address TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE order_items (
  id SERIAL PRIMARY KEY,
  order_id INTEGER REFERENCES orders(id),
  product_id INTEGER REFERENCES products(id),
  quantity INTEGER NOT NULL,
  price DECIMAL(10,2) NOT NULL
);
```
