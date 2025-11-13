# RDF Schemas and Examples

This directory contains RDF data model definitions and example files for the Location Tracker Solid integration.

## Contents

### Documentation
- [`SOLID_DATA_MODELS.md`](../SOLID_DATA_MODELS.md) - Complete data model specification

### Example Files

#### Locations (`examples/location-example.ttl`)
- **Format:** Turtle
- **Vocabulary:** schema.org + geo: (WGS84)
- **Use:** Tracking user location data with geographic coordinates

#### Error Logs (`examples/errorlog-example.jsonld`)
- **Format:** JSON-LD
- **Vocabulary:** schema.org
- **Use:** Application errors with media (GIFs, memes, songs, stories)

#### Commercial Real Estate (`examples/commercial-realestate-example.ttl`)
- **Format:** Turtle
- **Vocabulary:** schema.org
- **Use:** Business listings from Perplexity API queries

#### Tips/Donations (`examples/tip-example.jsonld`)
- **Format:** JSON-LD
- **Vocabulary:** schema.org + GoodRelations
- **Use:** Payment transactions via Stripe

## Quick Reference

### Vocabularies Used

| Prefix | Namespace | Purpose |
|--------|-----------|---------|
| `schema:` | http://schema.org/ | Primary vocabulary (80%) |
| `geo:` | http://www.w3.org/2003/01/geo/wgs84_pos# | Geographic coordinates |
| `dcterms:` | http://purl.org/dc/terms/ | Metadata (dates, creators) |
| `foaf:` | http://xmlns.com/foaf/0.1/ | Social connections |
| `xsd:` | http://www.w3.org/2001/XMLSchema# | Datatypes |

### Format Selection

- **Turtle (.ttl)** - Simple, flat structures (locations, real estate)
- **JSON-LD (.jsonld)** - Complex, nested structures (errors, payments)

## Validation

### Online Tools
- **Turtle:** http://ttl.summerofcode.be/
- **JSON-LD:** https://json-ld.org/playground/

### Command Line
```bash
# Validate Turtle syntax
rapper -i turtle -o turtle examples/location-example.ttl

# Validate JSON-LD
jsonld validate examples/errorlog-example.jsonld
```

## Usage in Code

### Go
```go
import "github.com/knakk/rdf"

// Parse Turtle
graph := rdf.NewGraph()
graph.Parse(reader, "text/turtle")

// Generate Turtle
turtle := LocationToTurtle(location)
```

### JavaScript
```javascript
import { parse } from '@rdfjs/parser-jsonld';

// Parse JSON-LD
const quads = await parse(jsonldData);
```

## Pod Storage Structure

```
/private/location-tracker/
├── locations/
│   └── 2025/11/12/
│       └── location-2025-11-12T183000Z.ttl
├── error-logs/
│   └── 2025/11/12/
│       └── error-2025-11-12T183000Z.jsonld
├── commercial/
│   └── commercial-sf-downtown.ttl
└── tips/
    └── tip-2025-11-12.jsonld
```

## Related Issues

- #47 - RDF Data Model Research
- #49 - Go Solid Client Library
- #52 - Pod Read/Write Operations

## Resources

- **schema.org:** https://schema.org/
- **RDF Primer:** https://www.w3.org/TR/rdf-primer/
- **Turtle Spec:** https://www.w3.org/TR/turtle/
- **JSON-LD Spec:** https://www.w3.org/TR/json-ld11/
