# Solid Pod Integration - Proof of Concept

This is a standalone application for testing and developing Solid Pod integration for the Location Tracker project. It runs completely separately from the main location-tracker application.

## Overview

The solid-poc app provides:
- **Frontend**: Browser-based authentication PoC (serves `/proof-of-concept`)
  - Full Solid OIDC authentication flow
  - Client-side Pod read/write operations
  - DPoP token handling (automatic via Inrupt libraries)
- **Backend**: Go API for RDF serialization utilities
  - Convert location data to/from RDF formats
  - No Pod operations (see "Architecture Notes" below)

## Architecture

```
solid-poc/
├── main.go                    # Web server with API endpoints
├── internal/solid/
│   └── rdf.go                # RDF serialization (Turtle/JSON-LD)
└── frontend/                 # Browser-based authentication PoC
    ├── index.html            # Interactive authentication UI
    ├── dist/                 # Bundled Solid libraries
    ├── src/                  # Source files for bundling
    └── README.md             # Frontend PoC documentation
```

## Quick Start

### Local Development

```bash
cd solid-poc
go run main.go
```

Then navigate to: `http://localhost:9090`

### Docker

```bash
# Build
docker build -f solid-poc/Dockerfile -t solid-poc .

# Run
docker run -p 9090:9090 solid-poc
```

## API Endpoints

### Health Check ✅
```bash
GET /api/health

# Response:
{
  "status": "ok",
  "service": "solid-poc",
  "version": "1.0.0"
}
```

### Serialize to RDF ✅
```bash
POST /api/rdf/serialize
Content-Type: application/json

{
  "format": "turtle",  // or "jsonld"
  "data": {
    "latitude": 37.7749,
    "longitude": -122.4194,
    "accuracy": 10.5,
    "timestamp": "2025-11-13T10:00:00Z",
    "device_id": "test-device"
  }
}
```

**Response:**
```json
{
  "format": "turtle",
  "result": "@prefix schema: <http://schema.org/> .\n@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .\n\n<#location> a schema:Place, geo:Point ;\n    geo:lat \"37.774900\"^^xsd:decimal ;\n    geo:long \"-122.419400\"^^xsd:decimal ;\n    schema:accuracy \"10.50\"^^xsd:decimal ."
}
```

### Deserialize from RDF ✅
```bash
POST /api/rdf/deserialize
Content-Type: application/json

{
  "format": "jsonld",  // or "turtle"
  "data": "{\"@context\": ...}"
}
```

**Response:**
```json
{
  "latitude": 37.7749,
  "longitude": -122.4194,
  "accuracy": 10.5,
  "device_id": "test-device"
}
```

## Features

### Frontend PoC (`/frontend`) ✅
- OIDC authentication with any Solid provider
- DPoP token handling (automatic via Inrupt libraries)
- WebID profile reading
- Pod write operations (location data in Turtle format)
- Pod read operations with RDF parsing
- Session persistence
- Interactive step-by-step UI

### Backend Library (`internal/solid`) ✅
- RDF serialization (Turtle and JSON-LD)
- RDF deserialization (JSON-LD)
- Location data to RDF conversion
- Follows schema from SOLID_DATA_MODELS.md

## Pod Operations

**ALL Pod operations (read/write) are done client-side** in the browser using the Inrupt Solid Client library.

The frontend proof-of-concept demonstrates:
1. Authenticating with a Solid provider
2. Creating containers in your Pod
3. Writing location data as Turtle RDF
4. Reading location data back from the Pod

**There are no server-side Pod operation endpoints** - see "Architecture Notes" below for why.

## Testing

### Prerequisites
1. A Solid Pod account (SolidCommunity.net or Inrupt PodSpaces)
2. Web browser with JavaScript enabled
3. HTTP server (not `file://` protocol)

### Test Flow
1. Start solid-poc: `go run main.go`
2. Navigate to `http://localhost:9090`
3. Follow the interactive authentication flow
4. Test Pod read/write operations in the UI
5. Test RDF serialization via API:

```bash
# Test RDF serialization
curl -X POST http://localhost:9090/api/rdf/serialize \
  -H "Content-Type: application/json" \
  -d '{
    "format": "turtle",
    "data": {
      "latitude": 37.7749,
      "longitude": -122.4194,
      "device_id": "test"
    }
  }'
```

## Architecture Notes

### Why No Server-Side Pod Operations?

This PoC originally included server-side Pod operation endpoints (`/api/pod/read`, `/api/pod/write`), but they were removed because **they fundamentally cannot work with DPoP authentication**.

**The Problem:**
- DPoP tokens are cryptographically bound to the HTTP client that created them
- Each Pod request requires a new DPoP proof signed with the client's private key
- The backend doesn't have access to the private key
- Pod servers return 401 Unauthorized when the backend tries to authenticate

**The Solution:**
All Pod operations must be done **client-side in the browser** using `session.fetch()` from the Inrupt Solid Client library. The frontend PoC demonstrates this working correctly.

### Correct Architecture

**Client-Side Pod Operations** ✅ (Current):
```javascript
// In browser with authenticated session
const data = await session.fetch(podUrl);  // Works - has private key
```

**Server-Side Pod Operations** ❌ (Not Possible with DPoP):
```bash
curl -X POST /api/pod/read \
  -d '{"token": "...", "url": "..."}' # Fails - no private key
```

### Future Options for Server-Side Access

To enable true server-side Pod operations, you would need:

1. **Client Credentials Flow**: App-level authentication (not user-specific)
2. **Refresh Token Storage**: Store user's refresh token server-side (security implications)
3. **Proxy Pattern**: Frontend makes authenticated requests, backend processes data
4. **Service Worker**: Keep authentication in browser, backend handles business logic

For this PoC, **all Pod operations are done client-side** in the frontend.

## Integration with Location Tracker

This PoC is intentionally isolated from the main location-tracker app. When ready for production integration:

1. The `internal/solid/rdf.go` package can be imported by location-tracker
2. Frontend auth flow can be integrated into main UI
3. Pod operations will remain client-side (browser-based)

## Namespacing

In production deployment, this app is served under the `/solid/` namespace:
- `/solid/` → Frontend PoC
- `/solid/api/*` → Backend API endpoints

This is configured in nginx reverse proxy.

## Development Status

**Phase 1: Foundation & Research** ✅ Complete
- Authentication research
- RDF data models
- Browser-based PoC

**Phase 2: Backend Integration** ✅ Complete
- RDF serialization library
- Cleaned up non-functional code
- Accurate documentation

**Phase 3: Production Integration** 🔜 Next
- Import RDF library into location-tracker
- Integrate frontend auth into main UI
- Deploy alongside main app

## Next Steps

After validating this PoC:
1. **Issue #50**: Integrate Solid authentication into location-tracker frontend
2. **Issue #51**: Use RDF serialization for location data
3. **Issue #52**: Implement client-side Pod operations in main app

## Resources

- **Solid Protocol**: https://solidproject.org/TR/protocol
- **Inrupt Docs**: https://docs.inrupt.com/
- **RDF Data Models**: See `../SOLID_DATA_MODELS.md`
- **Authentication Research**: See `../SOLID_AUTHENTICATION.md`
- **Frontend PoC**: See `./frontend/README.md`

## License

Part of the Location Tracker project. See main repository for license information.
