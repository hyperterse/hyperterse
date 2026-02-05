#!/usr/bin/env bun
/**
 * Build script for Hyperterse (Rust)
 * Usage:
 *   bun run build            - Build debug version
 *   bun run build --release  - Build release version
 *   bun run build --all      - Build for all targets (cross-compile)
 */

import { $ } from "bun";
import { parseArgs } from "util";

// Cross-compilation targets
const TARGETS = [
  { os: "linux", arch: "x86_64", target: "x86_64-unknown-linux-gnu" },
  { os: "linux", arch: "aarch64", target: "aarch64-unknown-linux-gnu" },
  { os: "darwin", arch: "x86_64", target: "x86_64-apple-darwin" },
  { os: "darwin", arch: "aarch64", target: "aarch64-apple-darwin" },
  { os: "windows", arch: "x86_64", target: "x86_64-pc-windows-msvc" },
];

async function getVersion(): Promise<string> {
  try {
    const result = await $`git describe --tags --always --dirty 2>/dev/null`.text();
    let version = result.trim();
    // Remove 'v' prefix if present
    version = version.replace(/^v/, "");
    // Remove -dirty suffix
    version = version.replace(/-dirty$/, "");
    // Remove commit hash suffix (e.g., 1.0.0-5-gabc1234 -> 1.0.0)
    version = version.replace(/-\d+-g[0-9a-f]+$/, "");
    return version || "dev";
  } catch {
    return "dev";
  }
}

async function buildDebug(): Promise<void> {
  console.log("🔨 Building Hyperterse (debug)...");
  await $`cargo build`;
  console.log("✅ Debug build complete");
}

async function buildRelease(): Promise<void> {
  const version = await getVersion();
  console.log(`🔨 Building Hyperterse (release) v${version}...`);
  
  await $`cargo build --release`;
  
  // Create dist directory
  await $`mkdir -p dist`;
  
  // Copy binary to dist
  const platform = process.platform;
  const binaryName = platform === "win32" ? "hyperterse.exe" : "hyperterse";
  await $`cp target/release/${binaryName} dist/`;
  
  console.log(`✅ Release build complete: dist/${binaryName} (v${version})`);
}

async function buildTarget(target: string, outputName: string): Promise<void> {
  const version = await getVersion();
  console.log(`🔨 Building ${outputName} (v${version})...`);
  
  await $`cargo build --release --target ${target}`;
  
  await $`mkdir -p dist`;
  
  const isWindows = target.includes("windows");
  const sourceBinary = isWindows ? "hyperterse.exe" : "hyperterse";
  const destBinary = isWindows ? `${outputName}.exe` : outputName;
  
  await $`cp target/${target}/release/${sourceBinary} dist/${destBinary}`;
  
  console.log(`✅ Built dist/${destBinary}`);
}

async function buildAll(): Promise<void> {
  const version = await getVersion();
  console.log(`🔨 Building Hyperterse for all targets (v${version})...\n`);
  
  await $`mkdir -p dist`;
  
  for (const { os, arch, target } of TARGETS) {
    const outputName = `hyperterse-${os}-${arch}`;
    try {
      await buildTarget(target, outputName);
    } catch (error) {
      console.warn(`⚠️  Skipping ${target}: ${error instanceof Error ? error.message : "Unknown error"}`);
      console.log(`   (Cross-compilation may require additional toolchains)`);
    }
  }
  
  console.log("\n✅ Cross-compilation complete");
  await $`ls -la dist/`;
}

async function main(): Promise<void> {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      release: { type: "boolean", default: false },
      all: { type: "boolean", default: false },
      target: { type: "string" },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Build Script

Usage:
  bun run build              Build debug version
  bun run build --release    Build release version
  bun run build --all        Build for all targets (cross-compile)
  bun run build --target T   Build for specific target T

Targets:
${TARGETS.map((t) => `  ${t.target}`).join("\n")}
`);
    return;
  }

  if (values.all) {
    await buildAll();
  } else if (values.target) {
    const targetInfo = TARGETS.find((t) => t.target === values.target);
    if (!targetInfo) {
      console.error(`❌ Unknown target: ${values.target}`);
      process.exit(1);
    }
    const outputName = `hyperterse-${targetInfo.os}-${targetInfo.arch}`;
    await buildTarget(values.target, outputName);
  } else if (values.release) {
    await buildRelease();
  } else {
    await buildDebug();
  }
}

main().catch((error) => {
  console.error("❌ Build failed:", error);
  process.exit(1);
});
