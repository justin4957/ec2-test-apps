# Claude Code Instructions for EC2 Test Apps

## Deployment Process

**IMPORTANT: Always use the deployment script first!**

When deploying any changes to EC2:

1. **First, always try the deploy script:**
   ```bash
   cd /Users/coolbeans/Development/dev/ec2-test-apps
   ./deploy-to-ec2.sh
   ```

2. **Only if the deploy script fails**, then try alternative methods like:
   - Direct SSH access
   - AWS SSM commands
   - Manual docker commands

The `deploy-to-ec2.sh` script handles:
- ECR login
- Pulling latest images
- Stopping and removing old containers
- Starting new containers with proper configuration
- Network setup
- Environment variable management
- Health checks and validation

## Building Containers

When building containers before deployment:

1. **Build location-tracker:**
   ```bash
   cd location-tracker
   docker buildx build --platform linux/amd64 -t 310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest --push .
   ```

2. **Build error-generator:**
   ```bash
   cd error-generator
   docker buildx build --platform linux/amd64 -t 310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest --push .
   ```

3. **Build slogan-server:**
   ```bash
   cd slogan-server
   docker buildx build --platform linux/amd64 -t 310829530225.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest --push .
   ```

After building, always run `./deploy-to-ec2.sh` to deploy the changes.

## Project Structure

- `location-tracker/` - Go application for location tracking with Google Maps integration
- `error-generator/` - Go application that generates error logs with GIFs, songs, and stories
- `slogan-server/` - Go application for generating dynamic slogans
- `code-fix-generator/` - Python application for satirical code fixes using DeepSeek
- `nginx/` - Nginx reverse proxy configuration
- `deploy-to-ec2.sh` - Main deployment script (USE THIS FIRST!)
- `.env.ec2` - Environment variables for EC2 deployment

## Key Features

- Location tracking with simulated location support for authenticated users
- Google Maps autocomplete for location simulation
- Commercial real estate search with Perplexity API
- Governing bodies display in UI (city council, planning, zoning, civic orgs)
- Interactive fiction stories with Anthropic Claude
- Daily cryptogram puzzles
- SMS integration via Twilio
- Real-time error logs with GIFs and music

---

## File Size and Modularity Standards

**IMPORTANT: Keep files small and modular for AI navigation efficiency.**

### File Size Limits

- **Hard Limit**: No file should exceed 500 lines of code
- **Soft Limit**: Prefer files under 300 lines
- **Warning Threshold**: Files over 400 lines should be reviewed for splitting
- **Token Consideration**: Large files (>2000 tokens) are hard for AI to navigate

### When to Split a File

Split a file when:
1. File exceeds 400 lines
2. Multiple unrelated responsibilities exist
3. Circular dependencies are forming
4. Testing becomes difficult
5. AI tools struggle to understand the file context

### How to Split Files

**Package Structure:**
```
app/
├── main.go (routing & initialization only, <200 lines)
├── types/           # Data structures
├── handlers/        # HTTP request handlers
├── services/        # Business logic
├── storage/         # Data persistence
├── clients/         # External API clients
├── middleware/      # HTTP middleware
└── internal/        # Internal utilities
```

**Refactoring Strategy:**
1. Extract types to `types/` package
2. Extract handlers to `handlers/` package
3. Extract business logic to `services/` package
4. Extract storage operations to `storage/` package
5. Keep `main.go` for routing and initialization only

---

## LinkedDoc+TTL Documentation Standards

**All Go modules must include a LinkedDoc header for AI navigation and semantic linking.**

### LinkedDoc Header Format

```go
/*
# Module: <package>/<filename>.go
<Brief one-line description>

## Linked Modules
- [<module-name>](<relative-path>): <relationship description>
- [<module-name>](<relative-path>): <relationship description>

## Tags
<tag1>, <tag2>, <tag3>

## Exports
<ExportedFunction>, <ExportedType>

<!-- LinkedDoc RDF -->
@prefix code: <https://schema.codedoc.org/> .
<this> a code:Module ;
    code:name "<package>/<filename>.go" ;
    code:description "<description>" ;
    code:dependsOn <relative-path> ;
    code:exports :<symbols> ;
    code:tags "<tags>" .
<!-- End LinkedDoc RDF -->
*/
package <package>
```

### Required Fields

- **Module name**: Full package/filename path
- **Description**: One-line summary of purpose
- **Linked Modules**: All direct dependencies with relationship descriptions
- **Tags**: Semantic tags from taxonomy (see below)
- **Exports**: Public functions and types
- **RDF metadata**: Machine-readable relationships

### Tag Taxonomy

