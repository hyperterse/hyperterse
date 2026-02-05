#!/usr/bin/env bun
/**
 * Archive script - Flatten binaries from artifact subdirectories
 * Used after GitHub Actions download-artifact step
 */

import { $ } from "bun";
import { readdir, rename, rm, stat } from "fs/promises";
import { join } from "path";
import { parseArgs } from "util";

async function isDirectory(path: string): Promise<boolean> {
  try {
    const s = await stat(path);
    return s.isDirectory();
  } catch {
    return false;
  }
}

async function flattenBinaries(distDir: string): Promise<void> {
  console.log(`🔨 Flattening binaries in ${distDir}/...\n`);

  const entries = await readdir(distDir);
  let flattened = 0;

  for (const entry of entries) {
    // Only process directories starting with hyperterse-
    if (!entry.startsWith("hyperterse-")) continue;

    const entryPath = join(distDir, entry);
    if (!(await isDirectory(entryPath))) continue;

    // Find binary inside the directory
    const subEntries = await readdir(entryPath);
    const binary = subEntries.find((f) => f.startsWith("hyperterse-"));

    if (!binary) {
      console.log(`  ⚠️  No binary found in ${entry}/`);
      continue;
    }

    const sourcePath = join(entryPath, binary);
    const tempPath = join(distDir, `${binary}.tmp`);
    const destPath = join(distDir, binary);

    // Move binary to temp location first to avoid conflict
    await rename(sourcePath, tempPath);

    // Clean up artifact directory
    await rm(entryPath, { recursive: true });

    // Rename to final name
    await rename(tempPath, destPath);

    console.log(`  ✅ ${entry}/ -> ${binary}`);
    flattened++;
  }

  if (flattened === 0) {
    console.log("  No directories to flatten found.");
  } else {
    console.log(`\n✅ Flattened ${flattened} binaries`);
  }

  // List final contents
  console.log("\nFinal contents:");
  await $`ls -la ${distDir}/`;
}

async function createArchives(distDir: string): Promise<void> {
  console.log(`\n📦 Creating archives...\n`);

  const entries = await readdir(distDir);
  let created = 0;

  for (const entry of entries) {
    // Skip if already an archive or not a hyperterse binary
    if (entry.endsWith(".tar.gz") || entry.endsWith(".zip")) continue;
    if (!entry.startsWith("hyperterse-")) continue;

    const entryPath = join(distDir, entry);
    if (await isDirectory(entryPath)) continue;

    const isWindows = entry.includes("windows") || entry.endsWith(".exe");
    const archiveName = isWindows
      ? entry.replace(".exe", ".zip")
      : `${entry}.tar.gz`;
    const archivePath = join(distDir, archiveName);

    if (isWindows) {
      await $`cd ${distDir} && zip ${archiveName} ${entry}`;
    } else {
      await $`cd ${distDir} && tar -czvf ${archiveName} ${entry}`;
    }

    console.log(`  ✅ ${entry} -> ${archiveName}`);
    created++;
  }

  if (created === 0) {
    console.log("  No binaries to archive found.");
  } else {
    console.log(`\n✅ Created ${created} archives`);
  }
}

async function main(): Promise<void> {
  const { values, positionals } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      "create-archives": { type: "boolean", default: false },
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Archive Script

Usage:
  bun run archive [DIR]              Flatten binaries from artifact subdirectories
  bun run archive --create-archives  Also create .tar.gz and .zip archives

Arguments:
  DIR   Distribution directory (default: dist)

This script handles the artifact structure from GitHub Actions:
  dist/hyperterse-linux-amd64/hyperterse-linux-amd64  ->  dist/hyperterse-linux-amd64
`);
    return;
  }

  const distDir = positionals[0] || "dist";

  // Check if dist directory exists
  if (!(await isDirectory(distDir))) {
    console.error(`❌ Directory not found: ${distDir}`);
    process.exit(1);
  }

  await flattenBinaries(distDir);

  if (values["create-archives"]) {
    await createArchives(distDir);
  }
}

main().catch((error) => {
  console.error("❌ Archive failed:", error);
  process.exit(1);
});
