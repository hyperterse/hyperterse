# Contributing to Hyperterse

Thank you for your interest in contributing to Hyperterse! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Adding New Features](#adding-new-features)
- [Release Process](#release-process)

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. By participating in this project, you agree to maintain a respectful and inclusive environment for all contributors.

Please read our [Code of Conduct](CODE_OF_CONDUCT.md) to understand the standards and expectations for behavior in our community.

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.25.1+** - [Download Go](https://go.dev/dl/)
- **protoc** (Protocol Buffers compiler) - Required for code generation
  - macOS: `brew install protobuf`
  - Linux: `sudo apt-get install protobuf-compiler` (Debian/Ubuntu) or `sudo yum install protobuf-compiler` (RHEL/CentOS)
- **protoc-gen-go** - Installed automatically during setup
- **Git** - For version control

### Initial Setup

1. **Fork and clone the repository:**

```bash
git clone https://github.com/hyperterse/hyperterse.git
cd hyperterse
```

2. **Set up your development environment:**

```bash
make setup
```

This will:

- Check and install required dependencies (Go, protoc, protoc-gen-go)
- Download Go module dependencies
- Generate protobuf code and type definitions

3. **Build the project:**

```bash
make build
```

4. **Verify everything works:**

```bash
go test ./...
```

## Development Environment

### Using Make Commands

The project includes several Make targets for common tasks:

```bash
make help      # Show all available commands
make setup     # Complete setup (install deps and generate code)
make generate  # Regenerate protobuf files
make build     # Build the project
make run       # Build and run (requires CONFIG_FILE env var)
```

### Hot Reload Development

For active development with automatic reloading:

1. **Using Air (recommended for code changes):**

```bash
# Air is included as a tool dependency
air
```

This watches for changes and automatically rebuilds and restarts the server.

2. **Using Dev Mode (for configuration changes):**

```bash
./dist/hyperterse dev -f config.terse
```

This watches your configuration file and reloads when it changes.

### IDE Setup

The project includes VS Code settings (`.vscode/settings.json`) that configure:

- Format on save for Go files
- Prettier for JSON/YAML/MDX files
- YAML schema validation for `.terse` files

## Project Structure

```
hyperterse/
├── core/                    # Core runtime code
│   ├── cli/                 # CLI commands and interface
│   ├── logger/              # Logging utilities
│   ├── parser/              # Configuration parsers (YAML, DSL)
│   ├── runtime/             # Runtime server implementation
│   │   ├── connectors/      # Database connectors (PostgreSQL, MySQL, Redis)
│   │   ├── executor/        # Query execution engine
│   │   ├── handlers/        # HTTP handlers (OpenAPI, MCP, LLMs.txt)
│   │   └── server/          # HTTP server setup
│   └── proto/               # Generated protobuf code
├── proto/                   # Protobuf definitions
│   ├── connectors/          # Connector type definitions
│   ├── primitives/          # Primitive type definitions
│   ├── hyperterse/          # Core Hyperterse types
│   └── runtime/             # Runtime types
├── scripts/                 # Build and utility scripts
│   ├── setup.sh             # Development setup
│   ├── build.sh              # Build script
│   ├── generate-proto.sh     # Protobuf generation
│   └── check-deps.sh         # Dependency checking
├── distributions/           # Distribution packages
│   ├── npm/                 # NPM package
│   └── homebrew/            # Homebrew formula
├── docs/                    # Documentation site (Astro)
├── schema/                  # JSON schemas for root/adapter/route .terse files
└── main.go                  # Application entry point
```

## Development Workflow

### 1. Create a Branch

Always create a new branch for your work:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
# or
git checkout -b docs/your-documentation-update
```

**Branch naming conventions:**

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation updates
- `refactor/` - Code refactoring
- `test/` - Test improvements
- `chore/` - Maintenance tasks

### 2. Make Your Changes

- Write clean, maintainable code
- Follow the [code style guidelines](#code-style-guidelines)
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests for a specific package
go test ./core/runtime/...

# Run tests with verbose output
go test -v ./...
```

### 4. Ensure Code Quality

Before committing:

```bash
# Format code
go fmt ./...

# Run tests
go test ./...

# Build to ensure compilation
make build
```

### 5. Commit Your Changes

Write clear, descriptive commit messages:

```bash
git commit -m "Add support for Redis pub/sub queries"
```

**Commit message guidelines:**

- Use present tense ("Add feature" not "Added feature")
- Keep the first line under 72 characters
- Add a blank line between the subject and body if needed
- Reference issues: "Fix #123" or "Closes #456"

### 6. Keep Your Branch Updated

Regularly sync with the main branch:

```bash
git fetch origin
git rebase origin/main
```

## Code Style Guidelines

### Go Code Style

- **Formatting:** Use `go fmt` to format code. The project uses standard Go formatting.
- **Naming:** Follow Go naming conventions:
  - Exported functions/types: PascalCase
  - Unexported functions/types: camelCase
  - Constants: PascalCase or UPPER_CASE
- **Comments:** Add comments for all exported functions, types, and packages
- **Line length:** Keep lines under 100 characters (see `.editorconfig`)
- **Error handling:** Always handle errors explicitly; don't ignore them

### File Organization

- One package per directory
- Keep files focused and reasonably sized
- Group related functionality together

### Example

```go
// Package executor provides query execution functionality.
package executor

// ExecuteQuery runs a query against the specified connector.
// It validates inputs, substitutes template variables, and returns results.
func ExecuteQuery(ctx context.Context, query *Query, connector Connector) (*Result, error) {
    // Implementation
}
```

### Editor Configuration

The project includes `.editorconfig` with these settings:

- Line endings: LF
- Indent: 2 spaces
- Charset: UTF-8
- Max line length: 100 characters
- Trim trailing whitespace

## Testing Guidelines

### Writing Tests

- **Test files:** Place test files alongside source files with `_test.go` suffix
- **Test functions:** Use `TestXxx` naming convention
- **Table-driven tests:** Prefer table-driven tests for multiple scenarios
- **Test coverage:** Aim for high test coverage, especially for critical paths

### Test Structure

```go
func TestExecuteQuery(t *testing.T) {
    tests := []struct {
        name    string
        query   *Query
        want    *Result
        wantErr bool
    }{
        {
            name: "valid query",
            query: &Query{...},
            want: &Result{...},
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ExecuteQuery(context.Background(), tt.query, mockConnector)
            if (err != nil) != tt.wantErr {
                t.Errorf("ExecuteQuery() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Assertions...
        })
    }
}
```

### Testing Best Practices

- Test both success and error cases
- Test edge cases and boundary conditions
- Use mocks for external dependencies
- Keep tests fast and isolated
- Test exported functions primarily, but test unexported functions if they're complex

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage report
go test -cover ./...

# Run with coverage HTML output
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests for a specific package
go test ./core/runtime/executor
```

## Documentation

### Code Documentation

- **Package comments:** Add a package-level comment describing the package's purpose
- **Exported symbols:** Document all exported functions, types, and constants
- **Complex logic:** Add inline comments for non-obvious code

### User Documentation

- **README.md:** Update for user-facing changes
- **docs/:** Update documentation site for new features or changes
- **Examples:** Keep examples in README and docs up to date

### Documentation Style

- Use clear, concise language
- Include code examples where helpful
- Keep documentation current with code changes

## Submitting Changes

### Before Submitting

1. ✅ All tests pass (`go test ./...`)
2. ✅ Code is formatted (`go fmt ./...`)
3. ✅ Code builds successfully (`make build`)
4. ✅ Documentation is updated
5. ✅ Commit messages are clear and descriptive
6. ✅ Branch is up to date with `main`

### Opening a Pull Request

1. **Push your branch:**

```bash
git push origin feature/your-feature-name
```

2. **Open a Pull Request on GitHub:**

   - Use a clear, descriptive title
   - Reference related issues (e.g., "Fixes #123")
   - Describe what changes you made and why
   - Include any relevant context or screenshots

3. **PR Description Template:**

```markdown
## Description

Brief description of changes

## Related Issue

Closes #123

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing

- [ ] Tests added/updated
- [ ] All tests pass
- [ ] Manual testing completed

## Checklist

- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings generated
```

### Review Process

- Maintainers will review your PR
- Address feedback promptly
- Keep discussions focused and constructive
- Be open to suggestions and improvements

## Adding New Features

### Adding a New Connector

1. **Define connector type** in `proto/connectors/connectors.proto`:

   ```protobuf
   enum Connector {
     // ... existing connectors
     NEW_CONNECTOR = X;
   }
   ```

2. **Implement connector interface** in `core/runtime/connectors/newconnector.go`:

   ```go
   type NewConnector struct {
       // fields
   }

   func (c *NewConnector) Execute(ctx context.Context, query string, args ...interface{}) (*Result, error) {
       // implementation
   }
   ```

3. **Update factory** in `core/runtime/connectors/connector.go`:

   ```go
   case proto.Connector_NEW_CONNECTOR:
       return NewNewConnector(config), nil
   ```

4. **Regenerate code:**

   ```bash
   make generate
   ```

5. **Add tests** for the new connector

6. **Update documentation** with usage examples

### Adding a New Primitive Type

1. **Define type** in `proto/primitives/primitives.proto`:

   ```protobuf
   enum Primitive {
     // ... existing types
     NEW_TYPE = X;
   }
   ```

2. **Regenerate code:**

   ```bash
   make generate
   ```

3. **Add conversion logic** in `core/runtime/executor/utils/validator.go`

4. **Update OpenAPI mapping** in `core/runtime/handlers/openapi_handler.go`

5. **Add tests** for validation and conversion

### Adding a New Parser

1. **Create parser** in `core/parser/`:

   ```go
   func ParseNewFormat(data []byte) (*Config, error) {
       // implementation
   }
   ```

2. **Add file detection** in `core/cli/internal/loader.go`:

   ```go
   if strings.HasSuffix(filename, ".newformat") {
       return ParseNewFormat(data)
   }
   ```

3. **Add tests** for the parser

4. **Update documentation** with format specification

## Release Process

Releases are managed by maintainers. The release process includes:

1. **Version tagging:** Releases are tagged with semantic versioning (e.g., `v1.2.3`)
2. **Automated builds:** GitHub Actions builds binaries for multiple platforms
3. **Distribution:** Binaries are published to GitHub Releases
4. **Package publishing:** NPM and Homebrew packages are updated automatically

### Version Numbering

- **Major:** Breaking changes
- **Minor:** New features (backward compatible)
- **Patch:** Bug fixes (backward compatible)

## Getting Help

- **Documentation:** [hyperterse.mintlify.app](https://hyperterse.mintlify.app)
- **Issues:** [GitHub Issues](https://github.com/hyperterse/hyperterse/issues)
- **Discussions:** [GitHub Discussions](https://github.com/hyperterse/hyperterse/discussions)

## Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [Protocol Buffers Guide](https://protobuf.dev/)
- [Effective Go](https://go.dev/doc/effective_go)

---

Thank you for contributing to Hyperterse! 🚀
