# LinkedDoc Build Tool

A command-line tool for parsing, validating, and indexing LinkedDoc+TTL documentation headers in Go source files.

## Installation

```bash
# Build the tool
go build -o linkedoc_build tools/*.go

# Or run directly
go run tools/*.go [flags]
```

## Usage

### Validate LinkedDoc Headers

Validates all LinkedDoc headers and checks for broken links:

```bash
go run tools/*.go --validate

# Validate specific directory
go run tools/*.go --validate --path location-tracker/handlers

# Verbose output
go run tools/*.go --validate --verbose
```

**Output:**
```
✓ All LinkedDoc headers are valid
  - 4 modules validated
  - 0 broken links
  - 0 schema violations
```

### Generate JSON Index

Generates an AI-optimized JSON index for code navigation:

```bash
go run tools/*.go --generate-index

# Custom output path
go run tools/*.go --generate-index --output docs/my_index.json

# Verbose output
go run tools/*.go --generate-index --verbose
```

**Output:**
```
✓ JSON index generated: docs/linkedoc_index.json
  - 4 modules indexed
```

### Incremental Builds

Only process files that have changed since last build:

```bash
go run tools/*.go --validate --incremental
go run tools/*.go --generate-index --incremental
```

Uses MD5 file hashing and caches results in `.linkedoc_cache/`.

### Combined Operations

Validate and generate index in one command:

```bash
go run tools/*.go --validate --generate-index
```

## Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--validate` | Validate LinkedDoc headers and links | false |
| `--generate-index` | Generate JSON index | false |
| `--path` | Directory to scan | `.` (current) |
| `--output` | Output path for JSON index | `docs/linkedoc_index.json` |
| `--incremental` | Only process changed files | false |
| `--verbose` | Verbose output | false |
| `--version` | Show version | - |

## Validation Rules

### Header Structure
- ✅ Module name must be present
- ✅ Description must be present
- ⚠️  Description should be under 150 characters
- ⚠️  Should have at least one export
- ⚠️  Should have at least one tag

### Links
- ✅ All linked files must exist
- ✅ Relative paths must resolve correctly
- ⚠️  Link relationships should have descriptions

### Tags
- ⚠️  Tags should be from defined taxonomy
- Known tags: `http`, `storage`, `api-client`, `business-logic`, `data-types`, `middleware`, `auth`, `validation`, `location`, `errors`, `tips`, `commercial`, `social`, `cryptogram`, `dynamodb`, `cache`, `rate-limiting`, `websocket`, `rdf`, `solid`

### RDF Metadata
- ✅ Must have `@prefix code:` declaration
- ✅ Must have `code:name` property
- ✅ Must have `code:description` property
- ⚠️  RDF module name should match header

**Legend:**
- ✅ Error (validation fails)
- ⚠️  Warning (validation passes with warning)

## JSON Index Format

Generated index includes:

```json
{
  "_metadata": {
    "generated_at": "2025-11-13T12:00:00Z",
    "module_count": 42,
    "total_loc": 5305,
    "avg_loc": 126,
    "modules_with_rdf": 42
  },
  "modules": {
    "handlers/location.go": {
      "description": "HTTP handlers for location tracking",
      "file_path": "location-tracker/handlers/location.go",
      "links_to": [
        {
          "name": "types/location",
          "path": "../types/location.go",
          "relationship": "Location data structures"
        }
      ],
      "exports": ["HandleLocation", "HandleLocationByID"],
      "tags": ["http", "location", "api"],
      "lines_of_code": 150,
      "last_modified": "2025-11-13T12:00:00Z",
      "has_rdf": true
    }
  }
}
```

## Caching

Incremental builds use file hashing to detect changes:

- Cache directory: `.linkedoc_cache/`
- Cache file: `.linkedoc_cache/file_hashes.txt`
- Format: `<filepath>\t<md5_hash>`

Add to `.gitignore`:
```
.linkedoc_cache/
```

## Pre-commit Hook

Add validation to your pre-commit hook:

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Validating LinkedDoc headers..."
go run tools/*.go --validate

if [ $? -ne 0 ]; then
    echo "LinkedDoc validation failed. Please fix errors before committing."
    exit 1
fi
```

## CI/CD Integration

### GitHub Actions

```yaml
jobs:
  linkedoc-validation:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Validate LinkedDoc
        run: go run tools/*.go --validate
      - name: Generate Index
        run: go run tools/*.go --generate-index
      - name: Upload Index
        uses: actions/upload-artifact@v3
        with:
          name: linkedoc-index
          path: docs/linkedoc_index.json
```

## Module Structure

```
tools/
├── linkedoc_build.go    # Main entry point with CLI
├── parser.go            # LinkedDoc header parser
├── validator.go         # Validation logic
├── index_generator.go   # JSON index generator
└── README.md            # This file
```

## Future Enhancements

- [ ] Circular dependency detection
- [ ] Dependency graph visualization (Mermaid, GraphViz)
- [ ] Tag-based index generation
- [ ] SPARQL query support
- [ ] Export metrics (test coverage, complexity)
- [ ] LSP integration for IDE support

## Examples

### Validate entire project
```bash
go run tools/*.go --validate --verbose
```

### Validate and generate index for specific package
```bash
go run tools/*.go --validate --generate-index --path location-tracker/handlers
```

### Incremental build (fast)
```bash
go run tools/*.go --validate --generate-index --incremental
```

## Troubleshooting

**No LinkedDoc headers found**
- Ensure files have LinkedDoc headers in comments
- Check that path is correct
- Tool only scans `.go` files (not `_test.go`)

**Broken links**
- Check that relative paths are correct
- Paths are relative to the file containing the header
- Use `--verbose` to see which links are being validated

**Unknown tags**
- Add custom tags to `validator.go` `knownTags` map
- Or update tag taxonomy in schema

## Related

- [LinkedDoc Schema](../linkedoc_schema.ttl)
- [Documentation](../docs/linkedoc/)
- [Issue #74](https://github.com/justin4957/ec2-test-apps/issues/74)
- [CLAUDE.md](../CLAUDE.md)
