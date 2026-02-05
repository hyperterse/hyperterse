# GitHub Actions Workflows

This directory contains GitHub Actions workflows for building, testing, releasing, and publishing Hyperterse.

## Workflow Files

### CI Workflow

- **`ci.yml`** - Continuous Integration workflow for pull requests and pushes
  - Runs on: `push` to main/refactor/rust, `pull_request` to main
  - Jobs:
    - **Check**: `cargo check` for fast compilation verification
    - **Format**: `cargo fmt --check` for code formatting
    - **Clippy**: Rust linter with `-D warnings`
    - **Test**: Full test suite
    - **Test (platforms)**: Tests on ubuntu, macos, and windows
    - **Build (platforms)**: Release builds on all platforms
    - **Security**: `cargo audit` for dependency vulnerabilities

### Release Workflow

- **`release.yml`** - Release workflow triggered on version tags (`v*`)
  - Builds binaries for all supported platforms:
    - `x86_64-unknown-linux-gnu` (linux-amd64)
    - `aarch64-unknown-linux-gnu` (linux-arm64, via cross)
    - `x86_64-apple-darwin` (darwin-amd64)
    - `aarch64-apple-darwin` (darwin-arm64)
    - `x86_64-pc-windows-msvc` (windows-amd64)
  - Creates GitHub release with checksums
  - Optionally publishes to NPM and Homebrew

### Publish Workflow

- **`publish.yml`** - Reusable workflow for publishing to distribution channels
  - Can be called from `release.yml` or run manually
  - Supports NPM and Homebrew publishing

### Composite Actions

- **`.github/actions/publish-npm/action.yml`** - Composite action for publishing to NPM
  - Inputs: `version` (required), `token` (required - NPM_TOKEN)
  - Verifies version matches `package.json`
  - Publishes to NPM registry

- **`.github/actions/publish-homebrew/action.yml`** - Composite action for publishing to Homebrew
  - Inputs: `version` (required), `tap_repo` (optional), `token` (optional)
  - Downloads binaries from GitHub release
  - Calculates SHA256 checksums
  - Updates Homebrew formula and pushes to tap repository

## Usage

### Automatic CI (on Push/PR)

Every push and pull request to main triggers the CI workflow, which:
1. Checks code formatting with `cargo fmt`
2. Runs linting with `cargo clippy`
3. Runs the full test suite
4. Builds release binaries for all platforms

### Creating a Release

1. Update version manifests using the version script:
   ```bash
   bun run version:patch   # or --minor, --major
   ```

2. Push the tag to trigger the release workflow:
   ```bash
   git push --follow-tags
   ```

3. The release workflow will:
   - Build binaries for all platforms
   - Create a GitHub release with release notes
   - Generate SHA256 checksums
   - (Optionally) Publish to NPM and Homebrew

### Manual Publishing

#### Publish to NPM Only

1. Go to Actions → Publish
2. Click "Run workflow"
3. Select `npm: true`
4. The workflow will publish to NPM

#### Publish to Homebrew Only

1. Go to Actions → Publish
2. Click "Run workflow"
3. Select `homebrew: true`
4. The workflow will update the Homebrew tap

## Requirements

### Secrets

- **`NPM_TOKEN`**: Required for NPM publishing
- **`HOMEBREW_TAP_TOKEN`**: Optional for Homebrew publishing (uses `GITHUB_TOKEN` if not set)

### Build Dependencies

The workflows automatically install:
- Rust toolchain (stable)
- `cross` for cross-compilation (Linux ARM64)
- `cargo-audit` for security audits

## Version Management

Versions are managed using the `bun run version:*` scripts:

```bash
bun run version:major      # 1.0.0 → 2.0.0
bun run version:minor      # 1.0.0 → 1.1.0
bun run version:patch      # 1.0.0 → 1.0.1
bun run version:bump --prerelease alpha  # 1.0.0 → 1.0.0-alpha.1
```

This automatically updates:
- All `Cargo.toml` files (workspace and crates)
- `distributions/npm/package.json`
- `distributions/homebrew/hyperterse.rb`

## Cross-Compilation

For Linux ARM64, the workflow uses [cross](https://github.com/cross-rs/cross) for cross-compilation. Other targets build natively on their respective runners.

## Caching

All workflows use [rust-cache](https://github.com/Swatinem/rust-cache) for efficient caching of:
- Cargo registry
- Cargo index
- Target directories (keyed by platform/target)
