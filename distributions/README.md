# Distribution Manifests

This directory contains distribution manifests for different package managers. These manifests are **automatically updated** by `scripts/version.sh` when you create a new version tag.

## Automatic Version Updates

When you run `scripts/version.sh` to create a new version tag, it will:
1. Update all distribution manifests with the new version
2. Commit the manifest changes
3. Create the git tag

**Example:**
```bash
# This will update all manifests to 1.2.3, commit them, and create tag v1.2.3
./scripts/version.sh --version 1.2.3

# Or use bump commands
./scripts/version.sh --patch  # Bumps patch version and updates all manifests
```

## Homebrew

**File:** `homebrew/hyperterse.rb`

The formula automatically detects the platform (macOS/Linux) and architecture (Intel/ARM) to download the correct binary. URLs use Ruby string interpolation (`#{version}`) to automatically use the correct version.

**Installation:**
```bash
brew install hyperterse/tap/hyperterse
```

**Manual usage (from local file):**
```bash
brew install distributions/homebrew/hyperterse.rb
```

## NPM

**Files:** `npm/package.json`, `npm/install.js`

The `install.js` postinstall script automatically detects the platform and architecture to download the correct binary during `npm install`. The version in `package.json` is automatically updated by `scripts/version.sh`.

**Manual usage:**
```bash
cd distributions/npm
npm install
```

**Note:** The `install.js` script can also use environment variables as a fallback:
- `HYPERTERSE_VERSION` or `VERSION` - Override version for binary download
- Falls back to `package.json` version if not set
