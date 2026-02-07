#!/usr/bin/env bun
/**
 * Release script for Hyperterse
 * Builds release binaries for specified or all targets
 */

import { $ } from "bun";
import { parseArgs } from "util";

// Cross-compilation targets
const TARGETS = [
  { os: "linux", arch: "amd64", target: "x86_64-unknown-linux-gnu" },
  { os: "linux", arch: "arm64", target: "aarch64-unknown-linux-gnu" },
  { os: "darwin", arch: "amd64", target: "x86_64-apple-darwin" },
  { os: "darwin", arch: "arm64", target: "aarch64-apple-darwin" },
  { os: "windows", arch: "amd64", target: "x86_64-pc-windows-msvc" },
];

async function getVersion(): Promise<string> {
  try {
    const result = await $`git describe --tags --always 2>/dev/null`.text();
    let version = result.trim().replace(/^v/, "");
    // Clean up version string
    version = version.replace(/-dirty$/, "");
    version = version.replace(/-\d+-g[0-9a-f]+$/, "");
    return version || "dev";
  } catch {
    return "dev";
  }
}

async function buildForTarget(
  target: string,
  os: string,
  arch: string,
  outputDir: string
): Promise<string> {
  const isWindows = target.includes("windows");
  const ext = isWindows ? ".exe" : "";
  const outputName = `hyperterse-${os}-${arch}${ext}`;
  const outputPath = `${outputDir}/${outputName}`;

  console.log(`  🔨 Building for ${os}/${arch}...`);

  try {
    await $`cargo build --release --target ${target}`;
    await $`cp target/${target}/release/hyperterse${ext} ${outputPath}`;
    console.log(`     ✅ ${outputName}`);
    return outputName;
  } catch (error) {
    console.log(`     ❌ Failed: ${error instanceof Error ? error.message : "Unknown error"}`);
    throw error;
  }
}

async function buildRelease(targets: typeof TARGETS, outputDir: string): Promise<string[]> {
  const version = await getVersion();
  console.log(`\n🚀 Building Hyperterse v${version} release\n`);

  await $`mkdir -p ${outputDir}`;

  const built: string[] = [];
  const failed: string[] = [];

  for (const { os, arch, target } of targets) {
    try {
      const name = await buildForTarget(target, os, arch, outputDir);
      built.push(name);
    } catch {
      failed.push(`${os}-${arch}`);
    }
  }

  console.log("\n" + "═".repeat(50));
  console.log(`✅ Successfully built: ${built.length}/${targets.length}`);

  if (failed.length > 0) {
    console.log(`⚠️  Failed: ${failed.join(", ")}`);
    console.log("   (Cross-compilation may require additional toolchains)");
  }

  console.log(`\nOutput directory: ${outputDir}/`);
  await $`ls -la ${outputDir}/`;

  return built;
}

async function main(): Promise<void> {
  const { values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      build: { type: "boolean", default: false },
      target: { type: "string" },
      output: { type: "string", short: "o", default: "dist" },
      all: { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Release Script

Usage:
  bun run release              Show release info
  bun run release --build      Build release for current platform
  bun run release --all        Build for all supported targets
  bun run release --target T   Build for specific target

Options:
  -o, --output DIR   Output directory (default: dist)
  --target TARGET    Specific Rust target triple

Supported targets:
${TARGETS.map((t) => `  ${t.os}-${t.arch} (${t.target})`).join("\n")}

Examples:
  bun run release --build
  bun run release --all
  bun run release --target x86_64-unknown-linux-gnu
`);
    return;
  }

  const outputDir = values.output;

  if (values.all) {
    await buildRelease(TARGETS, outputDir);
  } else if (values.target) {
    const targetInfo = TARGETS.find((t) => t.target === values.target);
    if (!targetInfo) {
      console.error(`❌ Unknown target: ${values.target}`);
      console.log("Supported targets:");
      for (const t of TARGETS) {
        console.log(`  ${t.target}`);
      }
      process.exit(1);
    }
    await buildRelease([targetInfo], outputDir);
  } else if (values.build) {
    // Build for current platform only
    const version = await getVersion();
    console.log(`🚀 Building Hyperterse v${version} release...\n`);

    await $`cargo build --release`;
    await $`mkdir -p ${outputDir}`;

    const platform = process.platform;
    const arch = process.arch === "arm64" ? "arm64" : "amd64";
    const os = platform === "darwin" ? "darwin" : platform === "win32" ? "windows" : "linux";
    const ext = platform === "win32" ? ".exe" : "";
    const outputName = `hyperterse-${os}-${arch}${ext}`;

    await $`cp target/release/hyperterse${ext} ${outputDir}/${outputName}`;

    console.log(`✅ Built: ${outputDir}/${outputName}`);
  } else {
    // Show release info
    const version = await getVersion();
    console.log(`Hyperterse Release v${version}`);
    console.log("\nUse --build to create a release build");
    console.log("Use --all to build for all platforms");
  }
}

main().catch((error) => {
  console.error("❌ Release failed:", error);
  process.exit(1);
});
