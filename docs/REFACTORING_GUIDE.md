# Location Tracker Refactoring Guide

This guide documents the incremental refactoring of `location-tracker/main.go` from a 5,305-line monolith into a modular, maintainable architecture.

## Goals

1. **Reduce main.go to <300 lines** (routing and initialization only)
2. **Create modular packages** with clear responsibilities
3. **Add LinkedDoc headers** to all modules for AI navigation
4. **Maintain 100% backwards compatibility** - all endpoints work identically
5. **Improve testability** through better separation of concerns

## Current State

```
location-tracker/main.go: 5,305 lines
├── Types (15+ structs)
├── Global variables
├── HTTP handlers (18+ functions)
├── Business logic functions
├── DynamoDB operations
├── External API clients
└── Helper functions
```

**Problems:**
- Difficult to navigate
- Hard for AI to process (token intensive)
- Testing is challenging
- Merge conflicts common
- Unclear dependencies

## Target Architecture

```
location-tracker/
├── main.go (~200 lines)          # Routing & initialization only
├── types/                         # Data structures
│   ├── location.go
│   ├── error_log.go
│   ├── tip.go
│   └── ...
├── handlers/                      # HTTP handlers (~15 files)
│   ├── health.go
│   ├── location.go
│   ├── errors.go
│   └── ...
├── services/                      # Business logic
│   ├── location_service.go
│   ├── business_search.go
│   └── ...
├── storage/                       # Data persistence
│   ├── dynamodb.go
│   └── cache.go
└── clients/                       # External APIs
    ├── perplexity.go
    ├── google_places.go
    └── ...
```

## Refactoring Phases

### Phase 1: Extract HTTP Handlers (Issue #76) ✅ Started

**Goal:** Move all `handleXXX` functions to `handlers/` package

**Status:**
- ✅ Created handlers/ package
- ✅ Extracted health handler as example
- ⏳ 17+ handlers remaining

**Next Steps:**
1. Extract simple handlers first (health, cryptogram info)
2. Extract handlers with dependencies (location, errors, tips)
3. Update main.go route registration
4. Test each extraction

### Phase 2: Extract Business Logic (Issue #77)

**Goal:** Move business logic to `services/` package

**Functions to extract:**
- Commercial real estate search
- Business search and discovery
- Context tracking
- Location generation
- Distance calculations
- Keyword extraction

**Dependencies:** Requires Phase 1 to be mostly complete

### Phase 3: Extract Data Types (Issue #78)

**Goal:** Move all type definitions to `types/` package

**Types to extract:**
- Location, ErrorLog, AnonymousTip, Donation
- CommercialRealEstate, Business
- API request/response types
- Context types

**Note:** Can be done in parallel with Phase 1

### Phase 4: Extract Storage Layer (Issue #79)

**Goal:** Move DynamoDB operations to `storage/` package

**Functions to extract:**
- saveLocationToDynamoDB
- getLocationsFromDynamoDB
- saveTipToDynamoDB
- getCachedCommercialRealEstate

### Phase 5: Extract API Clients (Issue #80)

**Goal:** Move external API clients to `clients/` package

**Clients to create:**
- Google Places API
- Perplexity API
- Twilio API
- Stripe API

### Phase 6: Final Cleanup

**Goal:** Clean and minimal main.go

- Remove all extracted code
- Keep only routing and initialization
- Add comprehensive comments
- Final LinkedDoc validation

## Step-by-Step: Extracting a Handler

### 1. Create New Handler File

```bash
# Create file in handlers/
touch location-tracker/handlers/<name>.go
```

### 2. Add LinkedDoc Header

```go
/*
# Module: handlers/<name>.go
<Brief description>

## Linked Modules
- [types/<type>](../types/<type>.go): <relationship>

## Tags
http, <domain>, api

## Exports
Handle<Action>

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "handlers/<name>.go" ;
    code:description "<description>" ;
    code:dependsOn <../types/<type>.go> ;
    code:exports :Handle<Action> ;
    code:tags "http", "<domain>", "api" .
<!-- End LinkedDoc RDF -->
*/
package handlers

import (
    "net/http"
    // Add other imports as needed
)
```

### 3. Copy Handler Function

```go
// Copy from main.go
// Change: func handleXXX → func HandleXXX (capitalize!)
func HandleXXX(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...
}
```

### 4. Update Imports

Add any imports the handler needs:
```go
import (
    "encoding/json"
    "net/http"
    "github.com/justin4957/ec2-test-apps/location-tracker/types"
)
```

### 5. Update main.go

```go
// Old
http.HandleFunc("/api/xxx", handleXXX)

// New
import "github.com/justin4957/ec2-test-apps/location-tracker/handlers"

http.HandleFunc("/api/xxx", handlers.HandleXXX)
```

### 6. Test

```bash
# Build
cd location-tracker
go build

# Run
./location-tracker

# Test endpoint
curl http://localhost:5000/api/xxx

# Validate
cd ..
go run tools/*.go --validate --path location-tracker/handlers/
```

