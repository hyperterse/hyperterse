import { $ } from "bun";
import { resolve, dirname } from "node:path";
import semver from "semver";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
const projectRoot = resolve(scriptDir, "..");
$.cwd(projectRoot);

function usage(): never {
  console.log("Usage: bun run scripts/version.ts [OPTIONS]");
  console.log("");
  console.log("Options:");
  console.log("  --major              Bump major version (e.g., 1.0.0 -> 2.0.0)");
  console.log("  --minor              Bump minor version (e.g., 1.0.0 -> 1.1.0)");
  console.log("  --patch              Bump patch version (e.g., 1.0.0 -> 1.0.1)");
  console.log("  --prerelease <tag>   Create a prerelease version (e.g., --prerelease alpha)");
  console.log("  --version <version>  Specify exact version (e.g., --version 1.2.3)");
  console.log("  --push               Push tag to remote using 'git push --follow-tags'");
  console.log("");
  console.log("Examples:");
  console.log("  bun run scripts/version.ts --major");
  console.log("  bun run scripts/version.ts --minor");
  console.log("  bun run scripts/version.ts --patch");
  console.log("  bun run scripts/version.ts --prerelease beta");
  console.log("  bun run scripts/version.ts --version 2.0.0");
  process.exit(1);
}

async function getLatestVersion(): Promise<string> {
  const output = (await $`git tag -l "v*"`.quiet().text()).trim();

  if (!output) return "0.0.0";

  const tags = output
    .split("\n")
    .filter(Boolean)
    .map((t) => t.replace(/^v/, ""))
    .filter((v) => semver.valid(v));

  if (tags.length === 0) return "0.0.0";

  // Sort using semver and return the highest
  const sorted = semver.rsort(tags);
  return sorted[0];
}

// Parse command line arguments
let bumpType: semver.ReleaseType = "patch";
let prereleaseTag = "";
let explicitVersion = "";
let push = false;

const args = process.argv.slice(2);
let i = 0;
while (i < args.length) {
  switch (args[i]) {
    case "--major":
      bumpType = "major";
      i++;
      break;
    case "--minor":
      bumpType = "minor";
      i++;
      break;
    case "--patch":
      bumpType = "patch";
      i++;
      break;
    case "--prerelease":
      bumpType = "prerelease";
      prereleaseTag = args[i + 1] || "";
      if (!prereleaseTag) {
        console.error("❌ Error: --prerelease requires a tag");
        usage();
      }
      i += 2;
      break;
    case "--version":
      explicitVersion = args[i + 1] || "";
      if (!explicitVersion) {
        console.error("❌ Error: --version requires a version number");
        usage();
      }
      i += 2;
      break;
    case "--push":
      push = true;
      i++;
      break;
    case "--help":
    case "-h":
      usage();
    default:
      console.error(`❌ Error: Unknown option: ${args[i]}`);
      usage();
  }
}

// Validate that exactly one option was provided
if (!bumpType && !explicitVersion) {
  console.error(
    "❌ Error: Must specify one of --major, --minor, --patch, --prerelease, or --version"
  );
  usage();
}

if (bumpType && explicitVersion) {
  console.error("❌ Error: Cannot specify both bump type and explicit version");
  usage();
}

// Check if we're in a git repository
try {
  await $`git rev-parse --git-dir`.quiet();
} catch {
  console.error("❌ Error: Not in a git repository");
  process.exit(1);
}

// Check for uncommitted changes (excluding distribution manifests)
try {
  await $`git diff --quiet -- ':!distributions/'`.quiet();
} catch {
  console.log("⚠️  Warning: You have uncommitted changes in your working directory");
  console.log("   (excluding distribution manifests)");
  process.stdout.write("   Continue anyway? (y/N) ");

  const { stdin } = process;
  if (stdin.isTTY) {
    stdin.setRawMode(true);
  }
  const response = await new Promise<string>((resolve) => {
    stdin.once("data", (data) => {
      if (stdin.isTTY) stdin.setRawMode(false);
      resolve(data.toString().trim());
    });
  });
  console.log();

  if (!/^[Yy]$/.test(response)) {
    console.error("❌ Aborted");
    process.exit(1);
  }
}

