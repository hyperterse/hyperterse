# Publishing to Distribution Registries

This document explains how to set up publishing to NPM and Homebrew registries.

## Required Secrets

The release workflow requires the following GitHub secrets to be configured:

### NPM Publishing

- **`NPM_TOKEN`** (required for NPM publishing)
  - Create an automation token at https://www.npmjs.com/settings/YOUR_USERNAME/tokens
  - Select "Automation" token type
  - Add the token as a GitHub secret named `NPM_TOKEN`

### Homebrew Tap Publishing

- **`HOMEBREW_TAP_REPO`** (optional for Homebrew publishing)
  - Format: `username/tap` (defaults to `hyperterse/tap`)
  - The tap repository must exist and be accessible
  - The repository should have a `Formula/` directory
  - Installation command: `brew install hyperterse/tap/hyperterse`

- **`HOMEBREW_TAP_TOKEN`** (optional)
  - Personal Access Token (PAT) with `repo` scope
  - If not set, uses `GITHUB_TOKEN` (may have limited permissions)
  - Recommended: Create a PAT specifically for the tap repository

## Setup Instructions

### 1. NPM Setup

1. Create an NPM account if you don't have one
2. Create an automation token at https://www.npmjs.com/settings/YOUR_USERNAME/tokens
3. Add the token as `NPM_TOKEN` in GitHub repository settings → Secrets and variables → Actions

### 2. Homebrew Tap Setup

1. Create a new repository named `tap` under your GitHub organization/user (e.g., `hyperterse/tap`)
2. Initialize it with a `Formula/` directory:
   ```bash
   mkdir -p Formula
   git init
   git add Formula/
   git commit -m "Initial commit"
   git push
   ```
3. Optionally set `HOMEBREW_TAP_REPO` secret if using a different repository name (defaults to `hyperterse/tap`)
4. Optionally create a PAT and add it as `HOMEBREW_TAP_TOKEN` for better permissions
5. Users can then install with: `brew install hyperterse/tap/hyperterse`

## How It Works

When you push a version tag (e.g., `v1.2.3`), the workflow will:

1. **Build binaries** for all platforms
2. **Create GitHub release** with binaries and checksums
3. **Publish to NPM** (if `NPM_TOKEN` is set)
   - Publishes the package from `distributions/npm/`
   - Uses the version from `package.json` (updated by `scripts/version.sh`)
4. **Update Homebrew tap** (runs automatically)
   - Calculates SHA256 checksums for all binaries
   - Updates the formula with version and checksums
   - Commits and pushes to the tap repository (`hyperterse/tap` by default)

## Testing

To test publishing without creating a release:

1. **NPM**: Run `npm publish --dry-run` from `distributions/npm/`
2. **Homebrew**: Manually update the formula and test locally

## Troubleshooting

### NPM Publishing Fails

- Verify `NPM_TOKEN` is set correctly
- Check that the package name is available on NPM
- Ensure version in `package.json` matches the tag

### Homebrew Publishing Fails

- Verify `HOMEBREW_TAP_REPO` points to an existing repository
- Check that the repository is accessible with the provided token
- Ensure the `Formula/` directory exists in the tap repository