**Functional Categories:**
- `http` - HTTP handlers/endpoints
- `storage` - Data persistence
- `api-client` - External API integration
- `business-logic` - Core business rules
- `data-types` - Data structures
- `middleware` - HTTP middleware
- `auth` - Authentication/authorization

**Domain Categories:**
- `location` - Location tracking
- `errors` - Error logging
- `tips` - Anonymous tips
- `commercial` - Commercial real estate
- `social` - Social sharing

**Technical Categories:**
- `dynamodb` - DynamoDB operations
- `cache` - Caching logic
- `rate-limiting` - Rate limiting
- `websocket` - WebSocket connections

### LinkedDoc Example

```go
/*
# Module: handlers/location.go
HTTP handlers for location tracking and retrieval endpoints.

## Linked Modules
- [types/location](../types/location.go): Location data structures
- [storage/dynamodb](../storage/dynamodb.go): Location persistence
- [services/business_search](../services/business_search.go): Nearby business discovery

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
    code:integratesWith <../services/business_search.go> ;
    code:exports :HandleLocation, :HandleLocationByID ;
    code:tags "http", "location", "api" .
<!-- End LinkedDoc RDF -->
*/
package handlers

import (
    "github.com/justin4957/ec2-test-apps/location-tracker/types"
    "github.com/justin4957/ec2-test-apps/location-tracker/storage"
)

// HandleLocation handles POST /api/location
func HandleLocation(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### LinkedDoc Best Practices

1. **Keep descriptions concise** - One line maximum
2. **Link all dependencies** - Include both direct and integration dependencies
3. **Use semantic tags** - Follow the taxonomy
4. **Update on changes** - When dependencies change, update links
5. **Validate regularly** - Run `go run tools/linkedoc_build.go --validate` before commits

---

## Code Organization Guidelines

### Package Responsibilities

- **types/** - Data structures and domain models only
- **handlers/** - HTTP handlers (thin layer: validation + service calls)
- **services/** - Business logic and orchestration
- **storage/** - Data persistence (DynamoDB, cache, etc.)
- **clients/** - External API clients (Google, Perplexity, Twilio, etc.)
- **middleware/** - HTTP middleware (auth, rate limiting, logging)
- **internal/** - Internal utilities not for external use

### Dependency Flow

```
handlers → services → storage
    ↓
  clients
```

- **Handlers** should be thin (validation + service call)
- **Services** contain business logic
- **Storage** handles persistence only
- **No circular dependencies** allowed

### main.go Responsibilities

main.go should ONLY contain:
- Route registration
- Server initialization
- Environment variable loading
- Graceful shutdown

**All other logic belongs in packages.**

**Example main.go (target structure):**
```go
package main

import (
    "github.com/justin4957/ec2-test-apps/location-tracker/handlers"
    "github.com/justin4957/ec2-test-apps/location-tracker/services"
    "github.com/justin4957/ec2-test-apps/location-tracker/storage"
)

func main() {
    // Load config
    config := loadConfig()

    // Initialize storage
    db := storage.NewDynamoDB(config.DynamoDBTable)

    // Initialize services
    locationService := services.NewLocationService(db)

    // Initialize handlers
    locationHandler := handlers.NewLocationHandler(locationService)

    // Register routes
    http.HandleFunc("/api/location", locationHandler.HandleLocation)
    http.HandleFunc("/api/health", handlers.HandleHealth)

    // Start server
    log.Fatal(http.ListenAndServe(":5000", nil))
}
```

---

## AI Code Navigation (For Claude Code)

### Using LinkedDoc for Navigation

1. **Check JSON index first**: Always check `docs/linkedoc_index.json` for module overview
2. **Read LinkedDoc headers**: Understand dependencies without reading full files
3. **Follow semantic links**: Use links in headers to discover related code
4. **Read targeted modules**: Only read specific modules needed for the task
5. **Validate before changes**: Run link validation before suggesting refactoring

### Token Efficiency Strategy

- **Small modules** = fewer tokens per context read
- **JSON index** provides overview without reading all files
- **LinkedDoc headers** guide navigation without full file reads
- **Modularity** enables targeted analysis

### When Refactoring Code

1. Check existing LinkedDoc headers for dependencies
2. Maintain link relationships when moving code
3. Update RDF metadata after changes
4. Run validation: `go run tools/linkedoc_build.go --validate`
5. Regenerate JSON index: `go run tools/linkedoc_build.go --generate-index`

### Validation Commands

```bash
# Validate all LinkedDoc headers and links
go run tools/linkedoc_build.go --validate

# Generate AI-optimized JSON index
go run tools/linkedoc_build.go --generate-index

# Check specific package
go run tools/linkedoc_build.go --path location-tracker/handlers

# Incremental build (only changed files)
go run tools/linkedoc_build.go --incremental
```

---