// Determine the new version
let newVersion: string;
if (explicitVersion) {
  newVersion = explicitVersion;
} else {
  const currentVersion = await getLatestVersion();
  console.log(`📋 Current version: v${currentVersion}`);

  const bumped = semver.inc(currentVersion, bumpType, prereleaseTag || "");
  if (!bumped) {
    console.error(`❌ Error: Failed to bump version ${currentVersion} with ${bumpType}`);
    process.exit(1);
  }
  newVersion = bumped;
}

// Check if version actually changed
let currentManifestVersion = "";
const npmPkgPath = resolve(projectRoot, "distributions/npm/package.json");
if (await Bun.file(npmPkgPath).exists()) {
  try {
    const pkg = await Bun.file(npmPkgPath).json();
    currentManifestVersion = pkg.version || "";
  } catch {
    const content = await Bun.file(npmPkgPath).text();
    const match = content.match(/"version":\s*"([^"]*)"/);
    currentManifestVersion = match?.[1] || "";
  }
}

let skipManifestUpdate = false;
if (newVersion === currentManifestVersion && currentManifestVersion) {
  console.log(`ℹ️  Version ${newVersion} is already set in manifests`);
  console.log("   Skipping manifest updates");
  skipManifestUpdate = true;
}

// Validate version format using semver
if (!semver.valid(newVersion)) {
  console.error(`❌ Error: Invalid version format: ${newVersion}`);
  console.error("   Expected format: MAJOR.MINOR.PATCH[-PRERELEASE]");
  process.exit(1);
}

// Check if tag already exists
const tagName = `v${newVersion}`;
try {
  await $`git rev-parse ${tagName}`.quiet();
  console.error(`❌ Error: Tag ${tagName} already exists`);
  process.exit(1);
} catch {
  // Tag doesn't exist — good
}

// Update distribution manifests
if (!skipManifestUpdate) {
  console.log("");
  console.log("📦 Updating distribution manifests...");

  // Update Homebrew formula
  const formulaPath = resolve(projectRoot, "distributions/homebrew/hyperterse.rb");
  if (await Bun.file(formulaPath).exists()) {
    console.log("📝 Updating Homebrew formula...");
    let formula = await Bun.file(formulaPath).text();
    formula = formula.replace(/^(\s*version\s+").*(")/m, `$1${newVersion}$2`);
    await Bun.write(formulaPath, formula);
    console.log(`   ✓ Updated ${formulaPath}`);
  } else {
    console.log(`⚠️  Warning: Homebrew formula not found: ${formulaPath}`);
  }

  // Update NPM package.json
  if (await Bun.file(npmPkgPath).exists()) {
    console.log("📝 Updating NPM package.json...");
    const pkg = await Bun.file(npmPkgPath).json();
    pkg.version = newVersion;
    await Bun.write(npmPkgPath, JSON.stringify(pkg, null, 2) + "\n");
    console.log(`   ✓ Updated ${npmPkgPath}`);
  } else {
    console.log(`⚠️  Warning: package.json not found: ${npmPkgPath}`);
  }

  // Check if there are manifest changes to commit
  try {
    await $`git diff --quiet distributions/`.quiet();
    console.log("   ℹ️  No manifest changes detected");
  } catch {
    console.log("");
    console.log("💾 Committing manifest changes...");
    await $`git add distributions/homebrew/hyperterse.rb distributions/npm/package.json`;
    await $`git commit -m ${"Update distribution manifests to v" + newVersion}`;
    console.log("   ✓ Committed manifest changes");
  }
}

// Get timestamp
const timestamp = new Date()
  .toISOString()
  .replace("T", " ")
  .replace(/\.\d+Z/, " UTC");

// Create annotated tag
console.log("");
console.log(`🏷️  Creating tag: ${tagName}`);
console.log(`    Timestamp: ${timestamp}`);

await $`git tag -a ${tagName} -m ${"Release " + tagName + "\n\nTimestamp: " + timestamp}`;

console.log("");
console.log(`✅ Successfully created tag: ${tagName}`);

// Push if --push flag was provided
if (push) {
  console.log("");
  console.log("🚀 Pushing tag to remote...");
  await $`git push --follow-tags`;
  console.log(`✅ Successfully pushed tag: ${tagName}`);
} else {
  console.log("");
  console.log("Next steps:");
  console.log("  git push --follow-tags    # Push the tag and commits to remote");
  console.log(`  Or use: bun run scripts/version.ts --version ${newVersion} --push`);
}
