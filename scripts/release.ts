import { $ } from "bun";
import { resolve, dirname } from "node:path";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
$.cwd(resolve(scriptDir, ".."));

const [goos, goarch, outputDir = "dist"] = process.argv.slice(2);

if (!goos || !goarch) {
  console.error(`Usage: bun run ${process.argv[1]} <GOOS> <GOARCH> [OUTPUT_DIR]`);
  console.error(`Example: bun run ${process.argv[1]} linux amd64 dist`);
  process.exit(1);
}

// Use build.ts for building
await $`bun run ${resolve(scriptDir, "build.ts")} ${goos} ${goarch} ${outputDir} hyperterse`;
