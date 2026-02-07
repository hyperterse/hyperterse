#!/usr/bin/env bun
/**
 * Check script - Run cargo check for fast compilation checks
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
Hyperterse Check Script

Usage:
  bun run check        Run cargo check (fast compilation check)
  bun run check --all  Check all targets and features
`);
    return;
  }

  console.log("🔍 Running cargo check...\n");
  
  if (values.all) {
    await $`cargo check --all-targets --all-features`;
  } else {
    await $`cargo check`;
  }
  
  console.log("\n✅ Check passed");
}

main().catch((error) => {
  console.error("❌ Check failed:", error);
  process.exit(1);
});
