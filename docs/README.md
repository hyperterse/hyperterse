# Hyperterse Docs (Mintlify)

This directory now contains Mintlify-native docs source.

## Structure

- `docs.json` - Mintlify site configuration and navigation.
- `assets/` - logo and favicon assets.
- `*.mdx` - documentation pages.
- section folders:
  - `getting-started/`
  - `concepts/`
  - `runtime/`
  - `reference/`
  - `security/`
  - `deployment/`
  - `migration/`

## Local development

From this directory:

```bash
bun install
bun run dev
```

This starts the Mintlify local server (hot reload enabled).

## Build

```bash
bun run build
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
