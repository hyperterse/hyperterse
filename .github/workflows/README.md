# GitHub Actions Workflows

This directory contains GitHub Actions workflows for building, releasing, and publishing Hyperterse.

## Workflow Files

### Main Workflows

- **`release.yml`** - Main release workflow triggered on version tags
  - Builds binaries for all platforms
  - Creates GitHub release
  - Publishes to NPM and Homebrew using composite actions

### Composite Actions

- **`.github/actions/publish-npm/action.yml`** - Composite action for publishing to NPM
  - Inputs: `version` (required), `token` (required - NPM_TOKEN)
  - Verifies version matches `package.json`
  - Publishes to NPM registry

- **`.github/actions/publish-homebrew/action.yml`** - Composite action for publishing to Homebrew
  - Inputs: `version` (required), `tap_repo` (optional, defaults to `hyperterse/tap`), `token` (optional)
  - Downloads binaries from GitHub release or uses provided artifacts
  - Calculates SHA256 checksums
  - Updates Homebrew formula and pushes to tap repository

### Manual Workflows

- **`manual-publish-npm.yml`** - Manual workflow for publishing to NPM
  - Triggered via `workflow_dispatch`
  - Extracts version from `distributions/npm/package.json`
  - Uses `publish-npm` composite action

- **`manual-publish-homebrew.yml`** - Manual workflow for publishing to Homebrew
  - Triggered via `workflow_dispatch`
  - Extracts version from `distributions/homebrew/hyperterse.rb`
  - Optional input: `tap_repo` (defaults to `hyperterse/tap`)
  - Uses `publish-homebrew` composite action

## Usage

### Automatic Publishing (on Release)

When you push a version tag (e.g., `v1.2.3`), the `release.yml` workflow will:
1. Build binaries for all platforms
2. Create a GitHub release
3. Automatically publish to NPM and Homebrew

### Manual Publishing

#### Publish to NPM Only

1. Go to Actions → Manual Publish to NPM
2. Click "Run workflow"
3. The workflow will read the version from `distributions/npm/package.json` and publish

#### Publish to Homebrew Only

1. Go to Actions → Manual Publish to Homebrew
2. Click "Run workflow"
3. Optionally specify a different tap repository
4. The workflow will:
   - Read the version from `distributions/homebrew/hyperterse.rb`
   - Download binaries from the GitHub release for that version
   - Calculate SHA256 checksums
   - Update and push to the Homebrew tap

## Requirements

### NPM Publishing
- `NPM_TOKEN` secret must be set in repository settings

### Homebrew Publishing
- `hyperterse/tap` repository must exist (or specify different repo)
- `HOMEBREW_TAP_TOKEN` secret is optional (uses `GITHUB_TOKEN` if not set)

## Version Management

Versions are managed in:
- **NPM**: `distributions/npm/package.json` → `version` field
- **Homebrew**: `distributions/homebrew/hyperterse.rb` → `version` field

These are automatically updated by `scripts/version.sh` when creating version tags.
