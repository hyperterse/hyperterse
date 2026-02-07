# Contributing to Hyperterse

Thanks for helping improve Hyperterse. This repository has been migrated to **Rust**, so the development workflow is centered around **Cargo** (Rust) and **Bun** (repo scripts).

## Code of Conduct

Please read our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you agree to uphold a respectful, inclusive environment.

## Prerequisites

- **Rust toolchain** (via rustup): includes `rustc` and `cargo`
- **Bun**: used for the repo’s convenience scripts (`bun run ...`)
- **Git**

Optional (only if you’re working on proto/schema tooling or local DB testing):
- **Docker**

## Quick setup

```bash
git clone https://github.com/hyperterse/hyperterse.git
cd hyperterse

bun run setup
bun run build
bun run test
```

## Common commands

- **build**: `bun run build` (debug), `bun run build:release` (release)
- **fast compile check**: `bun run check` (or `bun run check --all`)
- **format**: `bun run fmt` (or `bun run fmt:check`)
- **lint**: `bun run lint` (or `bun run lint --strict`)
- **tests**: `bun run test`

You can always run Cargo commands directly too (e.g., `cargo test --all-features`).

## Project structure (Rust workspace)

```text
hyperterse/
├── src/
│   ├── modules/
│   │   ├── cli/             # CLI binary + commands
│   │   ├── core/            # Core domain model + shared errors/types
│   │   ├── parser/          # Config parsing + validation
│   │   ├── runtime/         # Runtime server, connectors, handlers (MCP/OpenAPI)
│   │   └── types/           # Public enums/types (connector + primitive types)
│   └── schema/              # Generated JSON schema (committed)
├── docs/                    # Documentation site (Astro)
└── distributions/           # Homebrew + NPM distribution assets
```

## Development workflow

1. **Create a branch**

```bash
git checkout -b feature/your-feature-name
# or fix/..., docs/..., refactor/..., chore/...
```

2. **Make changes**

- Keep PRs focused and reviewable
- Add tests for behavior changes
- Update docs when user-facing behavior changes

3. **Before submitting**

```bash
bun run fmt:check
bun run lint --strict
bun run test
bun run check --all
```

## Code style guidelines (Rust)

- **Formatting**: `cargo fmt`
- **Linting**: `cargo clippy --all-targets --all-features -- -D warnings`
- **Errors**: prefer `thiserror` for domain errors and `anyhow` at app boundaries (where appropriate)

## Adding new features

### Adding a new connector

1. Add a new variant in `hyperterse-types/src/connector.rs`.
2. Implement the connector in `hyperterse-runtime/src/connectors/<name>.rs` by implementing the `Connector` trait (`hyperterse-runtime/src/connectors/traits.rs`).
3. Register it in `hyperterse-runtime/src/connectors/manager.rs` (the `match` in `create_connector`).
4. Update docs (and `schema/terse.schema.json` if you add a new `connector` enum value).

### Adding a new primitive type

1. Add the new variant in `hyperterse-types/src/primitive.rs`.
2. Update validation/conversion logic in `hyperterse-parser/src/validator.rs` (and any runtime mapping code if needed).
3. Update docs/schema as appropriate.

## Getting help

- **Docs**: `https://docs.hyperterse.com`
- **Issues**: `https://github.com/hyperterse/hyperterse/issues`
- **Discussions**: `https://github.com/hyperterse/hyperterse/discussions`
