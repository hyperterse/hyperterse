#!/usr/bin/env bun
/**
 * Version management script for Hyperterse
 * Handles version bumping, tagging, and manifest updates
 *
 * Usage:
 *   bun run version major          Bump major version (1.0.0 -> 2.0.0)
 *   bun run version minor          Bump minor version (1.0.0 -> 1.1.0)
 *   bun run version patch          Bump patch version (1.0.0 -> 1.0.1)
 *   bun run version prerelease     Create prerelease (1.0.0 -> 1.0.1-0)
 *   bun run version 2.0.0          Set explicit version
 *   bun run version                Show current version
 *
 * Options:
 *   --push          Push tag to remote after creating
 *   --dry-run       Show what would be done without making changes
 *   --preid <tag>   Prerelease identifier (alpha, beta, rc)
 */

import { $ } from "bun";
import semver from "semver";
import { readFile, writeFile } from "fs/promises";
import { existsSync } from "fs";

const CARGO_FILES = [
  "Cargo.toml",
  "src/modules/cli/Cargo.toml",
  "src/modules/core/Cargo.toml",
  "src/modules/parser/Cargo.toml",
  "src/modules/runtime/Cargo.toml",
  "src/modules/types/Cargo.toml",
];

async function getLatestVersion(): Promise<string> {
  try {
    const result = await $`git tag -l "v*" --sort=-v:refname | head -n 1`.text();
    const tag = result.trim();
    if (tag && semver.valid(tag.replace(/^v/, ""))) {
      return tag.replace(/^v/, "");
    }
  } catch {
    // Ignore errors
  }
  return "0.0.0";
}

async function updateCargoToml(version: string): Promise<void> {
  for (const file of CARGO_FILES) {
    if (!existsSync(file)) continue;

    const content = await readFile(file, "utf-8");
    const updated = content.replace(
      /^(version\s*=\s*")[^"]+(")/m,
      `$1${version}$2`
    );
    await writeFile(file, updated);
    console.log(`  ✓ ${file}`);
  }
}

async function updateHomebrewFormula(version: string): Promise<void> {
  const formulaPath = "distributions/homebrew/hyperterse.rb";
  if (!existsSync(formulaPath)) return;

  const content = await readFile(formulaPath, "utf-8");
  const updated = content.replace(
    /^(\s*version\s+")[^"]+(")/m,
    `$1${version}$2`
  );
  await writeFile(formulaPath, updated);
  console.log(`  ✓ ${formulaPath}`);
}

async function updateNpmPackage(version: string): Promise<void> {
  const packagePath = "distributions/npm/package.json";
  if (!existsSync(packagePath)) return;

  const content = await readFile(packagePath, "utf-8");
  const pkg = JSON.parse(content);
  pkg.version = version;
  await writeFile(packagePath, JSON.stringify(pkg, null, 2) + "\n");
  console.log(`  ✓ ${packagePath}`);
}

async function createTag(version: string, push: boolean): Promise<void> {
  const tagName = `v${version}`;

  // Check if tag already exists
  try {
    await $`git rev-parse ${tagName}`.quiet();
    throw new Error(`Tag ${tagName} already exists`);
  } catch (e) {
    if (e instanceof Error && e.message.includes("already exists")) {
      throw e;
    }
  }

  // Commit manifest changes if any
  try {
    await $`git add Cargo.toml */Cargo.toml src/*/Cargo.toml distributions/`.quiet();
    await $`git diff --cached --quiet`.quiet();
  } catch {
    await $`git commit -m "Release version ${version}"`;
    console.log(`  ✓ Committed changes`);
  }

  // Create annotated tag
  await $`git tag -a ${tagName} -m ${"Release " + tagName}`;
  console.log(`  ✓ Created tag: ${tagName}`);

  if (push) {
    await $`git push --follow-tags`;
    console.log(`  ✓ Pushed to remote`);
  }
}

function showHelp(): void {
  console.log(`
Hyperterse Version Manager

Usage:
  bun run version [command] [options]

Commands:
  major              Bump major version (1.0.0 -> 2.0.0)
  minor              Bump minor version (1.0.0 -> 1.1.0)
  patch              Bump patch version (1.0.0 -> 1.0.1)
  prerelease         Bump prerelease (1.0.0 -> 1.0.1-0, or 1.0.1-alpha.0 with --preid)
  premajor           Bump premajor (1.0.0 -> 2.0.0-0)
  preminor           Bump preminor (1.0.0 -> 1.1.0-0)
  prepatch           Bump prepatch (1.0.0 -> 1.0.1-0)
  <version>          Set explicit version (e.g., 2.0.0, 1.0.0-beta.1)
  (no command)       Show current version

Options:
  --push             Push tag to remote after creating
  --dry-run          Show what would be done without making changes
  --preid <id>       Prerelease identifier (alpha, beta, rc)
  --help, -h         Show this help message

Examples:
  bun run version                      Show current version
  bun run version patch                1.0.0 -> 1.0.1
  bun run version minor --push         1.0.0 -> 1.1.0 and push
  bun run version prerelease --preid alpha   1.0.0 -> 1.0.1-alpha.0
  bun run version 2.0.0-rc.1           Set explicit version
`);
}

async function main(): Promise<void> {
  const args = Bun.argv.slice(2);
  
  // Parse flags
  const push = args.includes("--push");
  const dryRun = args.includes("--dry-run");
  const help = args.includes("--help") || args.includes("-h");
  
  // Get preid value
  const preidIndex = args.indexOf("--preid");
  const preid = preidIndex !== -1 ? args[preidIndex + 1] : undefined;
  
  // Filter out flags to get the command
  const command = args.find(arg => 
    !arg.startsWith("--") && 
    !arg.startsWith("-") && 
    arg !== preid
  );

  if (help) {
    showHelp();
    return;
  }

  const currentVersion = await getLatestVersion();

  // No command = show current version
  if (!command) {
    console.log(`Current version: v${currentVersion}`);
    return;
  }

  // Calculate new version
  let newVersion: string | null = null;
  const releaseTypes = ["major", "minor", "patch", "premajor", "preminor", "prepatch", "prerelease"] as const;
  
  if (releaseTypes.includes(command as any)) {
    newVersion = semver.inc(currentVersion, command as semver.ReleaseType, preid);
  } else if (semver.valid(command)) {
    newVersion = command;
  } else {
    console.error(`Error: Invalid command or version "${command}"`);
    console.error(`Run "bun run version --help" for usage`);
    process.exit(1);
  }

  if (!newVersion) {
    console.error(`Error: Could not calculate new version from "${currentVersion}"`);
    process.exit(1);
  }

  console.log(`\nVersion: v${currentVersion} -> v${newVersion}\n`);

  if (dryRun) {
    console.log("Dry run - would update:");
    CARGO_FILES.filter(existsSync).forEach(f => console.log(`  - ${f}`));
    console.log("  - distributions/homebrew/hyperterse.rb");
    console.log("  - distributions/npm/package.json");
    console.log(`  - Create git tag v${newVersion}`);
    if (push) console.log("  - Push to remote");
    return;
  }

  console.log("Updating manifests...");
  await updateCargoToml(newVersion);
  await updateHomebrewFormula(newVersion);
  await updateNpmPackage(newVersion);

  console.log("\nCreating tag...");
  await createTag(newVersion, push);

  console.log(`\n✓ Version bumped to v${newVersion}`);

  if (!push) {
    console.log("\nTo push: git push --follow-tags");
  }
}

main().catch((error) => {
  console.error("Error:", error.message);
  process.exit(1);
});
