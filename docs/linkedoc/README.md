# LinkedDoc+TTL Documentation System

## Overview

LinkedDoc+TTL is a hybrid documentation system that combines:
- **Human-readable** Markdown navigation links
- **Machine-readable** RDF/Turtle semantic metadata
- **AI-optimized** JSON index for efficient code navigation

This system enables both developers and AI systems to efficiently navigate large, modular codebases.

## Quick Start

### 1. Add LinkedDoc Header to Your Module

Every Go module should start with a LinkedDoc header:

```go
/*
# Module: handlers/location.go
HTTP handlers for location tracking endpoints.

## Linked Modules
- [types/location](../types/location.go): Location data structures
- [storage/dynamodb](../storage/dynamodb.go): Location persistence

## Tags
http, location, tracking, api

## Exports
HandleLocation, HandleLocationByID

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "handlers/location.go" ;
    code:description "HTTP handlers for location tracking" ;
    code:dependsOn <../types/location.go>, <../storage/dynamodb.go> ;
    code:exports :HandleLocation, :HandleLocationByID ;
    code:tags "http", "location", "api" .
<!-- End LinkedDoc RDF -->
*/
package handlers
```

### 2. Validate Links

```bash
go run tools/linkedoc_build.go --validate
```

### 3. Generate JSON Index

```bash
go run tools/linkedoc_build.go --generate-index
```

## Schema Reference

See [`linkedoc_schema.ttl`](../../linkedoc_schema.ttl) for the complete RDF ontology.

### Core Classes

- **`code:Module`** - A source code file
- **`code:Package`** - A collection of related modules
- **`code:Function`** - An exported function
- **`code:Type`** - An exported data type
- **`code:Handler`** - An HTTP handler function
- **`code:Service`** - A business logic service
- **`code:Client`** - An external API client

### Core Properties

- **`code:name`** - Module name with package path
- **`code:description`** - Human-readable purpose
- **`code:dependsOn`** - Direct dependency
- **`code:integratesWith`** - Integration relationship
- **`code:exports`** - Exported symbols
- **`code:tags`** - Semantic categorization tags

## Tag Taxonomy

### Functional Categories

- `http` - HTTP handlers/endpoints
- `storage` - Data persistence
- `api-client` - External API integration
- `business-logic` - Core business rules
- `data-types` - Data structures
- `middleware` - HTTP middleware
- `auth` - Authentication/authorization
- `validation` - Input validation

### Domain Categories

- `location` - Location tracking
- `errors` - Error logging
- `tips` - Anonymous tips
- `commercial` - Commercial real estate
- `social` - Social sharing
- `cryptogram` - Cryptogram puzzles

### Technical Categories

- `dynamodb` - DynamoDB operations
- `cache` - Caching logic
- `rate-limiting` - Rate limiting
- `websocket` - WebSocket connections
- `rdf` - RDF operations
- `solid` - Solid Protocol

## File Structure

```
project/
├── linkedoc_schema.ttl        # RDF ontology definition
├── tools/
│   ├── linkedoc_build.go      # Parser and validator
│   ├── parser.go              # LinkedDoc header parser
│   ├── validator.go           # Link validator
│   └── index_generator.go     # JSON index generator
├── docs/
│   ├── linkedoc/
│   │   ├── README.md          # This file
│   │   ├── schema.md          # Schema documentation
│   │   ├── best-practices.md  # Writing guidelines
│   │   └── examples/          # Example modules
│   └── linkedoc_index.json    # Generated AI index
└── <packages>/
    └── *.go (with LinkedDoc headers)
```

## Best Practices

### 1. Keep Descriptions Concise

✅ Good:
```
HTTP handlers for location tracking endpoints.
```

❌ Too verbose:
```
This module contains a collection of HTTP handler functions that are
responsible for processing incoming HTTP requests related to location
tracking functionality, including both creating new location records
and retrieving existing location data from the database.
```

### 2. Link All Direct Dependencies

```go
## Linked Modules
- [types/location](../types/location.go): Location data structures
- [storage/dynamodb](../storage/dynamodb.go): Persistence layer
- [services/business_search](../services/business_search.go): Business logic
```

### 3. Use Consistent Tags

Always use tags from the defined taxonomy. Don't invent new tags without updating the schema.

### 4. Update Links When Refactoring

When you move or rename a module, update all LinkedDoc headers that reference it.

### 5. Validate Before Committing

```bash
go run tools/linkedoc_build.go --validate
```

## Benefits

### For Developers

- **Quick Navigation**: Click Markdown links to jump between related modules
- **Clear Dependencies**: See what each module depends on at a glance
- **Consistent Structure**: Standardized documentation format
- **Automated Validation**: Catch broken links in CI/CD

### For AI Systems

- **Token Efficiency**: Small modules with semantic metadata reduce token usage
- **Semantic Understanding**: RDF relationships enable reasoning about code structure
- **Targeted Reading**: JSON index guides AI to relevant modules
- **Relationship Discovery**: Understand dependencies without parsing imports

### For Teams

- **Onboarding**: New developers can navigate codebase via links
- **Architecture Clarity**: Module relationships are explicit
- **Refactoring Safety**: Link validation catches breaking changes
- **Documentation Currency**: Headers live with code, stay up-to-date

## Next Steps

1. Read [Schema Documentation](schema.md)
2. Review [Best Practices](best-practices.md)
3. Check [Examples](examples/)
4. Start adding LinkedDoc headers to your modules!

## Related

- [CLAUDE.md](../../CLAUDE.md) - AI navigation guidelines
- [Issue #72](https://github.com/justin4957/ec2-test-apps/issues/72) - LinkedDoc EPIC
- [Issue #73](https://github.com/justin4957/ec2-test-apps/issues/73) - This implementation
