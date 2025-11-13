# Solid Integration - Isolated Proof of Concept

## Overview

The Solid Pod integration is being developed as a **completely isolated proof-of-concept application** (`solid-poc`) that runs alongside the main Location Tracker application. This approach keeps the experimental Solid features separate from the production-stable location-tracker code.

## Architecture

```
ec2-test-apps/
├── location-tracker/          # Main production application (unchanged)
│   ├── main.go
│   ├── Dockerfile
│   └── ...
│
├── solid-poc/                 # Isolated Solid PoC application
│   ├── main.go               # Go web server with API endpoints
│   ├── internal/solid/       # Reusable Solid library
│   │   ├── client.go        # HTTP client with DPoP
│   │   ├── dpop.go          # Token validation
│   │   └── rdf.go           # RDF serialization
│   ├── Dockerfile
│   └── README.md
│
└── proof-of-concept/         # Browser-based authentication PoC
    ├── index.html            # Interactive Solid auth UI
    ├── dist/                 # Bundled JavaScript libraries
    └── README.md
```

## Deployment Strategy

### Development
- **Location Tracker**: Runs on port 8080 (main app)
- **Solid PoC**: Runs on port 9090 (isolated testing)

### Production (via Nginx)
```
https://notspies.org/              → location-tracker (main app)
https://notspies.org/solid/        → solid-poc (isolated PoC)
https://notspies.org/solid/api/*   → solid-poc backend API
```

All Solid functionality is namespaced under `/solid/` to keep it completely separate from the main application.

## Benefits of Isolation

### 1. **Zero Risk to Production**
- Main location-tracker app is unaffected by Solid experiments
- No dependencies added to location-tracker
- Can test/break Solid features without impacting users

### 2. **Independent Development**
- solid-poc can be developed/deployed independently
- Different release cycles for main app vs. Solid features
- Easy to roll back or remove if needed

### 3. **Clear Boundaries**
- Clean separation of concerns
- Easy to reason about which code is experimental
- No mixing of stable production code with PoC code

### 4. **Future Integration Path**
When Solid integration is mature and tested:
- The `internal/solid` library can be imported by location-tracker
- API endpoints can be integrated into main app
- Or keep running as separate service (microservices approach)

## What Was Removed from Location Tracker

The following Solid-related code was removed from `location-tracker/`:

- ✅ `solid_auth.go` - OIDC authentication handlers
- ✅ `storage.go` - DataStore abstraction interface
- ✅ `/api/solid/*` routes - All Solid API endpoints
- ✅ `cleanupExpiredStates()` goroutine - OIDC state cleanup
- ✅ `internal/solid/` directory - Moved to solid-poc

Location Tracker now has **zero** Solid dependencies or code.

## Current Status

### Completed ✅
- **Phase 1.1**: Development environment setup
- **Phase 1.2**: Authentication research and documentation
- **Phase 1.3**: RDF data model design
- **Phase 1.4**: Browser-based authentication PoC
- **Phase 2.1**: Go Solid client library (solid-poc)

### In Progress 🚧
- Isolated solid-poc application
- Backend API endpoints
- RDF serialization (Turtle/JSON-LD)
- DPoP token validation

### Next Steps 📋
- Deploy solid-poc alongside location-tracker
- Configure nginx for `/solid/` namespace
- Test end-to-end with real Solid Pods
- Iterate on PoC features independently

## Testing the Solid PoC

### Local Development

```bash
# Terminal 1: Run location-tracker (main app)
cd location-tracker
go run main.go
# Access at: http://localhost:8080

# Terminal 2: Run solid-poc (isolated)
cd solid-poc
go run main.go
# Access at: http://localhost:9090
```

### Docker

```bash
# Build solid-poc image
docker build -f solid-poc/Dockerfile -t solid-poc .

# Run solid-poc container
docker run -p 9090:9090 solid-poc

# Access at: http://localhost:9090
```

## API Endpoints

### Solid PoC Backend (`/solid/api/`)
- `GET /api/health` - Health check
- `POST /api/validate-token` - Validate DPoP token
- `POST /api/pod/read` - Read from Pod (server-side)
- `POST /api/pod/write` - Write to Pod (server-side)
- `POST /api/rdf/serialize` - Convert data to RDF
- `POST /api/rdf/deserialize` - Parse RDF to data

### Location Tracker (unchanged)
- All existing endpoints remain at root (`/api/*`)
- No Solid functionality in main app

## Documentation

- **solid-poc/README.md** - Solid PoC application documentation
- **proof-of-concept/README.md** - Browser PoC testing instructions
- **SOLID_AUTHENTICATION.md** - Authentication research
- **SOLID_DATA_MODELS.md** - RDF schema specifications
- **SOLID_INTEGRATION_ROADMAP.md** - Full integration plan

## Future Integration Options

### Option 1: Import Library (Recommended)
```go
import "github.com/justin4957/ec2-test-apps/solid-poc/internal/solid"

// In location-tracker
client, err := solid.NewClient(dPopToken)
data, _, err := client.GetResource(ctx, podURL)
```

### Option 2: Microservices
- Keep solid-poc as separate service
- Location-tracker calls solid-poc APIs
- Independent scaling and deployment

### Option 3: Merge When Mature
- Copy proven code from solid-poc to location-tracker
- Deprecate solid-poc once integrated
- Full feature parity with main app

## Rollback Strategy

If Solid integration proves problematic:
1. Simply don't deploy solid-poc to production
2. Remove `/solid/` nginx configuration
3. Delete solid-poc directory
4. Zero impact on location-tracker

## Decision Log

**2025-11-13**: Isolated Solid PoC
- **Decision**: Create separate solid-poc application
- **Rationale**: Keep experimental features isolated from production
- **Impact**: Zero risk to main app, clean development boundaries
- **Alternatives Considered**: Adding to location-tracker (rejected - too risky)

---

This isolation approach allows us to innovate rapidly on Solid integration while maintaining 100% stability and reliability for the production Location Tracker application.
