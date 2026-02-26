/**
 * Hyperterse TypeScript Handler Templates
 *
 * Copy and modify these for your tools.
 *
 * Note: All payload fields are optional for defensive coding.
 * Always use nullish coalescing (??) or optional chaining (?.)
 */

// ============================================
// OUTPUT MAPPER
// Transform database results
// File naming: output.ts, *-mapper.ts, or *-data-mapper.ts
// ============================================
export function outputMapper(payload: { results?: any[]; tool?: string }) {
  const rows = payload?.results ?? [];
  return rows.map((row) => ({
    id: row.id,
    name: row.name,
    // Rename snake_case to camelCase
    createdAt: row.created_at,
    updatedAt: row.updated_at,
  }));
}

// ============================================
// INPUT VALIDATOR
// Validate inputs before execution
// File naming: input.ts, *-validator.ts, or *-input-validator.ts
// ============================================
export function inputValidator(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};
  const errors: string[] = [];

  // Email validation
  if (inputs.email && !inputs.email.includes("@")) {
    errors.push("Invalid email format");
  }

  // Positive integer validation
  if (inputs.id && inputs.id <= 0) {
    errors.push("ID must be a positive integer");
  }

  // String length validation
  if (inputs.name && inputs.name.length < 2) {
    errors.push("Name must be at least 2 characters");
  }

  if (errors.length > 0) {
    throw new Error(errors.join("; "));
  }

  return {
    ...inputs,
    // Normalize values
    email: inputs.email?.toLowerCase().trim(),
  };
}

// ============================================
// CUSTOM HANDLER
// Full control, no database
// File naming: handler.ts, *-handler.ts
// ============================================
export function customHandler(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};

  // Example: External API call (fetch is synchronous in Hyperterse!)
  // const response = fetch(`https://api.example.com/data?q=${inputs.query}`);
  // const data = response.json();  // No await needed - sync method

  // Example: Computed result
  const result = {
    query: inputs.query,
    timestamp: new Date().toISOString(),
    computed: (inputs.value ?? 0) * 2,
  };

  return [result];
}

// ============================================
// DEFAULT EXPORT
// Use for single-function scripts
// ============================================
export default function handler(payload: {
  inputs?: Record<string, any>;
  tool?: string;
}) {
  const inputs = payload?.inputs ?? {};
  // Your logic here
  return [{ message: "Hello from handler", tool: payload?.tool }];
}
