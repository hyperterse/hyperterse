# Hyperterse Builder Skill

A skills.sh-compatible skill that generates complete Hyperterse MCP server projects from natural language descriptions.

## What This Does

Tell an AI assistant what API you want, and it generates a production-ready Hyperterse project with:

- Database adapters (PostgreSQL, MySQL, MongoDB, Redis)
- MCP tools with CRUD operations
- TypeScript handlers and mappers
- Authentication configuration

## Installation

This skill is part of the Hyperterse repository. To use it:

1. Clone the repo or reference the skill path
2. The skill is located at `skills/hyperterse-builder/`

For Claude Code users, reference the skill in your configuration.

## Usage

Once installed, just describe what you want:

```
"Build me a CRM API with contacts and companies"
```

```
"Create an e-commerce backend with products, orders, and inventory"
```

```
"I need a blog API with posts, authors, and comments"
```

The AI will generate a complete project structure you can run with:

```bash
hyperterse start --watch
```

## Project Structure

Generated projects follow this structure:

```
my-api/
├── .hyperterse              # Root configuration
├── app/
│   ├── adapters/            # Database connections
│   │   └── main-db.terse
│   └── tools/               # MCP tools
│       ├── users/
│       │   ├── get/config.terse
│       │   ├── list/config.terse
│       │   └── create/config.terse
│       └── ...
```

## Documentation

- `SKILL.md` - Main skill instructions for AI
- `references/SYNTAX.md` - Complete .terse file syntax
- `references/ADAPTERS.md` - Database configurations
- `references/TOOLS.md` - Tool patterns
- `references/HANDLERS.md` - TypeScript handlers
- `references/AUTH.md` - Authentication plugins
- `references/EXAMPLES.md` - Full project examples

## Prerequisites

Install Hyperterse:

```bash
curl -fsSL https://hyperterse.com/install | bash
```

## Quick Start

1. Install the skill
2. Ask AI to build your API
3. Set environment variables (DATABASE_URL, etc.)
4. Run `hyperterse start --watch`
5. Test at `http://localhost:8080/mcp`

## License

Apache-2.0
