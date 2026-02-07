import { $ } from "bun";
import { resolve, dirname } from "node:path";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
const projectRoot = resolve(scriptDir, "..");
$.cwd(projectRoot);

// Check if protoc is available
if (!Bun.which("protoc")) {
  console.error(
    "❌ Error: protoc not found. Run 'make setup' or 'bun run scripts/setup.ts' first"
  );
  process.exit(1);
}

// Clean and generate protobuf code
console.log("🔨 Generating protobuf files...");
await $`rm -rf core/proto core/types`;
await $`mkdir -p core/proto`;

const goPath = (await $`go env GOPATH`.quiet().text()).trim();
$.env({
  ...process.env,
  PATH: `${goPath}/bin:${process.env.PATH}`,
});

await $`protoc -I. --go_out=core --go_opt=paths=source_relative proto/connectors/connectors.proto proto/primitives/primitives.proto proto/hyperterse/hyperterse.proto proto/runtime/runtime.proto`;

// Generate types
console.log("🔨 Generating types...");
await $`mkdir -p core/types`;
await $`bun run ${resolve(scriptDir, "generate-types.ts")} proto/connectors/connectors.proto proto/primitives/primitives.proto`;

// Generate JSON schema for .terse files
console.log("🔨 Generating JSON schema...");
await $`bun run ${resolve(scriptDir, "generate-schema.ts")} proto/connectors/connectors.proto proto/primitives/primitives.proto`;

console.log("✓ Protobuf generation complete");
