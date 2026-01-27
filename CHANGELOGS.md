# v1.0.0

Timestamp: 2026-01-28 UTC

🎉 **Hyperterse v1.0.0 — Production Ready!**

We're thrilled to announce the first stable release of Hyperterse! After months of development, testing, and community feedback through our alpha and beta releases, Hyperterse v1.0.0 is now production-ready and available for everyone.

This release represents a major milestone in making database queries accessible as RESTful APIs and MCP tools. Whether you're building AI applications, microservices, or modern APIs, Hyperterse provides a powerful, flexible foundation for transforming your database queries into production-ready endpoints.

## 🚀 What's New in v1.0.0

### Production-Ready Features

- **Stable API** — All APIs are now stable and ready for production use
- **Multi-Package Manager Support** — Install via NPM (`npm install -g hyperterse`) or Homebrew (`brew install hyperterse/tap/hyperterse`)
- **Enterprise-Grade Security** — Runtime environment variable substitution for secure configuration management
- **Model Context Protocol** — Full Streamable HTTP transport support for seamless AI integrations

### Core Capabilities

- **Automatic Endpoint Generation** — Transform database queries into RESTful API endpoints instantly
- **Multi-Database Support** — PostgreSQL, MySQL, and Redis connectors with automatic connection management
- **MCP Integration** — Expose queries as MCP tools via JSON-RPC 2.0 for AI/LLM applications
- **OpenAPI Documentation** — Auto-generated OpenAPI 3.0 specifications for all endpoints
- **Type-Safe Configuration** — JSON schema validation with IDE support for `.terse` configuration files
- **Developer Experience** — Hot reload development mode, upgrade command, and init templates

### Installation & Getting Started

Get started with Hyperterse in seconds:

```bash
# Via NPM
npm install -g hyperterse

# Via Homebrew
brew install hyperterse/tap/hyperterse

# Or download directly
curl -fsSL https://hyperterse.com/install | bash
```

Create your first configuration:

```bash
hyperterse init
hyperterse run -f config.terse
```

## 🙏 Thank You

A huge thank you to everyone who tested the alpha and beta releases, reported issues, and provided feedback. Your contributions have been invaluable in making Hyperterse production-ready.

## 📚 Resources

