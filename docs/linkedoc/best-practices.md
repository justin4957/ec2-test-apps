# LinkedDoc+TTL Best Practices

## Writing LinkedDoc Headers

### ✅ DO: Keep Descriptions Concise

One line, focused on the module's primary purpose.

**Good:**
```go
/*
# Module: handlers/location.go
HTTP handlers for location tracking endpoints.
```

**Bad:**
```go
/*
# Module: handlers/location.go
This module contains a comprehensive collection of HTTP handler functions that
are designed to process various types of HTTP requests that are related to the
location tracking functionality within our application, including but not
limited to creating new location records in the database and retrieving
existing location information based on various criteria.
```

### ✅ DO: Link All Direct Dependencies

Include every module your code imports and uses.

**Complete:**
```go
## Linked Modules
- [types/location](../types/location.go): Location data structures
- [storage/dynamodb](../storage/dynamodb.go): Persistence layer
- [services/business_search](../services/business_search.go): Business logic
- [middleware/auth](../middleware/auth.go): Authentication
```

**Incomplete:**
```go
## Linked Modules
- [types/location](../types/location.go): Location data structures
// Missing storage, services, middleware!
```

### ✅ DO: Use Descriptive Relationship Labels

Explain WHY the module is linked, not just WHAT it is.

**Good:**
```go
- [storage/dynamodb](../storage/dynamodb.go): Persists location data to DynamoDB
- [services/geocoding](../services/geocoding.go): Converts addresses to coordinates
```

**Bad:**
```go
- [storage/dynamodb](../storage/dynamodb.go): Database stuff
- [services/geocoding](../services/geocoding.go): Helper
```

### ✅ DO: Use Tags from the Taxonomy

Stick to defined tags. Don't invent new ones without updating the schema.

**Good:**
```go
## Tags
http, location, api, storage
```

**Bad:**
```go
## Tags
web-stuff, geo-things, database-operations, misc
```

### ✅ DO: List All Exports

Include every public function and type.

**Complete:**
```go
## Exports
HandleLocation, HandleLocationByID, HandleDeleteLocation, HandleUpdateLocation
```

**Incomplete:**
```go
## Exports
HandleLocation
// Missing the other 3 handlers!
```

### ✅ DO: Keep RDF Concise

The RDF section should mirror the Markdown, not add new information.

**Good:**
```turtle
<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "handlers/location.go" ;
    code:description "HTTP handlers for location tracking endpoints" ;
    code:dependsOn <../types/location.go>, <../storage/dynamodb.go> ;
    code:exports :HandleLocation, :HandleLocationByID ;
    code:tags "http", "location", "api" .
<!-- End LinkedDoc RDF -->
```

**Too verbose:**
```turtle
<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix dcterms: <http://purl.org/dc/terms/> .

<this> a code:Module ;
    code:name "handlers/location.go" ;
    code:description "HTTP handlers for location tracking endpoints" ;
    code:packageName "handlers" ;
    code:category "handlers" ;
    code:dependsOn <../types/location.go> ;
    code:dependsOn <../storage/dynamodb.go> ;
    code:exports :HandleLocation ;
    code:exports :HandleLocationByID ;
    code:tags "http" ;
    code:tags "location" ;
    code:tags "api" ;
    code:lastUpdated "2025-11-13T12:00:00Z"^^xsd:dateTime ;
    code:author "Development Team" ;
    code:version "1.0.0" ;
    code:linesOfCode 150 ;
    code:testCoverage 0.85 ;
    code:complexity 0.4 .
<!-- End LinkedDoc RDF -->
```

## File Organization

### ✅ DO: Keep Files Small

Target 300 lines or fewer. Hard limit 500 lines.

**Good:**
```
handlers/
├── location.go (250 lines)
├── errors.go (180 lines)
└── tips.go (220 lines)
```

**Bad:**
```
handlers/
└── all_handlers.go (2,500 lines) ❌
```

### ✅ DO: Group by Function

Organize packages by functional responsibility.

**Good:**
```
location-tracker/
├── handlers/    # HTTP layer
├── services/    # Business logic
├── types/       # Data structures
├── storage/     # Persistence
└── clients/     # External APIs
```

**Bad:**
```
location-tracker/
├── location_stuff/   # Mixed concerns
├── utils/            # Kitchen sink
└── helpers/          # Everything else
```

### ✅ DO: One Responsibility Per File

Each file should have a single, clear purpose.

**Good:**
```
services/
├── location_service.go        # Location business logic
├── business_search.go         # Business search logic
└── commercial_real_estate.go  # CRE search logic
```

**Bad:**
```
services/
└── service.go  # All business logic! ❌
```

## Dependency Management

### ✅ DO: Follow Dependency Flow

```
handlers → services → storage
    ↓
  clients
```

**Good:**
```go
// handlers/location.go
import (
    "project/services"  // ✓ handlers call services
    "project/types"     // ✓ handlers use types
)
```

**Bad:**
```go
// storage/dynamodb.go
import (
    "project/handlers"  // ✗ storage should not know about handlers
)
```

### ❌ DON'T: Create Circular Dependencies

**Bad:**
```
handlers/location.go imports services/location.go
services/location.go imports handlers/location.go  ❌
```

Use interfaces or dependency injection to break cycles.

### ✅ DO: Minimize Dependencies

Only import what you actually use.

**Good:**
```go
import (
    "project/types"     // Used for Location struct
    "project/storage"   // Used for DB operations
)
```

**Bad:**
```go
import (
    "project/types"
    "project/storage"
    "project/services"  // Not actually used ❌
    "project/clients"   // Not actually used ❌
)
```

## Updating LinkedDoc Headers

### ✅ DO: Update When Refactoring

