#!/usr/bin/env bun
/**
 * Lint script - Run cargo clippy for linting
 */

import { $ } from "bun";
import { parseArgs } from "util";

async function main(): Promise<void> {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      fix: { type: "boolean", default: false },
      strict: { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Lint Script

Usage:
  bun run lint          Run clippy linter
  bun run lint --fix    Run clippy and automatically fix issues
  bun run lint --strict Run clippy with strict settings (deny warnings)
`);
    return;
  }

  console.log("🔍 Running clippy...\n");

  if (values.fix) {
    await $`cargo clippy --fix --allow-dirty --allow-staged -- -D warnings`;
    console.log("\n✅ Lint issues fixed");
  } else if (values.strict) {
    await $`cargo clippy --all-targets --all-features -- -D warnings -D clippy::all`;
    console.log("\n✅ Strict lint passed");
  } else {
    await $`cargo clippy --all-targets --all-features -- -D warnings`;
    console.log("\n✅ Lint passed");
  }
}

main().catch((error) => {
  console.error("❌ Lint failed:", error);
  process.exit(1);
});
