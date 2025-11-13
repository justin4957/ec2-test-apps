# Solid Pod Integration - Proof of Concept

This is a standalone application for testing and developing Solid Pod integration for the Location Tracker project. It runs completely separately from the main location-tracker application.

## Overview

The solid-poc app provides:
- **Frontend**: Browser-based authentication PoC (serves `/proof-of-concept`)
- **Backend**: Go API for server-side Pod operations
- **Library**: Reusable `internal/solid` package for Pod HTTP operations and RDF serialization

## Architecture

```
solid-poc/
├── main.go                    # Web server with API endpoints
├── internal/solid/
│   ├── client.go             # HTTP client for Pod operations
│   ├── dpop.go               # DPoP token validation
│   └── rdf.go                # RDF serialization (Turtle/JSON-LD)
└── ../proof-of-concept/      # Browser-based authentication PoC
    ├── index.html            # Interactive authentication UI
    ├── dist/                 # Bundled Solid libraries
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

### Health Check
```bash
GET /api/health
```

### Validate DPoP Token
```bash
POST /api/validate-token
Content-Type: application/json

{
  "token": "eyJ0eXAiOiJkcG9wK2p3dCIsImFsZyI6..."
}
```

### Read from Pod (Server-Side)
```bash
POST /api/pod/read
Content-Type: application/json

{
  "token": "eyJ0eXAiOiJkcG9wK2p3dCIsImFsZyI6...",
  "url": "https://alice.solidcommunity.net/private/data.ttl"
}
```

### Write to Pod (Server-Side)
```bash
POST /api/pod/write
Content-Type: application/json

{
  "token": "eyJ0eXAiOiJkcG9wK2p3dCIsImFsZyI6...",
  "url": "https://alice.solidcommunity.net/private/data.ttl",
  "data": "@prefix schema: <http://schema.org/> ...",
  "content_type": "text/turtle"
}
```

### Serialize to RDF
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

### Deserialize from RDF
```bash
POST /api/rdf/deserialize
Content-Type: application/json

{
  "format": "jsonld",  // or "turtle"
  "data": "{\"@context\": ...}"
}
```

## Features

### Frontend PoC (`/proof-of-concept`)
- ✅ OIDC authentication with any Solid provider
- ✅ DPoP token handling (automatic via Inrupt libraries)
- ✅ WebID profile reading
- ✅ Pod write operations (location data in Turtle format)
- ✅ Pod read operations with RDF parsing
- ✅ Session persistence
- ✅ Interactive step-by-step UI

### Backend Library (`internal/solid`)
- ✅ HTTP client with DPoP authentication
- ✅ GET, PUT, DELETE, HEAD operations
- ✅ Container (directory) creation
- ✅ DPoP token validation
- ✅ RDF serialization (Turtle and JSON-LD)
- ✅ Location data to RDF conversion

## Integration with Location Tracker

This PoC is intentionally isolated from the main location-tracker app. When ready for production integration:

1. The `internal/solid` package can be imported by location-tracker
2. API endpoints can be adapted for main app use
3. Frontend auth flow can be integrated into main UI

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

**Phase 2: Backend Integration** 🚧 In Progress
- Go Solid client library (this app)
- Server-side Pod operations
- RDF serialization

## Testing

### Prerequisites
1. A Solid Pod account (SolidCommunity.net or Inrupt PodSpaces)
2. Web browser with JavaScript enabled
3. HTTP server (not `file://` protocol)

### Test Flow
1. Start solid-poc: `go run main.go`
2. Navigate to `http://localhost:9090`
3. Follow the interactive authentication flow
4. Test Pod read/write operations
5. Verify RDF serialization

## Security Notes

- DPoP tokens are validated but NOT cryptographically verified (PoC only)
- Production should implement full JWT signature verification
- All Pod operations require DPoP authentication
- Frontend handles authentication, backend validates tokens

## Next Steps

After validating this PoC:
1. **Issue #50**: Integrate Solid endpoints into location-tracker
2. **Issue #51**: Create data storage abstraction layer
3. **Issue #52**: Implement production Pod operations

## Resources

- **Solid Protocol**: https://solidproject.org/TR/protocol
- **Inrupt Docs**: https://docs.inrupt.com/
- **RDF Data Models**: See `../SOLID_DATA_MODELS.md`
- **Authentication Research**: See `../SOLID_AUTHENTICATION.md`
- **Frontend PoC**: See `../proof-of-concept/README.md`

## License

Part of the Location Tracker project. See main repository for license information.
