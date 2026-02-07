#!/usr/bin/env bun
/**
 * Setup script for Hyperterse development environment
 * Checks and installs required dependencies
 */

import { $ } from "bun";

interface Dependency {
  name: string;
  command: string;
  versionFlag: string;
  installInstructions: string[];
}

const REQUIRED_DEPS: Dependency[] = [
  {
    name: "Rust",
    command: "rustc",
    versionFlag: "--version",
    installInstructions: [
      "Install Rust using rustup:",
      "  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh",
      "  or visit: https://www.rust-lang.org/tools/install",
    ],
  },
  {
    name: "Cargo",
    command: "cargo",
    versionFlag: "--version",
    installInstructions: ["Cargo is installed with Rust via rustup"],
  },
  {
    name: "Bun",
    command: "bun",
    versionFlag: "--version",
    installInstructions: [
      "Install Bun:",
      "  curl -fsSL https://bun.sh/install | bash",
      "  or visit: https://bun.sh",
    ],
  },
];

const OPTIONAL_DEPS: Dependency[] = [
  {
    name: "Protocol Buffers Compiler",
    command: "protoc",
    versionFlag: "--version",
    installInstructions: [
      "macOS: brew install protobuf",
      "Ubuntu: sudo apt install protobuf-compiler",
      "Windows: choco install protoc",
    ],
  },
  {
    name: "Docker",
    command: "docker",
    versionFlag: "--version",
    installInstructions: [
      "Install Docker Desktop:",
      "  https://www.docker.com/products/docker-desktop",
    ],
  },
];

async function checkCommand(dep: Dependency): Promise<{ found: boolean; version?: string }> {
  try {
    const result = await $`${dep.command} ${dep.versionFlag}`.text();
    return { found: true, version: result.trim().split("\n")[0] };
  } catch {
    return { found: false };
  }
}

async function checkDependencies(): Promise<{ required: boolean[]; optional: boolean[] }> {
  console.log("🔍 Checking dependencies...\n");

  const requiredResults: boolean[] = [];
  const optionalResults: boolean[] = [];

  // Check required dependencies
  console.log("Required:");
  for (const dep of REQUIRED_DEPS) {
    const result = await checkCommand(dep);
    requiredResults.push(result.found);
    
    if (result.found) {
      console.log(`  ✅ ${dep.name}: ${result.version}`);
    } else {
      console.log(`  ❌ ${dep.name}: Not found`);
      for (const instruction of dep.installInstructions) {
        console.log(`     ${instruction}`);
      }
    }
  }

  // Check optional dependencies
  console.log("\nOptional:");
  for (const dep of OPTIONAL_DEPS) {
    const result = await checkCommand(dep);
    optionalResults.push(result.found);
    
    if (result.found) {
      console.log(`  ✅ ${dep.name}: ${result.version}`);
    } else {
      console.log(`  ⚠️  ${dep.name}: Not found (optional)`);
    }
  }

  return { required: requiredResults, optional: optionalResults };
}

async function installRustTargets(): Promise<void> {
  console.log("\n📦 Installing Rust cross-compilation targets...");
  
  const targets = [
    "x86_64-unknown-linux-gnu",
    "aarch64-unknown-linux-gnu",
    "x86_64-apple-darwin",
    "aarch64-apple-darwin",
    "x86_64-pc-windows-msvc",
  ];

  for (const target of targets) {
    try {
      await $`rustup target add ${target}`.quiet();
      console.log(`  ✅ ${target}`);
    } catch {
      console.log(`  ⚠️  ${target} (may require additional setup)`);
    }
  }
}

async function installRustComponents(): Promise<void> {
  console.log("\n📦 Installing Rust components...");
  
  const components = ["clippy", "rustfmt"];
  
  for (const component of components) {
    try {
      await $`rustup component add ${component}`.quiet();
      console.log(`  ✅ ${component}`);
    } catch {
      console.log(`  ⚠️  ${component} (already installed or failed)`);
    }
  }
}

async function downloadCargoDeps(): Promise<void> {
  console.log("\n📥 Downloading Cargo dependencies...");
  await $`cargo fetch`;
  console.log("  ✅ Dependencies downloaded");
}

async function main(): Promise<void> {
  console.log("🚀 Setting up Hyperterse development environment\n");
  console.log("═".repeat(50));

  const { required } = await checkDependencies();

  // Check if all required dependencies are met
  const allRequiredMet = required.every((r) => r);

  if (!allRequiredMet) {
    console.log("\n❌ Some required dependencies are missing.");
    console.log("   Please install them and run 'bun run setup' again.");
    process.exit(1);
  }

  // Install Rust components
  await installRustComponents();

  // Optionally install cross-compilation targets
  const args = Bun.argv.slice(2);
  if (args.includes("--with-targets")) {
    await installRustTargets();
  }

  // Download dependencies
  await downloadCargoDeps();

  console.log("\n" + "═".repeat(50));
  console.log("✅ Setup complete!\n");
  console.log("Next steps:");
  console.log("  bun run build        Build the project (debug)");
  console.log("  bun run build:release Build release version");
  console.log("  bun run test         Run tests");
  console.log("  bun run help         Show all available commands");
  
  if (!args.includes("--with-targets")) {
    console.log("\nFor cross-compilation support:");
    console.log("  bun run setup --with-targets");
  }
}

main().catch((error) => {
  console.error("❌ Setup failed:", error);
  process.exit(1);
});
