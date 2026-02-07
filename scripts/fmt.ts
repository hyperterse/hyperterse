#!/usr/bin/env bun
/**
 * Format script - Run cargo fmt for code formatting
 */

import { $ } from "bun";
import { parseArgs } from "util";

async function main(): Promise<void> {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      check: { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Format Script

Usage:
  bun run fmt          Format all Rust code
  bun run fmt --check  Check formatting without modifying files
`);
    return;
  }

  if (values.check) {
    console.log("🔍 Checking code formatting...\n");
    await $`cargo fmt --all -- --check`;
    console.log("\n✅ Formatting check passed");
  } else {
    console.log("🔨 Formatting code...\n");
    await $`cargo fmt --all`;
    console.log("✅ Code formatted");
  }
}

main().catch((error) => {
  console.error("❌ Format failed:", error);
  process.exit(1);
});