If you move code, update ALL references.

**Scenario:** Moving `getLocation()` from handlers to services

1. Update the module that moved:
   ```go
   // services/location.go (new file)
   /*
   # Module: services/location.go
   Location business logic and orchestration.
   ```

2. Update all modules that depended on it:
   ```go
   // handlers/location.go
   ## Linked Modules
   - [services/location](../services/location.go): Location business logic
   ```

### ✅ DO: Update When Adding Dependencies

If you add an import, add a link.

**Before:**
```go
## Linked Modules
- [types/location](../types/location.go): Location data structures
```

**After adding cache import:**
```go
## Linked Modules
- [types/location](../types/location.go): Location data structures
- [storage/cache](../storage/cache.go): Location caching layer
```

### ✅ DO: Update Tags When Purpose Changes

If a module's role changes, update its tags.

**Before (just HTTP):**
```go
## Tags
http, location, api
```

**After adding caching:**
```go
## Tags
http, location, api, cache
```

## Validation

### ✅ DO: Validate Before Committing

```bash
go run tools/linkedoc_build.go --validate
```

Add this to your pre-commit hook or CI pipeline.

### ✅ DO: Regenerate Index After Changes

```bash
go run tools/linkedoc_build.go --generate-index
```

This keeps the AI-optimized JSON index current.

### ✅ DO: Fix Broken Links Immediately

If validation fails:
1. Don't commit
2. Fix the broken link or remove it
3. Re-validate
4. Then commit

## Common Patterns

### Pattern: HTTP Handler

```go
/*
# Module: handlers/<name>.go
HTTP handlers for <feature> endpoints.

## Linked Modules
- [types/<name>](../types/<name>.go): <Feature> data structures
- [services/<name>](../services/<name>.go): <Feature> business logic
- [middleware/auth](../middleware/auth.go): Authentication

## Tags
http, <domain>, api

## Exports
Handle<Action>, Handle<Action>By ID

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "handlers/<name>.go" ;
    code:description "HTTP handlers for <feature> endpoints" ;
    code:dependsOn <../types/<name>.go>, <../services/<name>.go> ;
    code:exports :Handle<Action> ;
    code:tags "http", "<domain>", "api" .
<!-- End LinkedDoc RDF -->
*/
package handlers
```

### Pattern: Business Service

```go
/*
# Module: services/<name>.go
Business logic for <feature> operations.

## Linked Modules
- [types/<name>](../types/<name>.go): <Feature> data structures
- [storage/<store>](../storage/<store>.go): Data persistence
- [clients/<api>](../clients/<api>.go): External API integration

## Tags
business-logic, <domain>

## Exports
<Name>Service, New<Name>Service

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "services/<name>.go" ;
    code:description "Business logic for <feature> operations" ;
    code:dependsOn <../types/<name>.go>, <../storage/<store>.go> ;
    code:integratesWith <../clients/<api>.go> ;
    code:exports :<Name>Service ;
    code:tags "business-logic", "<domain>" .
<!-- End LinkedDoc RDF -->
*/
package services
```

### Pattern: Data Type

```go
/*
# Module: types/<name>.go
Data structures for <feature>.

## Linked Modules
(Usually none - types are leaf nodes)

## Tags
data-types, <domain>

## Exports
<Name>, <Name>Request, <Name>Response

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Type ;
    code:name "types/<name>.go" ;
    code:description "Data structures for <feature>" ;
    code:exports :<Name> ;
    code:tags "data-types", "<domain>" .
<!-- End LinkedDoc RDF -->
*/
package types
```

### Pattern: Storage Layer

```go
/*
# Module: storage/<store>.go
<Storage> persistence operations.

## Linked Modules
- [types/<name>](../types/<name>.go): Data structures to persist

## Tags
storage, <technology>

## Exports
<Name>Store, New<Name>Store

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "storage/<store>.go" ;
    code:description "<Storage> persistence operations" ;
    code:dependsOn <../types/<name>.go> ;
    code:exports :<Name>Store ;
    code:tags "storage", "<technology>" .
<!-- End LinkedDoc RDF -->
*/
package storage
```

## Anti-Patterns

### ❌ DON'T: Skip the Header

Every module needs a LinkedDoc header. No exceptions.

### ❌ DON'T: Copy-Paste Without Updating

Don't copy a template and forget to update the specifics.

### ❌ DON'T: Link to Non-Existent Files

All linked paths must exist. Validation will catch this.

### ❌ DON'T: Use Relative Paths from Wrong Location

Links are relative to the current file, not the project root.

**Wrong:**
```go
// In handlers/location.go
- [types/location](types/location.go)  ❌
```

**Right:**
```go
// In handlers/location.go
- [types/location](../types/location.go)  ✓
```

### ❌ DON'T: Make Up Your Own RDF Predicates

Use only predicates defined in the schema.

**Wrong:**
```turtle
<this> code:madeUpPredicate "value" .  ❌
```

**Right:**
```turtle
<this> code:tags "value" .  ✓
```

## Summary Checklist

Before committing code with LinkedDoc headers:

- [ ] Header present in every `.go` file
- [ ] Description is one concise line
- [ ] All imports listed in Linked Modules
- [ ] Relationship descriptions are clear
- [ ] Tags are from the taxonomy
- [ ] All exports listed
- [ ] RDF mirrors Markdown
- [ ] Relative paths are correct
- [ ] Ran `linkedoc_build --validate`
- [ ] All links resolve
- [ ] No circular dependencies
- [ ] File under 500 lines

## Questions?

See:
- [LinkedDoc README](README.md)
- [Schema Reference](schema.md)
- [Examples](examples/)
- [Issue #72](https://github.com/justin4957/ec2-test-apps/issues/72)
