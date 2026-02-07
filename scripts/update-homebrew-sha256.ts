#!/usr/bin/env bun
/**
 * Update SHA256 checksums in the Homebrew formula.
 *
 * Usage:
 *   bun run scripts/update-homebrew-sha256.ts <formula_file> <darwin_amd64> <darwin_arm64> <linux_amd64> <linux_arm64> <linux_arm>
 */

import { readFile, writeFile } from "fs/promises";
import { parseArgs } from "util";

type ShaArgs = {
  darwinAmd64: string;
  darwinArm64: string;
  linuxAmd64: string;
  linuxArm64: string;
  linuxArm: string;
};

function applySha(
  content: string,
  platform: "darwin-amd64" | "darwin-arm64" | "linux-amd64" | "linux-arm64" | "linux-arm",
  sha256: string,
): string {
  if (!sha256) return content;

  const re = new RegExp(`(${platform}"\\s*\\n\\s*)sha256\\s+"[^"]*"`, "g");
  return content.replace(re, `$1sha256 "${sha256}"`);
}

async function main(): Promise<void> {
  const { positionals, values } = parseArgs({
    args: Bun.argv.slice(2),
    options: {
      help: { type: "boolean", short: "h", default: false },
    },
    allowPositionals: true,
  });

  if (values.help) {
    console.log(`
Hyperterse Homebrew SHA256 Updater

Usage:
  bun run scripts/update-homebrew-sha256.ts <formula_file> <darwin_amd64> <darwin_arm64> <linux_amd64> <linux_arm64> <linux_arm>
`);
    return;
  }

  const [formulaFile, darwinAmd64, darwinArm64, linuxAmd64, linuxArm64, linuxArm] =
    positionals as string[];

  if (!formulaFile) {
    console.error("❌ Missing formula file path.");
    process.exit(1);
  }

  const sha: ShaArgs = {
    darwinAmd64: darwinAmd64 || "",
    darwinArm64: darwinArm64 || "",
    linuxAmd64: linuxAmd64 || "",
    linuxArm64: linuxArm64 || "",
    linuxArm: linuxArm || "",
  };

  const original = await readFile(formulaFile, "utf8");

  let updated = original;
  updated = applySha(updated, "darwin-amd64", sha.darwinAmd64);
  updated = applySha(updated, "darwin-arm64", sha.darwinArm64);
  updated = applySha(updated, "linux-amd64", sha.linuxAmd64);
  updated = applySha(updated, "linux-arm64", sha.linuxArm64);
  updated = applySha(updated, "linux-arm", sha.linuxArm);

  if (updated === original) {
    console.log(`ℹ️  No changes applied to ${formulaFile}`);
    return;
  }

  await writeFile(formulaFile, updated, "utf8");
  console.log(`✓ Updated SHA256 checksums in ${formulaFile}`);
}

main().catch((error) => {
  console.error("❌ Failed to update Homebrew formula SHA256:", error);
  process.exit(1);
});

