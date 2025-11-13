# LinkedDoc+TTL Schema Reference

## Overview

The LinkedDoc+TTL schema is an RDF ontology that defines semantic relationships between code modules. It's defined in [`linkedoc_schema.ttl`](../../linkedoc_schema.ttl).

## Namespace

```turtle
@prefix code: <https://schema.codedoc.org/> .
```

All LinkedDoc classes and properties use the `code:` namespace.

## Classes

### code:Module

A source code module (file) containing related functionality.

**Usage:**
```turtle
<handlers/location.go> a code:Module ;
    code:name "handlers/location.go" ;
    code:description "HTTP handlers for location tracking" .
```

### code:Package

A collection of related modules grouped by functionality.

**Example:** `handlers/`, `services/`, `types/`

### code:Function

An exported function or method.

**Usage:**
```turtle
:HandleLocation a code:Function ;
    rdfs:label "HandleLocation" ;
    rdfs:comment "Handles POST /api/location" .
```

### code:Type

An exported data type, struct, or interface.

**Usage:**
```turtle
:Location a code:Type ;
    rdfs:label "Location" ;
    rdfs:comment "Represents a tracked location point" .
```

### code:Handler

A subclass of `code:Function` specifically for HTTP handlers.

**Usage:**
```turtle
:HandleLocation a code:Handler ;
    rdfs:label "HandleLocation" .
```

### code:Service

A business logic service containing orchestration and domain rules.

**Usage:**
```turtle
:LocationService a code:Service ;
    rdfs:label "LocationService" .
```

### code:Client

A client for interacting with external APIs.

**Usage:**
```turtle
:GooglePlacesClient a code:Client ;
    rdfs:label "GooglePlacesClient" .
```

## Properties

### Basic Information

#### code:name

The module name including package path.

- **Domain:** `code:Module`
- **Range:** `xsd:string`

**Example:** `"handlers/location.go"`

#### code:description

A concise human-readable description of the module's purpose.

- **Domain:** `code:Module`
- **Range:** `xsd:string`

**Example:** `"HTTP handlers for location tracking endpoints"`

#### code:packageName

The Go package name for this module.

- **Domain:** `code:Module`
- **Range:** `xsd:string`

**Example:** `"handlers"`

### Relationships

#### code:dependsOn

A direct dependency where this module imports and uses another module.

- **Domain:** `code:Module`
- **Range:** `code:Module`

**Usage:**
```turtle
<handlers/location.go> code:dependsOn <types/location.go> .
```

**When to use:** When your module has an `import` statement and directly calls functions or uses types from another module.

#### code:integratesWith

An integration relationship where modules work together but aren't direct dependencies.

- **Domain:** `code:Module`
- **Range:** `code:Module`

**Usage:**
```turtle
<handlers/location.go> code:integratesWith <services/business_search.go> .
```

**When to use:** When modules collaborate but the dependency is indirect (e.g., through interfaces or dependency injection).

#### code:uses

A usage relationship indicating this module utilizes functionality from another.

- **Domain:** `code:Module`
- **Range:** `code:Module`

**Usage:**
```turtle
<services/location.go> code:uses <storage/cache.go> .
```

**When to use:** For looser coupling where one module uses another's functionality without a hard import dependency.

#### code:implements

Indicates this module implements a specific interface or type.

- **Domain:** `code:Module`
- **Range:** `code:Type`

**Usage:**
```turtle
<storage/dynamodb.go> code:implements :DataStore .
```

#### code:calledBy

Inverse relationship showing which modules call this module.

- **Domain:** `code:Module`
- **Range:** `code:Module`

**Usage:**
```turtle
<types/location.go> code:calledBy <handlers/location.go> .
```

**Note:** Usually inferred automatically from `code:dependsOn` relationships.

### Exports

#### code:exports

Generic property for any exported symbol.

- **Domain:** `code:Module`

**Usage:**
```turtle
<handlers/location.go> code:exports :HandleLocation, :HandleLocationByID .
```

#### code:exportsFunction

A function exported by this module (subproperty of `code:exports`).

- **Range:** `code:Function`

#### code:exportsType

A type or struct exported by this module (subproperty of `code:exports`).

- **Range:** `code:Type`

#### code:exportsHandler

An HTTP handler exported by this module (subproperty of `code:exportsFunction`).

- **Range:** `code:Handler`

### Semantic Tags

#### code:tags

Semantic tags for categorizing the module.

- **Domain:** `code:Module`
- **Range:** `xsd:string`

**Usage:**
```turtle
<handlers/location.go> code:tags "http", "location", "api" .
```

