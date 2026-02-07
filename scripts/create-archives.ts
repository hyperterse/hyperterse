import { resolve, dirname } from "node:path";
import { readdirSync, statSync, renameSync, rmSync } from "node:fs";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
const projectRoot = resolve(scriptDir, "..");

const distDir = resolve(projectRoot, process.argv[2] || "dist");

if (!statSync(distDir, { throwIfNoEntry: false })?.isDirectory()) {
  console.error(`Error: Directory ${distDir} does not exist`);
  process.exit(1);
}

// Flatten binaries from artifact subdirectories to dist root
// After download-artifact, structure is: dist/hyperterse-linux-amd64/hyperterse-linux-amd64
// This script flattens to: dist/hyperterse-linux-amd64

const entries = readdirSync(distDir);

for (const entry of entries) {
  const entryPath = resolve(distDir, entry);
  const stat = statSync(entryPath, { throwIfNoEntry: false });

  if (!stat?.isDirectory() || !entry.startsWith("hyperterse-")) continue;

  // Find binary inside the directory
  const innerEntries = readdirSync(entryPath);
  const binary = innerEntries.find(
    (f) =>
      f.startsWith("hyperterse-") &&
      statSync(resolve(entryPath, f)).isFile()
  );

  if (!binary) continue;

  const binaryPath = resolve(entryPath, binary);
  const tmpPath = resolve(distDir, `${binary}.tmp`);
  const finalPath = resolve(distDir, binary);

  // Move binary to temp location first to avoid conflict with same-named directory
  renameSync(binaryPath, tmpPath);

  // Clean up artifact directory
  rmSync(entryPath, { recursive: true, force: true });

  // Rename to final name
  renameSync(tmpPath, finalPath);
}

console.log("✓ Binaries flattened successfully");
