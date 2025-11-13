# Location Tracker Handlers

HTTP request handlers extracted from `main.go` into modular, maintainable files.

## Overview

This package contains all HTTP endpoint handlers for the location tracker application. Each handler is in its own file with a LinkedDoc header for documentation and navigation.

## Refactoring Status

🚧 **In Progress** - Incremental extraction from main.go (5,305 lines)

### Extracted Handlers

- ✅ `health.go` - Health check endpoint (Example/Proof of Concept)

### Remaining in main.go

- [ ] `handleLogin` → `auth.go`
- [ ] `handleVerifyTurnstile` → `auth.go`
- [ ] `handleLocation` → `location.go`
- [ ] `handleErrorLogs`, `handleErrorLogByID` → `errors.go`
- [ ] `handleBusinesses` → `business.go`
- [ ] `handleCommercialContext`, `handlePendingKeywords`, `handleLastInteractionContext`, `handleCommercialRealEstate` → `commercial.go`
- [ ] `handleTwilioWebhook` → `twilio.go`
- [ ] `handleTips`, `handleTipByID` → `tips.go`
- [ ] `handleCryptogram`, `handleCryptogramInfo` → `cryptogram.go`
- [ ] `handleCreatePaymentIntent`, `handleStripeWebhook` → `stripe.go`
- [ ] Additional handlers...

## File Organization

Each handler file follows this pattern:

```
handlers/
├── health.go         # Health check (✅ complete)
├── auth.go           # Authentication endpoints
├── location.go       # Location tracking
├── errors.go         # Error log management
├── business.go       # Business search
├── commercial.go     # Commercial real estate
├── twilio.go         # Twilio webhooks
├── tips.go           # Anonymous tips
├── cryptogram.go     # Cryptogram puzzles
├── stripe.go         # Stripe payments
└── README.md         # This file
```

## LinkedDoc Standards

All handler files include LinkedDoc headers:

```go
/*
# Module: handlers/<name>.go
<Description>

## Linked Modules
- [<module>](<path>): <relationship>

## Tags
http, <domain>, api

## Exports
Handle<Action>

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "handlers/<name>.go" ;
    code:description "<description>" ;
    code:dependsOn <path> ;
    code:exports :Handle<Action> ;
    code:tags "http", "<domain>", "api" .
<!-- End LinkedDoc RDF -->
*/
package handlers
```

## Refactoring Guidelines

### 1. Extract One Handler at a Time

```bash
# 1. Create new handler file
# 2. Add LinkedDoc header
# 3. Copy handler function from main.go
# 4. Capitalize function name (make public)
# 5. Update imports in new file
# 6. Update main.go to use handlers.HandleX
# 7. Test the endpoint
# 8. Commit
```

### 2. Update main.go Route Registration

```go
// Old (in main.go)
http.HandleFunc("/api/health", handleHealth)

// New (after extraction)
http.HandleFunc("/api/health", handlers.HandleHealth)
```

### 3. Maintain Existing Functionality

**Critical:** All endpoints must continue working exactly as before. This is a refactoring, not a rewrite.

- Same request/response formats
- Same error handling
- Same dependencies
- Same behavior

### 4. Keep Files Small

Target: <300 lines per file

If a handler file exceeds 300 lines, consider splitting further:
- Multiple related endpoints can share a file
- Complex handlers might need their own file

## Dependencies

When extracting handlers, you may need to extract dependencies first:

### Types Needed
Many handlers use types defined in main.go:
- `Location`
- `ErrorLog`
- `Business`
- `AnonymousTip`
- etc.

**Solution:** Extract these to `location-tracker/types/` package first

### Global Variables Needed
Some handlers use shared state:
- `useDynamoDB`, `dynamoDBTable`
- `perplexityAPIKey`, `googleMapsAPIKey`
- Authentication state

**Solution:** Pass via dependency injection or context

## Testing Strategy

After each handler extraction:

```bash
# 1. Build
cd location-tracker
go build

# 2. Run locally
./location-tracker

# 3. Test endpoint
curl http://localhost:5000/api/health

# 4. Verify response matches original
```

## Migration Checklist

For each handler extraction:

- [ ] Create handler file with LinkedDoc header
- [ ] Copy function from main.go
- [ ] Capitalize function name (export it)
- [ ] Update imports
- [ ] Update main.go route registration
- [ ] Test endpoint locally
- [ ] Run `go run tools/*.go --validate`
- [ ] Commit with clear message

## Benefits of Extraction

### Before (main.go - 5,305 lines)
```go
func handleHealth(...) { }
func handleLogin(...) { }
func handleLocation(...) { }
// ... 15+ more handlers
// ... plus types, globals, helpers
```

**Problems:**
- Hard to navigate
- Difficult for AI to process (token limit)
- Merge conflicts common
- Testing is difficult
- Unclear dependencies

### After (handlers/ package)
```
handlers/
├── health.go (30 lines)
├── location.go (150 lines)
├── errors.go (120 lines)
└── ...
```

**Benefits:**
- Easy to find specific handler
- AI can read individual files efficiently
- Clear module boundaries
- Easier testing
- LinkedDoc documents dependencies
- Reduced merge conflicts

## Next Steps

1. **Extract types first** (see #78) - Required by many handlers
2. **Extract remaining handlers** - One at a time, test each
3. **Extract business logic to services/** (see #77)
4. **Final cleanup** - Remove extracted code from main.go

## Related Issues

- #72 - LinkedDoc+TTL EPIC
- #76 - Extract HTTP handlers (this PR)
- #77 - Extract business logic to services/
- #78 - Extract data types to types/

## Progress Tracking

Track progress with:
```bash
# Count remaining handlers in main.go
grep "^func handle" location-tracker/main.go | wc -l

# Count extracted handlers
ls -1 location-tracker/handlers/*.go | wc -l
```

**Current Status:**
- Extracted: 1 handler (health)
- Remaining: 17+ handlers
- Target: All handlers in handlers/ package
- Goal: main.go < 300 lines (routing only)
