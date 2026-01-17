# JSON Schema Generation

This script generates a JSON schema for `.terse` configuration files based on the protobuf definitions.

## Overview

The script reads the `connectors.proto` and `primitives.proto` files to extract enum values and generates a JSON Schema (Draft 07) that validates the structure of `.terse` configuration files.

## Usage

The script is automatically run as part of `make generate` or `./scripts/generate-proto.sh`. It can also be run manually:

```bash
go run scripts/generate_schema/script.go proto/connectors/connectors.proto proto/primitives/primitives.proto
```

## Output

The generated schema is written to `schema/terse.schema.json` and includes:

- **Server configuration**: Optional port and log_level settings
- **Adapters**: Map of adapter configurations with connector types, connection strings, and options
- **Queries**: Map of query definitions with use, description, statement, inputs, and data fields

## Schema Features

- Validates connector types against enum values from `connectors.proto`
- Validates primitive types against enum values from `primitives.proto`
- Enforces required fields for adapters and queries
- Supports both string and array formats for the `use` field in queries
- Includes descriptions for all fields

## Integration

The schema generation is integrated into the proto generation pipeline:

1. Protobuf files are compiled
2. Go types are generated
3. **JSON schema is generated** ← This script
4. Build completes

This ensures the schema stays in sync with the protobuf definitions.
