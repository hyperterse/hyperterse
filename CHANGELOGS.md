# v2.5.0

Timestamp: 2026-04-21

**A2A-compatible agents**

This release aligns declarative agents with the **[A2A (Agent2Agent) protocol](https://github.com/a2aproject/a2a-go)**: new wire/executor paths, registry and HTTP surface updates, and broad test coverage. **Older, non-A2A agent shapes are no longer supported**—upgrade agent configs and clients to the new endpoints and behavior.

Distribution manifests are bumped to **v2.5.0**. Demo apps gain example agents (`demo-concierge`, `notes-analyst`, `weather-guide`) and README updates; agent docs (overview, quickstart, runtime API, model providers) and the agent JSON schema / generator are refreshed. `hyperterse init` output is updated for the new agent endpoint layout.

## ✨ Features

- **A2A runtime** — New A2A wiring and executor (`a2a_wire.go`, `a2a_executor.go`) integrated with the agent registry, server, and endpoints; extensive unit and runtime tests.
- **Dependencies** — Adds `github.com/a2aproject/a2a-go` v0.3.3 and `github.com/google/go-cmp` v0.7.0; dependency graph cleanup (e.g. drops some prior indirects such as GCP OTel detector and `gorilla/mux` where no longer needed).

## ⚠️ Breaking changes

- **Agent compatibility** — Compatibility with the previous (pre-A2A) agent API has been removed; migrate to A2A-oriented configuration and HTTP usage as documented.

---

# v2.4.0

Timestamp: 2026-04-17

🎉 **Declarative Agents**

This release adds first-class support for **declarative AI agents** in the Hyperterse engine: discover agents from the project layout, compile them alongside tools, and run OpenAI-compatible and Google model providers with configurable tool access and validation. Distribution manifests are updated to **v2.4.0**, dependencies are refreshed (including security-related bumps), and the README now includes an MCP **spec compliance** summary table.

**Backward compatible** for projects that do not define agents; add `app/agents/` and related config when you are ready.

## ✨ Features

### Declarative agents (engine)

- **Discovery and compilation** — Agents are discovered and compiled with the framework, with inherited tool access and validation of model definitions and tool policies.
- **Model providers** — Runtime support for OpenAI-compatible APIs and Google-backed providers (including Gemini and Vertex AI), with environment-variable substitution for secrets and model options (for example `{{ env.VAR_NAME }}`).
- **Tool access** — Configurable tool access for agents, including sensible defaults and inheritance when omitted.
- **Observability** — Improved logging for agent endpoints, requests, and model calls for easier operations and debugging.

## 🔧 Improvements

- **Dependencies** — Upgraded vulnerable and stale dependencies (including `esbuild`, MCP `go-sdk`, OpenTelemetry, and related indirect packages).

---

# v2.3.0

Timestamp: 2026-03-17

🎉 **Full MCP Spec + SQLite**

This release completes MCP specification compliance and adds SQLite as a first-class connector. Hyperterse servers can now expose **tools**, **resources**, and **prompts** in one place — the three pillars of the MCP protocol. Add SQLite (local or Turso/libSQL) for lightweight backends, side‑project demos, or edge-deployed AI apps.

**What you can build now:** AI agents that read release notes, summarize incidents, or query SQLite — with prompt templates and static context discovered automatically. Zero extra setup beyond config.

**Backward compatible** — no migration required. Existing configs work as-is; add `app/prompts/` and `app/resources/` when you're ready.

## ✨ Features

### MCP Resources & Prompts (Full Spec Compliance)

Hyperterse now implements the complete MCP resource and prompt surface. AI clients can discover and consume static context alongside callable tools.

- **Resources** — Expose read-only content via `resources/list`, `resources/read`, and `resources/templates/list`. Support concrete resources (fixed URI) and resource templates (parameterized `uri_template` with `{{ id }}` placeholders). Content can be inline (`text`, `text_template`) or file-backed (`file`, `file_template`).
- **Prompts** — Define reusable prompt templates in `app/prompts/**/*.terse` with argument interpolation, completion hints, and multi-message scaffolding. Exposed through `prompts/list` and `prompts/get`; clients get rendered messages with `{{ argument }}` replaced.
- **Completion API** — `completion/complete` supports both prompts and resource templates, so clients get typeahead and validation for template arguments when you declare `completion` values.
- **Subscriptions & notifications** — `resources/subscribe` / `resources/unsubscribe`; `notifications/resources/updated`, `notifications/prompts/list_changed` for live updates after model reload.
- **Discovery conventions** — Config in `app/prompts/` and `app/resources/`; declarative YAML with MIME types, descriptions, and optional argument schemas.

### SQLite Connector

A new database connector for SQLite: local files, in-memory, or remote libSQL/Turso.

- **Local** — `file:./app.db`, `:memory:`, or absolute paths for embedded and dev workloads.
- **Remote (libSQL / Turso)** — `libsql://` or `https://` for hosted Turso; `http://` for self-hosted libSQL. Auth via `authToken`, `auth_token`, or `jwt` query params; TLS control via `?tls=0|1`.
- **Adapter config** — Uses `app/adapters/*.terse` like PostgreSQL and MySQL; same patterns for connection strings, env substitution, and options.
- **Execution** — Parameterized queries, transaction handling, OpenTelemetry tracing, and observability metrics. Compatible with the existing tool executor and framework.

---

# v2.2.1

Timestamp: 2026-03-12

## Bug fixes

- **Init command directory path** — Fixed directory references from `.agent` to `.agents` when scaffolding for Hyperterse integration with AI agents

## 🔧 Improvements

- **Agent skills documentation** — Updated the documentation link in SKILL.md to point to the correct resource
- **Init output messaging** — Revised the output message for running the Hyperterse command after scaffolding

---

# v2.2.0

Timestamp: 2026-03-10

## ✨ Features

- **Smarter MCP search relevance ranking** — Search scoring now adapts to the metadata each tool actually has, improving hit quality across SQL-backed and handler-only tools

## 🔧 Improvements

- **Handler-only tool relevance** — Handler-only tools no longer receive statement-based relevance from synthetic placeholder statements (for example, `SELECT 1`)
- **Conversational query matching** — Token scoring now applies a soft miss penalty so intent-heavy natural language prompts rank more accurately
- **Search ranking test coverage** — Added targeted tests for handler-only scoring, conversational queries, and active-weight normalization

## Bug fixes

- **Search payload shape update** — MCP search results no longer include `statement`; clients should rely on `name`, `description`, `relevance_score`, and `inputs`

---

# v2.1.0

Timestamp: 2026-03-03

## ✨ Features

- **Init command folder support** — `hyperterse init` now accepts an optional folder name as argument; scaffolds into `<folder>/.hyperterse` by default
- **Script-backed tool scaffolding** — Init command generates script-backed tool structure instead of adapter-based config; includes greeting handler example
- **Agent skills documentation** — New agent skills directory with documentation for Hyperterse integration with AI agents

## 🔧 Improvements

- **Init output path logic** — Output file defaults to `<folder>/.hyperterse` when folder is specified; prompts for folder name when not provided
- **Documentation updates** — Revised quickstart and CLI reference to reflect script-backed tools and new project structure

---

# v2.0.0

Timestamp: 2026-02-24

🎉 **Hyperterse v2.0.0 — Tool-First MCP Framework**

Hyperterse v2 is a breaking redesign. The framework shifts from a single-file query gateway to a compiled, filesystem-based MCP framework. Define tools in declarative config, add optional TypeScript handlers, and serve them as a standards-compliant MCP server over Streamable HTTP.

## ✨ Features

- **Filesystem-based tool discovery** — Each directory under `app/tools/` maps to one MCP tool; no manual registration
- **Declarative tool config** — Root config + `app/adapters/*.terse` + `app/tools/*/config.terse` for clear separation of concerns
- **Embedded scripting** — TypeScript handlers and input/output transforms for logic that config alone cannot express; scripts bundled at compile time
- **Per-tool authentication** — Pluggable auth per tool: `allow_all`, `api_key`, or custom plugins
- **Build and serve pipeline** — `hyperterse build` compiles config and scripts into a deployable artifact; `hyperterse serve` runs from the artifact
- **Search tool** — Built-in MCP search tool with configurable result limit and statement support for tool discovery
- **Graceful shutdown** — BaseContext-based shutdown for in-flight handlers to complete before exit

## 🔧 Improvements

- **MCP-only transport** — Tools exposed exclusively via MCP Streamable HTTP (`/mcp`); `/query/{name}` removed
- **Per-tool caching** — Global cache config plus per-tool override for TTL and behavior
- **OpenTelemetry observability** — Distributed tracing, metrics, and structured logging out of the box
- **CLI commands** — `start` (with optional `--watch`), `build`, `serve`, `validate`, `init`; `run` replaced by `start`
- **Project convention** — `.hyperterse` root config; `hyperterse start` for development

## ⚠️ Breaking Changes

| Change                    | Impact                           | Action                                |
| ------------------------- | -------------------------------- | ------------------------------------- |
| Inline `adapters` removed | Config will not parse            | Extract to `app/adapters/*.terse`     |
| Inline `queries` removed  | Config will not parse            | Extract to `app/tools/*/config.terse` |
| `/query/{name}` removed   | HTTP clients break               | Migrate to MCP `tools/call`           |
| Auth now per-tool         | Tools unauthenticated by default | Add `auth` blocks                     |
| DSL parser deprecated     | Text-format blocks unsupported   | Convert to YAML                       |
| Build step introduced     | Direct interpretation replaced   | Add `build` + `serve` to pipeline     |

See the [v1 to v2 migration guide](https://docs.hyperterse.com/migration/v1-to-v2) for step-by-step migration instructions.

---

# v1.4.0

Timestamp: 2026-02-12

## ✨ Features

- **Config validation command** — Added a dedicated CLI command to validate `.terse` configuration files before runtime
- **Query execution caching** — Added executor-level query caching to reduce repeated work and improve runtime performance

## 🔧 Improvements

- **CLI error handling and logging** — Improved command-level error handling and log output for clearer diagnostics
- **Observability setup** — Added analytics and observability wiring to improve operational visibility

---

# v1.3.0

Timestamp: 2026-02-08

## ✨ Features

- **MongoDB connector** — New database connector for MongoDB, including BSON-to-JSON parsing and support for MongoDB connection strings and queries

## 🐛 Bug Fixes

- **MySQL connection strings** — Fixed URL-to-DSN conversion for MySQL connection strings
- **MongoDB JSON responses** — Simplified and corrected JSON serialization of MongoDB query results

## 🔧 Improvements

- **`.env` loading** — Environment files are now loaded earlier and more reliably during startup

## ⚠️ Schema Changes

- Configuration schema updated to support `mongodb` as a connector type in adapters

---

# v1.2.1

Timestamp: 2026-02-02

## 🐛 Bug Fixes

- **NPM distribution** — Fixed npm-installed CLI not forwarding command-line arguments to the binary; commands like `hyperterse run -f config.terse` now work correctly when installed via `npm install -g hyperterse`

---

# v1.2.0

Timestamp: 2026-02-01

## ✨ Features & Enhancements

- **Heartbeat endpoint** — New `GET /heartbeat` utility route for health checks and load balancer probes
- **Automatic .env loading** — Runtime automatically loads `.env`, `.env.development`, and `.env.local` when present (no manual sourcing required)

## 🔧 Improvements

- **Connector error handling** — Connectors now report errors once and exit the program instead of repeating error output

---

# v1.1.1

Timestamp: 2026-01-29 16:28:28 UTC

🐛 **Bug Fixes & CLI Improvements**

This release fixes several bugs related to logging, validation, and CLI behavior.

## 🐛 Bug Fixes

- **Logger Log Levels** — Fixed issue where logger was not respecting log levels in some scenarios, ensuring proper log filtering
- **YAML Validation** — Fixed validation issue for non-string default values in configuration files, improving type handling

## 🔧 Improvements

- **Version Flag** — The `-v` flag now prints the version information instead of being a shorthand for verbose
- **Verbose Flag** — Removed shorthand for verbose flag, requiring explicit `--verbose` usage for clarity

---

# v1.1.0

Timestamp: 2026-01-29 04:32:32 UTC

🔧 **Export Enhancements & Logging Improvements**

This release enhances the export command with better control and directory management, introduces structured logging throughout the application, and includes several bug fixes and documentation updates.

## ✨ Features & Enhancements

### Export Command Improvements

- **Enhanced Export Configuration** — Added finer-grained control over export behavior with optional `export` configuration block
- **Mandatory Name Field** — Configuration now requires a `name` field that follows naming conventions for better organization
- **Clean Directory Option** — New `--clean-dir` flag allows cleaning the output directory before exporting
- **Automatic Directory Creation** — Export command now automatically creates output directories if they don't exist
- **Improved Error Handling** — Enhanced error handling and logging throughout the export process

### Logging System

- **Structured Logging** — Implemented structured, filterable logging throughout the entire application
- **Enhanced Log Levels** — Improved log level management and filtering capabilities
- **Better Debugging** — More detailed logging for connectors, executors, and handlers

## 🐛 Bug Fixes

- **Help Output Fix** — Fixed issue where help text was printed unnecessarily on panic, improving error handling
- **Export Directory Creation** — Fixed issue where export command failed when output directory didn't exist

---

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

- **Documentation**: [Full Documentation](https://hyperterse.mintlify.app)
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

- **Declarative Format** — Simple, readable `.terse` configuration files
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
- [Full Documentation](https://hyperterse.mintlify.app)

> _This is the first public alpha release of Hyperterse. We welcome feedback and contributions!_