- **Documentation**: [Full Documentation](https://github.com/hyperterse/hyperterse/blob/main/HYPERTERSE.md)
- **GitHub**: [hyperterse/hyperterse](https://github.com/hyperterse/hyperterse)
- **Issues**: [Report Issues](https://github.com/hyperterse/hyperterse/issues)

---

# v1.0.0-beta.5

Timestamp: 2026-01-26 10:23:27 UTC

📦 **Multi-Package Manager Support & Documentation**

This release adds support for distributing Hyperterse through multiple package managers (NPM and Homebrew), improves the release workflow, and consolidates documentation into the main repository.

## ✨ Enhancements

### Package Manager Support

- **NPM Package** — New NPM package for easy installation via `npm install -g hyperterse`
- **Homebrew Tap** — Official Homebrew formula available at `hyperterse/tap` for macOS and Linux users
- **Automatic Binary Detection** — Both package managers automatically detect platform and architecture to download the correct binary

### Documentation

- **Consolidated Docs** — Moved documentation into the main repository for easier maintenance and contribution
- **Improved Styling** — Enhanced documentation with content updates and subtle design changes

> _This is a beta release of Hyperterse. We welcome feedback and contributions!_

---

# v1.0.0-beta.4

Timestamp: 2026-01-25 08:34:13 UTC

🔐 **Security & Configuration Enhancements**

This release introduces environment variable substitution, improves security by parsing environment variables at runtime, and enhances adapter flexibility with raw option passthrough.

## ✨ Features & Enhancements

### Environment Variable Substitution

- **Runtime Variable Support** — Added support for environment variable substitution in configuration files
- **Security Improvement** — Environment variables are parsed at runtime for better security and are not shipped in export scripts
- **Flexible Configuration** — Use environment variables in your `.terse` configuration files for sensitive values

### Adapter Improvements

- **Raw Options Passthrough** — Adapters now pass raw options directly to underlying connectors, providing more flexibility and control

### Response & Error Handling

- **Enhanced Response Structure** — Improved response structure and error handling in server routes for better API consistency
- **Better Error Messages** — More descriptive error responses for improved debugging experience

## 🔧 Improvements

### Validation & Schema

- **Flexible Naming** — Refactored validation patterns in parser and schema to allow names starting with any letter case
- **Removed UUID References** — Removed UUID references from validation and handler files, updating related comments and schema accordingly
- **Enhanced Type Descriptions** — Enhanced default value descriptions for various types to clarify requirements in JSON schema

> _This is a beta release of Hyperterse. We welcome feedback and contributions!_

---

# v1.0.0-beta.3

Timestamp: 2026-01-18 21:00:06 UTC

🚀 **MCP Protocol Enhancement: Streamable HTTP Support**

This release enhances the MCP (Model Context Protocol) implementation with full Streamable HTTP transport support, replacing the deprecated SSE-only transport and providing a more robust, standards-compliant interface for AI integrations.

## ✨ Features & Enhancements

### MCP Streamable HTTP Transport

- **Modern Transport Protocol** — Implemented Streamable HTTP transport for MCP protocol, replacing deprecated SSE-only transport
- **Dual-Method Support** — POST endpoint for client-initiated JSON-RPC messages, GET endpoint for server-initiated messages via SSE
- **Protocol Version Support** — Added `MCP-Protocol-Version` header support (defaults to `2025-03-26`, also supports legacy `2024-11-05`)
- **Session Management** — Implemented `Mcp-Session-Id` header for session tracking across requests
- **Flexible Response Format** — Server responds with JSON for standard requests or SSE stream when appropriate
- **CORS Support** — Added comprehensive CORS headers for cross-origin requests
- **OpenAPI Documentation** — Updated OpenAPI spec to document Streamable HTTP endpoints and headers

### Protocol Improvements

- **JSON-RPC 2.0 Compliance** — Enhanced JSON-RPC error handling with proper error codes and messages
- **Request Validation** — Improved request parsing and validation with proper error responses
- **Backward Compatibility** — Maintains support for legacy protocol versions while defaulting to latest

## 🔧 Improvements

- **Documentation** — Updated README and LLM documentation with Streamable HTTP examples and usage instructions
- **Error Handling** — Enhanced error responses to follow JSON-RPC 2.0 specification

> _This is a beta release of Hyperterse. We welcome feedback and contributions!_

---

# v1.0.0-beta.2

Timestamp: 2026-01-17 21:28:32 UTC

🐛 **Bug Fixes & Documentation Updates**

This release fixes critical issues with the upgrade command and completes the migration away from YAML references.

## 🐛 Bug Fixes

- **Upgrade Command** — Fixed upgrade command not working correctly, improving version detection and update functionality

## 🔧 Improvements

- **Documentation** — Updated README with improved examples and clearer instructions
- **Configuration References** — Removed all remaining YAML references throughout the codebase, completing the migration to `.terse` extension
- **Build Configuration** — Updated build scripts and configuration files to use `.terse` extension consistently

> _This is a beta release of Hyperterse. We welcome feedback and contributions!_

---

# v1.0.0-beta.1

Timestamp: 2026-01-17 19:57:23 UTC

🛠️ **Developer Tools & Configuration Improvements**

This release introduces new CLI commands for easier project setup and updates, along with improved configuration file handling and validation.

## ✨ Features & Enhancements

### Upgrade Command

- **Automatic Updates** — New `upgrade` command checks for and installs the latest version of hyperterse
- **Version Management** — Version is now baked into the binary for update checking
- **Major Version Control** — Upgrade within the same major version by default, or use `--major` to upgrade across major versions
- **Pre-release Support** — Use `--prerelease` flag to include pre-releases when checking for updates
- **Smart Detection** — Automatically detects current version from binary, git, or fallback methods

### Init Command

- **Quick Start** — New `init` command creates a new `.terse` configuration file with sample adapter and query
- **Template Generation** — Generates a complete, ready-to-use configuration template
- **Custom Output** — Specify output file with `-o` or `--output` flag (defaults to `config.terse`)

### Configuration File Format

- **New Extension** — Configuration files now use `.terse` extension instead of `.yaml`/`.yml`
- **JSON Schema** — Added JSON schema validation for `.terse` files (`schema/terse.schema.json`)
- **IDE Support** — VS Code associations for `.terse` files with schema validation
- **Schema Generation** — New script to generate JSON schema from proto definitions (`scripts/generate_schema/`)

### Installation

- **Installer Script** — Added standalone installer script (`install`) for easier installation
- **Version Selection** — Installer supports installing specific versions or latest release
- **Local Binary Installation** — Support for installing from local binary files
- **Path Management** — Automatic PATH configuration for shell environments

## 🔧 Improvements

- **Build System** — Enhanced build scripts to support version baking and schema generation
- **Documentation** — Updated documentation and examples to use `.terse` extension
- **Export Command** — Updated export command to use `.terse` extension
- **Versioning** — Fixed versioning script to not bump patch version when creating prereleases

> _This is a beta release of Hyperterse. We welcome feedback and contributions!_

---

# v1.0.0-alpha.2

Timestamp: 2026-01-16 16:34:06 UTC

🚀 **Portable Deployment & Runtime Enhancements**

This release introduces the export command for creating portable deployment scripts and adds support for running from inline configuration strings.

## ✨ Features & Enhancements

### Export Command

- **Portable Scripts** — New `export` command generates self-contained bash scripts with embedded configuration and binary
- **Zero Dependencies** — Generated scripts can run in any environment without requiring hyperterse to be installed
- **Simple Deployment** — Export your config and binary as a single executable script: `hyperterse export -f config.yaml -o dist`

### Runtime Configuration

- **Inline Configuration** — Added `--source`/`-s` flag to run command for providing configuration as a string instead of a file
- **Flexible Input** — Run hyperterse with `hyperterse run --source "yaml: content"` for dynamic configuration

### Query Execution

- **Execution Context** — Implemented execution context for better query management
- **Parallelization** — Added support for parallel query execution

## 🔧 Improvements

- **Type System** — Modernized internal type system for better maintainability

> _This is an alpha release of Hyperterse. We welcome feedback and contributions!_

---

# v1.0.0-alpha.1

Timestamp: 2026-01-10 11:20:55 UTC

🔄 **Developer Experience Improvements**

This release focuses on improving the development workflow with graceful server reloads and better signal handling.

## ✨ Features & Enhancements

### Dev Server

- **Graceful Reloads** — Dev server now supports graceful reloads on config changes, allowing the old server to continue running until the new one is ready
- **Graceful Shutdown** — Dev command now handles exit signals for graceful shutdown

### Documentation

- **LLM Types** — Updated `generate llms` command handler to generate proper types in docs

## 🐛 Bug Fixes

- **Enum Validation** — Fixed runtime input validator parsing of enums

## 🔧 Improvements

- **Build Consistency** — Refactored build command in `.air.toml` to use Makefile for improved consistency and clarity
- **Air Integration** — Updated run command when using air

## 📝 Maintenance

- Ignore system files
- Updated project documentation

> _This is an alpha release of Hyperterse. We welcome feedback and contributions!_

---

# v1.0.0-alpha.0

Timestamp: 2026-01-10 11:20:55 UTC

🎉 **First Alpha Release**

Hyperterse is a high-performance runtime server that transforms database queries into RESTful API endpoints and MCP (Model Context Protocol) tools. Define queries in YAML, and Hyperterse automatically generates endpoints with full OpenAPI documentation.

## ✨ Features

### Core Runtime

- **Automatic Endpoint Generation** — Each query becomes its own REST endpoint at `POST /query/{query-name}`
- **Multi-Database Support** — PostgreSQL, MySQL, and Redis connectors out of the box
- **Input Validation** — Automatic type checking and validation for all query inputs
- **Template Variables** — Use `{{ inputs.parameterName }}` syntax in SQL statements

### Supported Data Types

- `string`, `int`, `float`, `boolean`, `uuid`, `datetime`
- Optional inputs with default values
- Required field enforcement

### AI & LLM Integration

- **MCP Protocol Support** — Expose queries as MCP tools via JSON-RPC 2.0 endpoint (`POST /mcp`)
- **LLM Documentation** — Auto-generated markdown documentation at `GET /llms.txt`
- **Agent Skills Export** — Generate Agent Skills compatible archives with `hyperterse generate skills`

### Documentation

- **OpenAPI 3.0 Specification** — Complete API documentation at `GET /docs`
- **Request/Response Schemas** — Auto-generated from query definitions
- **Example Values** — Included in OpenAPI spec

### Configuration

- **YAML Format** — Simple, readable configuration files (`.yaml`, `.yml`)
- **Comprehensive Validation** — Catches configuration errors before startup

### CLI Commands

- `hyperterse -f config.yaml` — Start the runtime server
- `hyperterse run -f config.yaml` — Start with explicit run command
- `hyperterse dev -f config.yaml` — Development mode with hot reload
- `hyperterse generate llms -f config.yaml` — Generate llms.txt documentation
- `hyperterse generate skills -f config.yaml` — Generate Agent Skills archive

### Security

- **Connection String Protection** — Never exposed in API responses or documentation
- **SQL Injection Prevention** — Proper escaping and type validation
- **Error Message Sanitization** — No sensitive database information leaked

## 📦 Installation

### Quick install

```bash
curl -fsSL https://github.com/hyperterse/hyperterse/releases/latest/download/install.sh | bash
```

Or download binary directly for your platform below.

### Supported platforms

| Platform | Architecture                         |
| -------- | ------------------------------------ |
| Linux    | amd64, arm64, arm                    |
| macOS    | amd64 (Intel), arm64 (Apple Silicon) |
| Windows  | amd64, arm64                         |

### ⚠️ Known Limitations

- This is an alpha release — APIs may change in future versions
- No built-in authentication/authorization (use a reverse proxy)
- No connection pooling configuration exposed yet

## 📚 Documentation

- [README](https://github.com/hyperterse/hyperterse#readme)
- [Full Documentation](https://github.com/hyperterse/hyperterse/blob/main/HYPERTERSE.md)

> _This is the first public alpha release of Hyperterse. We welcome feedback and contributions!_
