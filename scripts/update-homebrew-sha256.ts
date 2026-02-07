import { resolve, dirname } from "node:path";

// Parse arguments
const [formulaFile, darwinAmd64, darwinArm64, linuxAmd64, linuxArm64, linuxArm] =
  process.argv.slice(2);

if (!formulaFile || !(await Bun.file(formulaFile).exists())) {
  console.error(`Error: Formula file not found: ${formulaFile}`);
  process.exit(1);
}

let content = await Bun.file(formulaFile).text();

// Update SHA256 for each platform
const replacements: [string, string | undefined][] = [
  ["darwin-amd64", darwinAmd64],
  ["darwin-arm64", darwinArm64],
  ["linux-amd64", linuxAmd64],
  ["linux-arm64", linuxArm64],
  ["linux-arm", linuxArm],
];

for (const [platform, sha256] of replacements) {
  if (!sha256) continue;

  // Match the line after the platform URL line and replace its sha256
  const pattern = new RegExp(
    `(${platform.replace("-", "\\-")}"\\s*\\n\\s*)sha256 "[^"]*"`,
    "g"
  );
  content = content.replace(pattern, `$1sha256 "${sha256}"`);
}

await Bun.write(formulaFile, content);

console.log(`✓ Updated SHA256 checksums in ${formulaFile}`);
