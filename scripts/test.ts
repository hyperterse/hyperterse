#!/usr/bin/env bun
/**
 * Test script for Hyperterse (Rust)
 * Usage:
 *   bun run test            - Run all tests
 *   bun run test --unit     - Run unit tests only
 *   bun run test --ignored  - Run ignored tests (e.g., integration tests)
 */

import { $ } from "bun";
import { parseArgs } from "util";

async function runAllTests(): Promise<void> {
  console.log("🧪 Running all tests...\n");
  await $`cargo test`;
  console.log("\n✅ All tests passed");
}

async function runUnitTests(): Promise<void> {
  console.log("🧪 Running unit tests...\n");
  await $`cargo test --lib`;
  console.log("\n✅ Unit tests passed");
}

async function runIgnoredTests(): Promise<void> {
  console.log("🧪 Running ignored tests (requires running services)...\n");
  await $`cargo test -- --ignored`;
  console.log("\n✅ Ignored tests passed");
}

async function runTestsWithCoverage(): Promise<void> {
  console.log("🧪 Running tests with coverage...\n");
  
  // Check if cargo-llvm-cov is installed
  try {
    await $`cargo llvm-cov --version`.quiet();
  } catch {
    console.log("📦 Installing cargo-llvm-cov...");
    await $`cargo install cargo-llvm-cov`;
  }
  
  await $`cargo llvm-cov --html`;
  console.log("\n✅ Coverage report generated in target/llvm-cov/html/");
}

async function main(): Promise<void> {
  const { values, positionals } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      unit: { type: "boolean", default: false },
      ignored: { type: "boolean", default: false },
      coverage: { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Test Script

Usage:
  bun run test              Run all tests
  bun run test --unit       Run unit tests only (--lib)
  bun run test --ignored    Run ignored tests (integration tests)
  bun run test --coverage   Run tests with coverage report

Additional arguments are passed to cargo test:
  bun run test -- --nocapture
  bun run test -- test_name
`);
    return;
  }

  if (values.coverage) {
    await runTestsWithCoverage();
  } else if (values.unit) {
    await runUnitTests();
  } else if (values.ignored) {
    await runIgnoredTests();
  } else if (positionals.length > 0) {
    // Pass additional arguments to cargo test
    console.log("🧪 Running tests with custom arguments...\n");
    await $`cargo test -- ${positionals}`;
    console.log("\n✅ Tests completed");
  } else {
    await runAllTests();
  }
}

main().catch((error) => {
  console.error("❌ Tests failed:", error);
  process.exit(1);
});
