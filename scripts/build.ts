import { $ } from "bun";
import { resolve, dirname } from "node:path";
import semver from "semver";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
$.cwd(resolve(scriptDir, ".."));

// Parse arguments
const [goos, goarch, outputDir = "dist", outputName = "hyperterse"] =
  process.argv.slice(2);

// Get version from git tag
let rawVersion = (
  await $`git describe --tags --always --dirty 2>/dev/null || echo "dev"`.text()
).trim();

// Clean git describe output to a valid semver
// Remove 'v' prefix if present
if (rawVersion.startsWith("v")) rawVersion = rawVersion.slice(1);
// Remove -dirty suffix if present
rawVersion = rawVersion.replace(/-dirty$/, "");
// Remove commit hash suffix (e.g., 1.0.0-5-gabc1234 -> 1.0.0)
// But preserve prerelease tags (e.g., 1.0.0-alpha.1 stays as-is)
if (/-\d+-g[0-9a-f]+$/.test(rawVersion)) {
  rawVersion = rawVersion.replace(/-\d+-g[0-9a-f]+$/, "");
}

// Use semver to validate/coerce the version, fall back to raw string
const version = semver.valid(semver.coerce(rawVersion)) ?? rawVersion;

await $`mkdir -p ${outputDir}`;

if (goos && goarch) {
  // Cross-compilation mode
  let binName = `${outputName}-${goos}-${goarch}`;
  if (goos === "windows") binName += ".exe";

  console.log(`Building ${binName} for ${goos}/${goarch}...`);

  await $`CGO_ENABLED=0 GOOS=${goos} GOARCH=${goarch} go build -trimpath -ldflags=${`-s -w -X main.Version=${version}`} -o ${`${outputDir}/${binName}`} .`;

  console.log(`✓ Built ${binName} (version: ${version})`);
} else {
  // Local build mode (current platform)
  console.log("Building hyperterse...");

  await $`go build -mod=mod -trimpath -ldflags=${`-s -w -X main.Version=${version}`} -o ${`${outputDir}/${outputName}`} .`;

  console.log(`✓ Build complete (version: ${version})`);
}
