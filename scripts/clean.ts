#!/usr/bin/env bun
/**
 * Clean script - Remove build artifacts
 */

import { $ } from "bun";
import { parseArgs } from "util";

async function main(): Promise<void> {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      all: { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Clean Script

Usage:
  bun run clean        Clean cargo build artifacts
  bun run clean --all  Clean all artifacts including dist folder
`);
    return;
  }

  console.log("🧹 Cleaning build artifacts...\n");
  
  await $`cargo clean`;
  console.log("  ✅ Cargo artifacts cleaned");

  if (values.all) {
    await $`rm -rf dist`;
    console.log("  ✅ dist/ folder removed");
  }

  console.log("\n✅ Clean complete");
}

main().catch((error) => {
  console.error("❌ Clean failed:", error);
  process.exit(1);
});
