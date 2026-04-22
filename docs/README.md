# Hyperterse Docs (Mintlify)

Documentation for **Hyperterse**, the **agentic server framework**—declarative agents (A2A), MCP tools, prompts, resources, adapters, and runtime guides—in Mintlify-native source.

## Structure

- `docs.json` — Mintlify site configuration and navigation.
- `assets/` — Logo, favicon, and diagrams.
- `*.mdx` — Top-level documentation pages.
- Section folders:
  - `agents/` — Declarative agents, A2A runtime, model providers, tool access.
  - `concepts/` — Shared concepts; **Tools**, **Resources**, and **Prompts** each have dedicated guides under this folder (see sidebar groups in `docs.json`).
  - `runtime/` — MCP transport, A2A transport, execution pipeline, caching, observability.
  - `reference/` — CLI and configuration schema references.
  - `databases/` — Connector guides.
  - `deployment/` — Deployment methods and providers.
  - `security/` — Input safety and production hardening.
  - `migration/` — Version upgrades.

Navigation is grouped by **Agents**, **Tools**, **Resources**, and **Prompts**, plus **Foundation** (project structure and authentication), **Runtime**, **Databases**, **Deployment**, and **Security**.

## Local development

From this directory:

```bash
bun install
bun run dev
```

This starts the Mintlify local server (hot reload enabled).

## Validate (static checks)

```bash
bunx mintlify validate
```

## One-off (without install)

If you want to run without creating local `node_modules` first:

```bash
bunx mintlify dev
```

## Editing guidelines

- Keep docs aligned with implemented behavior in `core/`.
- Avoid documenting planned endpoints/commands as if they already exist.
- For config shape changes, update:
  - schema files in `schema/`
  - this docs content
  - `docs/reference/configuration-schemas.mdx`