**See:** [Tag Taxonomy](#tag-taxonomy) below

#### code:category

Primary functional category.

- **Domain:** `code:Module`
- **Range:** `xsd:string`

**Values:** `"handlers"`, `"services"`, `"types"`, `"storage"`, `"clients"`, `"middleware"`

### Metadata

#### code:lastUpdated

Timestamp of last modification.

- **Domain:** `code:Module`
- **Range:** `xsd:dateTime`

**Usage:**
```turtle
<handlers/location.go> code:lastUpdated "2025-11-13T12:00:00Z"^^xsd:dateTime .
```

#### code:linesOfCode

Total lines of code in this module.

- **Domain:** `code:Module`
- **Range:** `xsd:integer`

**Usage:**
```turtle
<handlers/location.go> code:linesOfCode 150 .
```

#### code:author

Original author or maintainer of this module.

- **Domain:** `code:Module`
- **Range:** `xsd:string`

#### code:version

Version number or identifier for this module.

- **Domain:** `code:Module`
- **Range:** `xsd:string`

### Quality Metrics

#### code:complexity

Cyclomatic complexity score (0.0-1.0, lower is better).

- **Domain:** `code:Module`
- **Range:** `xsd:decimal`

**Usage:**
```turtle
<services/complex_logic.go> code:complexity 0.7 .
```

#### code:testCoverage

Test coverage percentage (0.0-1.0).

- **Domain:** `code:Module`
- **Range:** `xsd:decimal`

**Usage:**
```turtle
<handlers/location.go> code:testCoverage 0.85 .
```

#### code:hasTests

Whether this module has associated test files.

- **Domain:** `code:Module`
- **Range:** `xsd:boolean`

**Usage:**
```turtle
<handlers/location.go> code:hasTests true .
```

## Tag Taxonomy

### Functional Tags

- **`http`** - HTTP handlers and endpoints
- **`storage`** - Data persistence operations
- **`api-client`** - External API integration clients
- **`business-logic`** - Core business rules and orchestration
- **`data-types`** - Data structures and domain models
- **`middleware`** - HTTP middleware functions
- **`auth`** - Authentication and authorization logic
- **`validation`** - Input validation and sanitization

### Domain Tags

- **`location`** - Location tracking and geolocation features
- **`errors`** - Error logging and reporting
- **`tips`** - Anonymous tips feature
- **`commercial`** - Commercial real estate features
- **`social`** - Social sharing and integration
- **`cryptogram`** - Cryptogram puzzle features

### Technical Tags

- **`dynamodb`** - DynamoDB database operations
- **`cache`** - Caching logic and operations
- **`rate-limiting`** - Rate limiting and throttling
- **`websocket`** - WebSocket connections
- **`rdf`** - RDF and semantic data operations
- **`solid`** - Solid Protocol integration

## Complete Example

```turtle
@prefix code: <https://schema.codedoc.org/> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .

<handlers/location.go> a code:Module ;
    code:name "handlers/location.go" ;
    code:description "HTTP handlers for location tracking endpoints" ;
    code:packageName "handlers" ;
    code:category "handlers" ;

    # Dependencies
    code:dependsOn <types/location.go>, <storage/dynamodb.go> ;
    code:integratesWith <services/business_search.go> ;

    # Exports
    code:exports :HandleLocation, :HandleLocationByID ;
    code:exportsHandler :HandleLocation, :HandleLocationByID ;

    # Tags
    code:tags "http", "location", "api" ;

    # Metadata
    code:linesOfCode 150 ;
    code:lastUpdated "2025-11-13T12:00:00Z"^^xsd:dateTime ;
    code:author "Development Team" ;

    # Quality
    code:testCoverage 0.85 ;
    code:hasTests true ;
    code:complexity 0.4 .

# Exported handlers
:HandleLocation a code:Handler ;
    rdfs:label "HandleLocation" ;
    rdfs:comment "Handles POST /api/location for creating location records" .

:HandleLocationByID a code:Handler ;
    rdfs:label "HandleLocationByID" ;
    rdfs:comment "Handles GET /api/location/:id for retrieving location by ID" .
```

## SPARQL Query Examples

Once you have multiple modules documented, you can query the RDF graph:

### Find all modules that depend on types/location.go

```sparql
PREFIX code: <https://schema.codedoc.org/>

SELECT ?module ?description
WHERE {
    ?module code:dependsOn <types/location.go> ;
            code:description ?description .
}
```

### Find modules with low test coverage

```sparql
PREFIX code: <https://schema.codedoc.org/>

SELECT ?module ?coverage
WHERE {
    ?module a code:Module ;
            code:testCoverage ?coverage .
    FILTER (?coverage < 0.5)
}
ORDER BY ?coverage
```

### Find all HTTP handlers

```sparql
PREFIX code: <https://schema.codedoc.org/>

SELECT ?module ?handler
WHERE {
    ?module code:tags "http" ;
            code:exports ?handler .
}
```

## Extending the Schema

To add new predicates or classes:

1. Edit `linkedoc_schema.ttl`
2. Define the new class/property with:
   - `rdfs:label` - Human-readable name
   - `rdfs:comment` - Description
   - `rdfs:domain` - What it applies to
   - `rdfs:range` - What values it accepts
3. Document it in this file
4. Update examples
5. Run validation tests

## Validation

The schema is validated using standard RDF/Turtle validators. The build tool (`linkedoc_build.go`) enforces:

1. Valid Turtle syntax
2. Use of defined predicates only
3. Correct property domains and ranges
4. Required properties present

## Related

- [LinkedDoc README](README.md)
- [Best Practices](best-practices.md)
- [Examples](examples/)
