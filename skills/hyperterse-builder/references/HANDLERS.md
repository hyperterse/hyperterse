# Hyperterse TypeScript Handlers Reference

TypeScript scripts extend tool functionality. They run in a sandboxed runtime with `fetch` and `console` available.

## Script Types

| Type | Purpose | Config Field |
|------|---------|--------------|
| Handler | Full custom logic (no DB) | `handler` |
| Input Mapper | Validate/transform inputs | `mappers.input` |
| Output Mapper | Transform DB results | `mappers.output` |

## Auto-Discovery

Hyperterse auto-discovers scripts by naming convention (no config needed):
- `handler.ts` or `*-handler.ts` → Handler
- `input.ts` or `*-validator.ts` → Input transform
- `output.ts` or `*-mapper.ts` → Output transform

---

## Handler

Handlers replace database execution entirely. Use for custom logic, external APIs, or computed results.

### Signature
```typescript
export default function handler(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}): any | Promise<any>;
```

### Example: External API Call
```typescript
export default async function handler(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};
  const city = inputs.city ?? "London";

  const response = fetch(
    `https://api.weather.com/v1/current?city=${encodeURIComponent(city)}`
  );

  if (!response.ok) {
    throw new Error(`Weather API error: ${response.status}`);
  }

  // Note: Hyperterse fetch returns sync methods, not promises
  const data = response.json();

  return [{
    city,
    temperature: data.temp,
    conditions: data.conditions,
    fetchedAt: new Date().toISOString(),
  }];
}
```

### Example: Computed Result
```typescript
export default function handler(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};
  const { weight = 0, distance = 0, expedited = false } = inputs;

  let cost = weight * 0.5 + distance * 0.1;
  if (expedited) {
    cost *= 1.5;
  }

  return [{
    weight,
    distance,
    expedited,
    shippingCost: Math.round(cost * 100) / 100,
  }];
}
```

### Config
```yaml
description: "Calculate shipping cost"
handler: "./shipping.ts"
inputs:
  weight:
    type: float
    description: "Package weight in kg"
  distance:
    type: float
    description: "Distance in km"
  expedited:
    type: boolean
    optional: true
    default: false
auth:
  plugin: allow_all
```

---

## Input Mapper

Input mappers validate and transform inputs before execution. Throwing an error halts the pipeline.

### Signature
```typescript
export default function inputTransform(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}): Record<string, any> | Promise<Record<string, any>>;
```

### Example: Validation
```typescript
export default function inputTransform(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};

  // Validate email format
  if (!inputs.email?.includes('@')) {
    throw new Error("Invalid email format");
  }

  // Validate positive number
  if (inputs.userId <= 0) {
    throw new Error("userId must be a positive integer");
  }

  return inputs;
}
```

### Example: Transformation
```typescript
export default function inputTransform(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};

  return {
    ...inputs,
    // Normalize email to lowercase
    email: inputs.email?.toLowerCase().trim(),
    // Ensure integer
    userId: Math.floor(Number(inputs.userId)),
    // Add computed field
    searchPattern: `%${inputs.query}%`,
  };
}
```

### Example: Complex Validation
```typescript
export default function inputTransform(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};
  const errors: string[] = [];

  if (!inputs.name || inputs.name.length < 2) {
    errors.push("Name must be at least 2 characters");
  }

  if (!inputs.email?.match(/^[^\s@]+@[^\s@]+\.[^\s@]+$/)) {
    errors.push("Invalid email format");
  }

  if (inputs.age && (inputs.age < 0 || inputs.age > 150)) {
    errors.push("Age must be between 0 and 150");
  }

  if (errors.length > 0) {
    throw new Error(errors.join("; "));
  }

  return inputs;
}
```

### Config
```yaml
description: "Create user"
use: main-db
statement: |
  INSERT INTO users (name, email)
  VALUES ('{{ inputs.name }}', '{{ inputs.email }}')
  RETURNING *
inputs:
  name:
    type: string
    description: "User name"
  email:
    type: string
    description: "User email"
mappers:
  input: "./validate-user.ts"
auth:
  plugin: allow_all
