import { $ } from "bun";
import { resolve, dirname } from "node:path";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
$.cwd(resolve(scriptDir, ".."));

console.log("🚀 Setting up Hyperterse...");

// Check and install dependencies
await $`bun run ${resolve(scriptDir, "check-deps.ts")}`;

// Generate protobuf files
await $`bun run ${resolve(scriptDir, "generate-proto.ts")}`;

console.log("");
console.log("✅ Setup complete!");
console.log("");
console.log("Next steps:");
console.log("  1. Build the project:  make build");
console.log("  2. Run the server:     ./hyperterse start -file .hyperterse");
console.log("");
console.log("Available Make commands:");
console.log("  make build   - Build the project");
console.log("  make generate - Regenerate protobuf files");
console.log("  make lint    - Lint proto files");
console.log("  make format  - Format proto files");