### 7. Commit

```bash
git add location-tracker/handlers/<name>.go location-tracker/main.go
git commit -m "Extract <name> handler to handlers/ package

- Created handlers/<name>.go with LinkedDoc header
- Updated main.go route registration
- Tested endpoint functionality
- LinkedDoc validation passing

Related: #76"
```

## Handling Dependencies

### Global Variables

**Problem:** Handlers use globals like `dynamoDBTable`, `perplexityAPIKey`

**Solution:** Dependency injection

```go
// Before
func handleLocation(...) {
    // Uses global: dynamoDBTable
}

// After
type LocationHandler struct {
    dynamoDBTable string
}

func (h *LocationHandler) HandleLocation(...) {
    // Uses h.dynamoDBTable
}
```

### Types from main.go

**Problem:** Handlers use types defined in main.go

**Solution:** Extract types first (or import main package temporarily)

```go
// Temporary solution
import "github.com/justin4957/ec2-test-apps/location-tracker"

func HandleLocation(w http.ResponseWriter, r *http.Request) {
    var loc main.Location  // Until types are extracted
}

// Better solution
import "github.com/justin4957/ec2-test-apps/location-tracker/types"

func HandleLocation(w http.ResponseWriter, r *http.Request) {
    var loc types.Location
}
```

### Shared Helper Functions

**Problem:** Handlers use helper functions from main.go

**Solution:**
1. Move helpers to appropriate package (services, storage, or internal)
2. Or duplicate temporarily until proper home is determined

## Testing Strategy

### Before Extraction

```bash
# Test the endpoint works
curl -X POST http://localhost:5000/api/location \
  -H "Content-Type: application/json" \
  -d '{"latitude": 37.7749, "longitude": -122.4194}'
```

### After Extraction

```bash
# Test it still works the same
curl -X POST http://localhost:5000/api/location \
  -H "Content-Type: application/json" \
  -d '{"latitude": 37.7749, "longitude": -122.4194}'

# Response should be identical
```

### Automated Testing

```bash
# Validate LinkedDoc headers
go run tools/*.go --validate

# Run Go tests
go test ./...

# Build check
go build
```

## Common Issues

### Import Cycles

**Problem:** `handlers` imports `types`, `types` imports `handlers`

**Solution:** One-way dependencies only
```
handlers → services → storage
    ↓         ↓
  types     types
```

### Missing Exports

**Problem:** `cannot refer to unexported name main.someFunc`

**Solution:** Capitalize function names when extracting

```go
// Before (in main.go)
func handleLocation(...) { }

// After (in handlers/location.go)
func HandleLocation(...) { }  // Capital H
```

### Build Errors After Extraction

**Problem:** `undefined: handleXXX`

**Solution:** Update all references in main.go

```bash
# Find remaining references
grep "handleXXX" location-tracker/main.go

# Replace with handlers.HandleXXX
```

## Progress Tracking

```bash
# Count handlers in main.go
grep "^func handle" location-tracker/main.go | wc -l

# Count extracted handlers
ls -1 location-tracker/handlers/*.go | grep -v README | wc -l

# Lines in main.go (goal: <300)
wc -l location-tracker/main.go
```

## Validation

After each extraction:

```bash
# 1. LinkedDoc validation
go run tools/*.go --validate

# 2. Build check
cd location-tracker && go build

# 3. Generate updated index
cd .. && go run tools/*.go --generate-index
```

## File Size Guidelines

From CLAUDE.md:
- **Hard limit:** 500 lines
- **Target:** <300 lines
- **Ideal:** <200 lines for handlers

If a handler exceeds these limits, split it further.

## Benefits

### For Developers
- ✅ Easy to find specific handler
- ✅ Clear file purpose
- ✅ Reduced merge conflicts
- ✅ Better testability

### For AI Systems
- ✅ Token efficient (read small files)
- ✅ Clear dependencies via LinkedDoc
- ✅ JSON index for navigation
- ✅ Better code comprehension

### For Codebase
- ✅ Modular architecture
- ✅ Clear separation of concerns
- ✅ Maintainable at scale
- ✅ Easier onboarding

## Timeline

**Estimated:** 8-12 weeks for complete refactoring

- Phase 1 (Handlers): 2-3 weeks
- Phase 2 (Services): 2-3 weeks
- Phase 3 (Types): 1 week
- Phase 4 (Storage): 1-2 weeks
- Phase 5 (Clients): 1-2 weeks
- Phase 6 (Cleanup): 1 week

**Incremental approach:** Can merge PRs after each handler extraction

## Related Issues

- #72 - LinkedDoc+TTL EPIC
- #76 - Extract HTTP handlers
- #77 - Extract business logic
- #78 - Extract data types
- #79 - Extract storage layer
- #80 - Extract API clients

## Questions?

See:
- [handlers/README.md](../location-tracker/handlers/README.md)
- [LinkedDoc documentation](linkedoc/)
- [CLAUDE.md](../CLAUDE.md)
- Issue #76 discussion