```

---

## Output Mapper

Output mappers transform database results before returning to the client.

### Signature
```typescript
export default function outputTransform(payload: {
  results?: any[];
  tool?: string;
}): any | Promise<any>;
```

### Example: Field Mapping
```typescript
export default function outputTransform(payload: {
  results?: any[];
  tool?: string;
}) {
  const rows = payload?.results ?? [];
  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    email: row.email,
    // Rename snake_case to camelCase
    createdAt: row.created_at,
    updatedAt: row.updated_at,
  }));
}
```

### Example: Data Enrichment
```typescript
export default function outputTransform(payload: {
  results?: any[];
  tool?: string;
}) {
  const rows = payload?.results ?? [];
  return rows.map((row) => ({
    ...row,
    // Add computed fields
    fullName: `${row.first_name} ${row.last_name}`,
    isActive: row.status === 'active',
    // Format dates
    createdAt: new Date(row.created_at).toISOString(),
  }));
}
```

### Example: Aggregation
```typescript
export default function outputTransform(payload: {
  results?: any[];
  tool?: string;
}) {
  const rows = payload?.results ?? [];

  if (rows.length === 0) {
    return { total: 0, items: [] };
  }

  return {
    total: rows.length,
    items: rows,
    summary: {
      minPrice: Math.min(...rows.map(r => r.price)),
      maxPrice: Math.max(...rows.map(r => r.price)),
      avgPrice: rows.reduce((sum, r) => sum + r.price, 0) / rows.length,
    },
  };
}
```

### Config
```yaml
description: "Get user by ID"
use: main-db
statement: |
  SELECT * FROM users WHERE id = {{ inputs.userId }}
inputs:
  userId:
    type: int
    description: "User ID"
mappers:
  output: "./format-user.ts"
auth:
  plugin: allow_all
```

---

## Named Exports

Target specific exports instead of default:

```typescript
// handlers.ts
export function weatherHandler(payload: { inputs: Record<string, any> }) {
  return [{ weather: "sunny" }];
}

export function stockHandler(payload: { inputs: Record<string, any> }) {
  return [{ price: 100.50 }];
}
```

```yaml
# weather tool
handler: "./handlers.ts#weatherHandler"

# stock tool
handler: "./handlers.ts#stockHandler"
```

---

## Available Runtime APIs

### fetch
HTTP requests (note: Hyperterse's fetch is **synchronous**, not standard):
```typescript
// fetch() returns a response object directly (not a Promise)
const response = fetch('https://api.example.com/data', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ key: 'value' }),
});

// .json() and .text() are also synchronous (not Promises)
const data = response.json();  // Returns object directly
const text = response.text();  // Returns string directly

// Response properties
response.status;  // HTTP status code (number)
response.ok;      // true if status 200-299
```

**Important:** Unlike browser/Node.js fetch, Hyperterse's fetch blocks until the request completes. Do not use `await` with `.json()` or `.text()`.

### console
Structured logging:
```typescript
console.log("Info message", { userId: 123 });
console.error("Error occurred", { error: err.message });
console.warn("Warning", { deprecated: true });
```

### Restrictions
- No filesystem access
- No network sockets (use fetch)
- No timers (setTimeout, setInterval)
- No process/environment access
- No async/await needed for fetch (it's synchronous)

---

## Error Handling

Throw errors to halt execution and return error to client:

```typescript
export default function handler(payload: { inputs: Record<string, any> }) {
  const { userId } = payload.inputs;

  if (!userId) {
    throw new Error("userId is required");
  }

  // Continue processing...
}
```

Errors are returned as MCP error responses.

---

## Type Definitions

Full TypeScript types for payloads:

```typescript
interface HandlerPayload {
  inputs?: Record<string, any>;
  tool?: string;  // Tool path (e.g., "users/get")
}

interface InputTransformPayload {
  inputs?: Record<string, any>;
  tool?: string;
}

interface OutputTransformPayload {
  results?: any[];
  tool?: string;
}
```

**Note:** All fields are optional for defensive coding. Always use nullish coalescing (`??`) or optional chaining (`?.`).

---

## Best Practices

1. **Keep handlers focused** - One responsibility per handler
2. **Validate early** - Use input mappers for all validation
3. **Type your inputs** - Use TypeScript interfaces for clarity
4. **Handle errors gracefully** - Throw descriptive error messages
5. **Log appropriately** - Use console for debugging, not production logging
6. **Avoid side effects** - Handlers should be deterministic when possible
