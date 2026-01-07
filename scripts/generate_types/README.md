# Code Generation Scripts

## generate_types.go

This script generates `pkg/types/connectors.go` and `pkg/types/primitives.go` from the enum definitions in `proto/hyperterse.proto`.

### Usage

```bash
go run scripts/generate_types/script.go proto/hyperterse.proto
```

### What it does

1. Parses the `Connector` and `Primitive` enums from the proto file
2. Generates Go helper functions and constants for both enums
3. Creates conversion functions between string values and proto enum values

### When to run

Run this script whenever you:

- Add new connector types to the `Connector` enum in the proto file
- Add new primitive types to the `Primitive` enum in the proto file
- Modify enum values in the proto file

The generated files are marked with `// Code generated` comments and should not be edited manually.
