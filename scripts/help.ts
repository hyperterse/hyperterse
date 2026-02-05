#!/usr/bin/env bun
/**
 * Help script - Display all available commands
 */

const commands = [
  { name: "build", description: "Build debug version" },
  { name: "build:release", description: "Build release version" },
  { name: "build:all", description: "Build for all targets (cross-compile)" },
  { name: "test", description: "Run all tests" },
  { name: "test:unit", description: "Run unit tests only" },
  { name: "test:ignored", description: "Run ignored tests (integration)" },
  { name: "setup", description: "Set up development environment" },
  { name: "check", description: "Run cargo check (fast compilation check)" },
  { name: "lint", description: "Run clippy linter" },
  { name: "lint:fix", description: "Run clippy and auto-fix issues" },
  { name: "fmt", description: "Format all Rust code" },
  { name: "fmt:check", description: "Check formatting without changes" },
  { name: "clean", description: "Clean build artifacts" },
  { name: "version:bump", description: "Bump version (specify --major/--minor/--patch)" },
  { name: "version:major", description: "Bump major version" },
  { name: "version:minor", description: "Bump minor version" },
  { name: "version:patch", description: "Bump patch version" },
  { name: "release", description: "Show release info" },
  { name: "release:build", description: "Build release for current platform" },
  { name: "archive", description: "Flatten binaries from artifact subdirectories" },
  { name: "help", description: "Show this help message" },
];

console.log(`
╔═══════════════════════════════════════════════════════════════════╗
║                    Hyperterse Build System                        ║
║                    (Powered by Bun Shell)                         ║
╚═══════════════════════════════════════════════════════════════════╝

Available commands:
`);

// Find the longest command name for alignment
const maxLen = Math.max(...commands.map((c) => c.name.length));

for (const { name, description } of commands) {
  const padding = " ".repeat(maxLen - name.length);
  console.log(`  bun run ${name}${padding}  ${description}`);
}

console.log(`
Quick Start:
  1. bun run setup        Set up development environment
  2. bun run build        Build the project
  3. bun run test         Run tests

For help on a specific command, run:
  bun run <command> --help

Examples:
  bun run build --help
  bun run version:bump --help
`);
