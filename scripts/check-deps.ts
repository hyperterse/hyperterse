import { $ } from "bun";
import { resolve, dirname } from "node:path";

// Ensure we're in the project root
const scriptDir = dirname(new URL(import.meta.url).pathname);
$.cwd(resolve(scriptDir, ".."));

// Check if Go is installed
if (!Bun.which("go")) {
  console.error(
    "❌ Go is not installed. Please install Go 1.25.1 or later from https://go.dev/dl/"
  );
  process.exit(1);
}

const goVersion = (await $`go version`.text()).trim();
console.log(`✓ Go found: ${goVersion}`);

// Check if protoc is installed
if (!Bun.which("protoc")) {
  console.log("📦 Installing protoc...");

  if (process.platform === "darwin") {
    if (Bun.which("brew")) {
      await $`brew install protobuf`;
    } else {
      console.error(
        "❌ Homebrew not found. Please install protoc manually:\n   brew install protobuf"
      );
      process.exit(1);
    }
  } else {
    console.error(
      "❌ protoc not found. Please install protoc manually:\n   sudo apt-get install protobuf-compiler  # Debian/Ubuntu\n   sudo yum install protobuf-compiler       # RHEL/CentOS"
    );
    process.exit(1);
  }
} else {
  const protocVersion = (await $`protoc --version`.text()).trim();
  console.log(`✓ protoc found: ${protocVersion}`);
}

// Install protoc-gen-go if not present
const goPath = (await $`go env GOPATH`.text()).trim();
const protocGenGoInPath = Bun.which("protoc-gen-go");
const protocGenGoInGoPath = await Bun.file(
  `${goPath}/bin/protoc-gen-go`
).exists();

if (!protocGenGoInPath && !protocGenGoInGoPath) {
  console.log("📦 Installing protoc-gen-go...");
  await $`go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`;
} else {
  console.log("✓ protoc-gen-go found");
}

// Download Go dependencies
console.log("📥 Downloading Go dependencies...");
await $`go mod download`;
