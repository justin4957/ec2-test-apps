# Solid RDF Data Models for Location Tracker

**Issue:** #47
**Phase:** Phase 1.3 - Foundation & Research
**Status:** ✅ Complete
**Dependencies:** None (parallel with #45, #46)

---

## Table of Contents

1. [Overview](#overview)
2. [RDF Basics](#rdf-basics)
3. [Vocabulary Selection](#vocabulary-selection)
4. [Location Data Model](#location-data-model)
5. [Error Log Data Model](#error-log-data-model)
6. [Commercial Real Estate Data Model](#commercial-real-estate-data-model)
7. [Tips and Donations Data Model](#tips-and-donations-data-model)
8. [Pod Container Structure](#pod-container-structure)
9. [Serialization Formats](#serialization-formats)
10. [Validation](#validation)
11. [Implementation Guide](#implementation-guide)

---

## Overview

This document defines RDF (Resource Description Framework) schemas for all Location Tracker data types that will be stored in users' Solid Pods.

### Design Principles

1. **Use Standard Vocabularies** - Prefer schema.org, FOAF, Dublin Core over custom ontologies
2. **Interoperability** - Data should be usable by other Solid apps
3. **Human & Machine Readable** - Clear predicates and meaningful URIs
4. **Extensible** - Easy to add fields without breaking existing data
5. **Linked Data** - Reference external resources via URIs

### Data Types Covered

| Data Type | Format | Vocabulary | Container |
|-----------|--------|------------|-----------|
| Location | Turtle | schema.org, geo | `/locations/` |
| Error Log | JSON-LD | schema.org | `/error-logs/` |
| Commercial Real Estate | Turtle | schema.org, vcard | `/commercial/` |
| Tips | JSON-LD | schema.org, gr | `/tips/` |
| Preferences | Turtle | solid, pim | `/preferences.ttl` |

---

## RDF Basics

### Triple Structure

RDF data consists of triples: **Subject → Predicate → Object**

```turtle
<#me> foaf:name "Alice Smith" .
  ↑       ↑            ↑
Subject Predicate    Object
```

### URIs and Prefixes

**Full URI:**
```turtle
<https://alice.solidcommunity.net/private/location-tracker/locations/2025/11/location-001.ttl#location>
```

**With Prefix:**
```turtle
@prefix : <https://alice.solidcommunity.net/private/location-tracker/locations/2025/11/location-001.ttl#> .

:location a schema:Place .
```

### Common Prefixes

```turtle
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix schema: <http://schema.org/> .
@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .
@prefix foaf: <http://xmlns.com/foaf/0.1/> .
@prefix dcterms: <http://purl.org/dc/terms/> .
@prefix solid: <http://www.w3.org/ns/solid/terms#> .
```

---

## Vocabulary Selection

### Primary Vocabularies

#### 1. schema.org
**URL:** https://schema.org/
**Purpose:** General-purpose structured data
**Used For:** Places, Events, Reports, Organizations

**Key Classes:**
- `schema:Place` - Geographic locations
- `schema:GeoCoordinates` - Latitude/longitude
- `schema:Report` - Error logs, reports
- `schema:ImageObject` - GIF images
- `schema:MusicRecording` - Songs
- `schema:MonetaryAmount` - Tips, donations

#### 2. WGS84 Geo Positioning (geo:)
**URL:** http://www.w3.org/2003/01/geo/wgs84_pos#
**Purpose:** Geographic coordinates
**Used For:** Precise location data

**Key Properties:**
- `geo:lat` - Latitude (decimal degrees)
- `geo:long` - Longitude (decimal degrees)
- `geo:alt` - Altitude (meters)

#### 3. FOAF (Friend of a Friend)
**URL:** http://xmlns.com/foaf/0.1/
**Purpose:** Social graph and identity
**Used For:** User profiles, relationships

#### 4. Dublin Core Terms (dcterms:)
**URL:** http://purl.org/dc/terms/
**Purpose:** Metadata
**Used For:** Created dates, creators, descriptions

#### 5. Solid Terms
**URL:** http://www.w3.org/ns/solid/terms#
**Purpose:** Solid-specific metadata
**Used For:** App registration, permissions

### Why These Vocabularies?

| Requirement | Vocabulary | Reason |
|-------------|-----------|---------|
| Geo coordinates | geo: | W3C standard, widely used |
| Business info | schema.org | Google-endorsed, rich semantics |
| Metadata | dcterms: | Standard for creation dates, authors |
| Interoperability | schema.org | Most Solid apps understand |
| Extensibility | schema.org | Easy to add custom properties |

---

## Location Data Model

### Conceptual Model

```
Location
├── Geographic Coordinates (lat, lng, accuracy)
├── Timestamp (when recorded)
├── Device Information (device ID)
├── Address (optional, reverse geocoded)
├── Metadata (created by, app version)
└── Simulation Flag (if simulated location)
```

### Turtle Schema

**File:** `rdf-schemas/examples/location-example.ttl`

```turtle
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix schema: <http://schema.org/> .
@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .
@prefix dcterms: <http://purl.org/dc/terms/> .
@prefix : <#> .

:location a schema:Place, geo:Point ;
    # Geographic Coordinates
    geo:lat "37.7749"^^xsd:decimal ;
    geo:long "-122.4194"^^xsd:decimal ;
    schema:geo [
        a schema:GeoCoordinates ;
        schema:latitude 37.7749 ;
        schema:longitude -122.4194 ;
    ] ;

    # Accuracy (meters)
    schema:accuracy "10.5"^^xsd:decimal ;

    # Timestamp
    dcterms:created "2025-11-12T18:30:00Z"^^xsd:dateTime ;
    schema:dateCreated "2025-11-12T18:30:00Z"^^xsd:dateTime ;

    # Device Information
    schema:identifier "device-abc123" ;
    schema:device [
        a schema:Thing ;
        schema:identifier "device-abc123" ;
        schema:name "iPhone 13" ;
    ] ;

    # Address (optional, from reverse geocoding)
    schema:address [
        a schema:PostalAddress ;
        schema:streetAddress "123 Market St" ;
        schema:addressLocality "San Francisco" ;
        schema:addressRegion "CA" ;
        schema:postalCode "94105" ;
        schema:addressCountry "US" ;
    ] ;

    # Metadata
    dcterms:creator <https://alice.solidcommunity.net/profile/card#me> ;
    schema:creator <https://alice.solidcommunity.net/profile/card#me> ;
    schema:isAccessibleForFree true ;

    # Application Info
    schema:applicationCategory "LocationTracking" ;
    schema:applicationSubCategory "location-tracker" ;
    schema:softwareVersion "2.0.0" ;

    # Simulation flag (if location was simulated)
    schema:additionalProperty [
        a schema:PropertyValue ;
        schema:propertyID "simulated" ;
        schema:value "false"^^xsd:boolean ;
    ] ;

    # Google Maps URL (if available)
    schema:hasMap "https://www.google.com/maps?q=37.7749,-122.4194" .
```

### Field Mapping

| Go Struct Field | RDF Property | Datatype | Required |
|----------------|--------------|----------|----------|
| `Latitude` | `geo:lat`, `schema:latitude` | xsd:decimal | ✅ |
| `Longitude` | `geo:long`, `schema:longitude` | xsd:decimal | ✅ |
| `Accuracy` | `schema:accuracy` | xsd:decimal | ✅ |
| `Timestamp` | `dcterms:created`, `schema:dateCreated` | xsd:dateTime | ✅ |
| `DeviceID` | `schema:identifier` | xsd:string | ✅ |
| `Address` | `schema:address` | schema:PostalAddress | ⚠️ |
| `Simulated` | `schema:additionalProperty` | xsd:boolean | ⚠️ |

---

## Error Log Data Model

### Conceptual Model

```
ErrorLog
├── Error Information (message, type, severity)
├── Timestamp (when occurred)
├── Location Context (where it happened)
├── Media (GIF, meme, screenshot)
├── Music (associated song)
├── Story (children's story or satire)
├── Device Information
└── Metadata (created by, app)
```

### JSON-LD Schema

**File:** `rdf-schemas/examples/errorlog-example.jsonld`

```json
{
  "@context": {
    "@vocab": "http://schema.org/",
    "geo": "http://www.w3.org/2003/01/geo/wgs84_pos#",
    "dcterms": "http://purl.org/dc/terms/",
    "xsd": "http://www.w3.org/2001/XMLSchema#"
  },
  "@type": "Report",
  "@id": "#error-log",

  "name": "Database Connection Timeout",
  "description": "Connection to PostgreSQL database timed out after 30 seconds",
  "text": "Full error stack trace: postgresql.connect() timeout exceeded...",

  "dateCreated": {
    "@type": "DateTime",
    "@value": "2025-11-12T18:30:00Z"
  },

  "reportNumber": "ERR-20251112-183000-ABC123",

  "category": "DatabaseError",
  "additionalType": "https://schema.org/TechArticle",

  "about": {
    "@type": "SoftwareApplication",
    "name": "Location Tracker",
    "applicationCategory": "UtilitiesApplication",
    "softwareVersion": "2.0.0"
  },

  "location": {
    "@type": "Place",
    "geo": {
      "@type": "GeoCoordinates",
      "latitude": 37.7749,
      "longitude": -122.4194
    },
    "address": {
      "@type": "PostalAddress",
      "addressLocality": "San Francisco",
      "addressRegion": "CA",
      "addressCountry": "US"
    }
  },

  "associatedMedia": [
    {
      "@type": "ImageObject",
      "contentUrl": "https://media.giphy.com/media/error-frustrated/giphy.gif",
      "encodingFormat": "image/gif",
      "name": "Frustrated Computer User GIF",
      "description": "Someone throwing computer out window",
      "thumbnail": "https://media.giphy.com/media/error-frustrated/200.gif"
    },
    {
      "@type": "ImageObject",
      "contentUrl": "https://error-generator-memes.s3.amazonaws.com/meme-12345.png",
      "encodingFormat": "image/png",
      "name": "Database Timeout Meme",
      "description": "Absurdist meme about database connections",
      "creator": {
        "@type": "SoftwareApplication",
        "name": "Vertex AI Imagen"
      }
    }
  ],

  "audio": {
    "@type": "MusicRecording",
    "name": "Bohemian Rhapsody",
    "byArtist": {
      "@type": "MusicGroup",
      "name": "Queen"
    },
    "url": "https://open.spotify.com/track/7tFiyTwD0nx5a1eklYtX2J",
    "isrcCode": "GBUM71029604",
    "duration": "PT5M55S"
  },

  "comment": [
    {
      "@type": "Comment",
      "text": "Once upon a time in Database Land, a little query got lost...",
      "name": "Children's Story",
      "author": {
        "@type": "SoftwareApplication",
        "name": "Anthropic Claude"
      },
      "commentTime": "2025-11-12T18:30:05Z"
    },
    {
      "@type": "Comment",
      "text": "// TODO: Replace database with blockchain and AI\n// This will definitely fix the timeout",
      "name": "Satirical Code Fix",
      "author": {
        "@type": "SoftwareApplication",
        "name": "DeepSeek Coder"
      },
      "commentTime": "2025-11-12T18:30:10Z"
    }
  ],

  "provider": {
    "@type": "Organization",
    "name": "Location Tracker Error Generator",
    "url": "https://notspies.org"
  },

  "creator": "https://alice.solidcommunity.net/profile/card#me",

  "identifier": {
    "@type": "PropertyValue",
    "propertyID": "device_id",
    "value": "device-abc123"
  },

  "additionalProperty": [
    {
      "@type": "PropertyValue",
      "propertyID": "error_type",
      "value": "TIMEOUT"
    },
    {
      "@type": "PropertyValue",
      "propertyID": "severity",
      "value": "HIGH"
    },
    {
      "@type": "PropertyValue",
      "propertyID": "user_hash",
      "value": "hash-xyz789"
    },
    {
      "@type": "PropertyValue",
      "propertyID": "cspan_video_url",
      "value": "https://www.c-span.org/video/?..."
    }
  ]
}
```

### Field Mapping

| Go Struct Field | RDF Property | Datatype | Required |
|----------------|--------------|----------|----------|
| `ErrorMessage` | `name`, `description` | xsd:string | ✅ |
| `Timestamp` | `dateCreated` | xsd:dateTime | ✅ |
| `Location` | `location` | schema:Place | ⚠️ |
| `GifURL` | `associatedMedia[0].contentUrl` | xsd:anyURI | ⚠️ |
| `MemeURL` | `associatedMedia[1].contentUrl` | xsd:anyURI | ⚠️ |
| `Song` | `audio` | schema:MusicRecording | ⚠️ |
| `Story` | `comment[0].text` | xsd:string | ⚠️ |
| `SatiricalFix` | `comment[1].text` | xsd:string | ⚠️ |
| `DeviceID` | `identifier.value` | xsd:string | ✅ |

---

## Commercial Real Estate Data Model

### Turtle Schema

**File:** `rdf-schemas/examples/commercial-realestate-example.ttl`

```turtle
@prefix schema: <http://schema.org/> .
@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .
@prefix vcard: <http://www.w3.org/2006/vcard/ns#> .
@prefix : <#> .

:property a schema:RealEstateListing, schema:Place ;
    schema:name "Office Space - Downtown SF" ;
    schema:description "Modern office space with bay views" ;

    # Location
    schema:geo [
        a schema:GeoCoordinates ;
        schema:latitude 37.7899 ;
        schema:longitude -122.3988 ;
    ] ;
    schema:address [
        a schema:PostalAddress ;
        schema:streetAddress "100 California St" ;
        schema:addressLocality "San Francisco" ;
        schema:addressRegion "CA" ;
        schema:postalCode "94111" ;
        schema:addressCountry "US" ;
    ] ;

    # Property Details
    schema:floorSize [
        a schema:QuantitativeValue ;
        schema:value 5000 ;
        schema:unitCode "FTK" ; # square feet
    ] ;
    schema:numberOfRooms 8 ;

    # Pricing
    schema:priceRange "$$$$" ;
    schema:price [
        a schema:PriceSpecification ;
        schema:price 12000 ;
        schema:priceCurrency "USD" ;
        schema:unitText "monthly" ;
    ] ;

    # Business Type
    schema:additionalType "https://schema.org/OfficeOrBusinessPlace" ;

    # Contact
    schema:telephone "+1-415-555-1234" ;
    schema:email "leasing@example.com" ;
    schema:url "https://example.com/listings/12345" ;

    # Metadata
    schema:datePublished "2025-11-12"^^xsd:date ;
    schema:dateModified "2025-11-12"^^xsd:date ;

    # Query Context (from Perplexity)
    schema:keywords "commercial real estate, office space, downtown" ;
    schema:inLanguage "en-US" .
```

---

## Tips and Donations Data Model

### JSON-LD Schema

**File:** `rdf-schemas/examples/tip-example.jsonld`

```json
{
  "@context": {
    "@vocab": "http://schema.org/",
    "gr": "http://purl.org/goodrelations/v1#"
  },
  "@type": "DonateAction",
  "@id": "#tip",

  "agent": "https://alice.solidcommunity.net/profile/card#me",

  "recipient": {
    "@type": "Organization",
    "name": "Location Tracker Development",
    "url": "https://notspies.org"
  },

  "object": {
    "@type": "MonetaryAmount",
    "currency": "USD",
    "value": 5.00
  },

  "startTime": "2025-11-12T18:30:00Z",
  "endTime": "2025-11-12T18:30:05Z",

  "paymentMethod": {
    "@type": "PaymentMethod",
    "name": "Stripe",
    "identifier": "ch_3abc123"
  },

  "description": "Thank you for this amazing app!",

  "isPartOf": {
    "@type": "Event",
    "name": "Monthly Support",
    "startDate": "2025-11-01"
  }
}
```

---

## Pod Container Structure

### Recommended Hierarchy

```
/private/
  location-tracker/
    locations/
      2025/
        11/
          12/
            location-2025-11-12T183000Z.ttl
            location-2025-11-12T184500Z.ttl
          13/
            location-2025-11-13T093000Z.ttl
        12/
          ...
    error-logs/
      2025/
        11/
          12/
            error-2025-11-12T183000Z.jsonld
            error-2025-11-12T190000Z.jsonld
    commercial/
      2025/
        11/
          commercial-sf-downtown.ttl
          commercial-oakland-uptown.ttl
    tips/
      2025/
        11/
          tip-2025-11-12T183000Z.jsonld
    preferences.ttl
    settings.jsonld

/public/
  location-tracker/
    shared-locations/
      (shared with friends)
```

### Container Metadata

**`.meta` file for each container:**

```turtle
@prefix acl: <http://www.w3.org/ns/auth/acl#> .
@prefix ldp: <http://www.w3.org/ns/ldp#> .

<> a ldp:BasicContainer ;
    acl:accessControl </private/location-tracker/.acl> .
```

### Access Control (ACL)

**`/private/location-tracker/.acl`:**

```turtle
@prefix acl: <http://www.w3.org/ns/auth/acl#> .

<#owner>
    a acl:Authorization ;
    acl:agent <https://alice.solidcommunity.net/profile/card#me> ;
    acl:accessTo </private/location-tracker/> ;
    acl:default </private/location-tracker/> ;
    acl:mode acl:Read, acl:Write, acl:Control .

<#appAccess>
    a acl:Authorization ;
    acl:origin <https://notspies.org> ;
    acl:accessTo </private/location-tracker/> ;
    acl:default </private/location-tracker/> ;
    acl:mode acl:Read, acl:Write .
```

---

## Serialization Formats

### Turtle (.ttl)

**Pros:**
- ✅ Human-readable
- ✅ Compact
- ✅ Native RDF format
- ✅ Good for simple data

**Cons:**
- ❌ Less familiar to developers
- ❌ Harder to parse than JSON

**Best For:** Locations, commercial real estate, preferences

### JSON-LD (.jsonld)

**Pros:**
- ✅ JSON-compatible
- ✅ Familiar to developers
- ✅ Easy to parse
- ✅ Good for complex nested data

**Cons:**
- ❌ More verbose than Turtle
- ❌ Requires @context

**Best For:** Error logs, tips, complex objects

### Format Selection Guide

| Data Type | Format | Reason |
|-----------|--------|--------|
| Location | Turtle | Simple, flat structure |
| Error Log | JSON-LD | Complex, nested media |
| Commercial | Turtle | Structured, repeating |
| Tips | JSON-LD | Payment metadata |
| Preferences | Turtle | Simple key-value |

---

## Validation

### SHACL Shapes

**Location Shape:**

```turtle
@prefix sh: <http://www.w3.org/ns/shacl#> .
@prefix schema: <http://schema.org/> .
@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .

:LocationShape a sh:NodeShape ;
    sh:targetClass schema:Place ;

    sh:property [
        sh:path geo:lat ;
        sh:datatype xsd:decimal ;
        sh:minInclusive -90 ;
        sh:maxInclusive 90 ;
        sh:minCount 1 ;
        sh:maxCount 1 ;
    ] ;

    sh:property [
        sh:path geo:long ;
        sh:datatype xsd:decimal ;
        sh:minInclusive -180 ;
        sh:maxInclusive 180 ;
        sh:minCount 1 ;
        sh:maxCount 1 ;
    ] ;

    sh:property [
        sh:path dcterms:created ;
        sh:datatype xsd:dateTime ;
        sh:minCount 1 ;
        sh:maxCount 1 ;
    ] .
```

### Validation Tools

**Command Line:**
```bash
# Validate Turtle syntax
rapper -i turtle -o turtle location.ttl

# Validate SHACL constraints
pyshacl -s shapes.ttl -d location.ttl
```

**Online:**
- Turtle Validator: http://ttl.summerofcode.be/
- JSON-LD Playground: https://json-ld.org/playground/

---

## Implementation Guide

### Go Struct to RDF Mapping

```go
type Location struct {
    Latitude  float64   `rdf:"geo:lat,http://www.w3.org/2003/01/geo/wgs84_pos#lat"`
    Longitude float64   `rdf:"geo:long,http://www.w3.org/2003/01/geo/wgs84_pos#long"`
    Accuracy  float64   `rdf:"schema:accuracy,http://schema.org/accuracy"`
    Timestamp time.Time `rdf:"dcterms:created,http://purl.org/dc/terms/created"`
    DeviceID  string    `rdf:"schema:identifier,http://schema.org/identifier"`
    Simulated bool      `rdf:"lt:simulated,http://notspies.org/ns#simulated"`
}
```

### Serialization Functions

```go
func LocationToTurtle(loc Location) (string, error) {
    template := `@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .
@prefix schema: <http://schema.org/> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix dcterms: <http://purl.org/dc/terms/> .

<#location> a schema:Place, geo:Point ;
    geo:lat "%.6f"^^xsd:decimal ;
    geo:long "%.6f"^^xsd:decimal ;
    schema:accuracy "%.2f"^^xsd:decimal ;
    dcterms:created "%s"^^xsd:dateTime ;
    schema:identifier "%s" .
`
    return fmt.Sprintf(template,
        loc.Latitude,
        loc.Longitude,
        loc.Accuracy,
        loc.Timestamp.Format(time.RFC3339),
        loc.DeviceID,
    ), nil
}
```

### Parsing Functions

```go
func TurtleToLocation(data string) (Location, error) {
    graph := rdf.NewGraph()
    err := graph.Parse(strings.NewReader(data), "text/turtle")
    if err != nil {
        return Location{}, err
    }

    loc := Location{}

    // Extract latitude
    latTriple := graph.Triple("<#location>", "http://www.w3.org/2003/01/geo/wgs84_pos#lat", nil)
    loc.Latitude, _ = strconv.ParseFloat(latTriple.Object.String(), 64)

    // Extract longitude
    longTriple := graph.Triple("<#location>", "http://www.w3.org/2003/01/geo/wgs84_pos#long", nil)
    loc.Longitude, _ = strconv.ParseFloat(longTriple.Object.String(), 64)

    // ... more fields

    return loc, nil
}
```

---

## Summary

### Data Models Defined

✅ **Location** - Turtle format with geo: and schema.org
✅ **Error Log** - JSON-LD with rich media metadata
✅ **Commercial Real Estate** - Turtle with vcard contact info
✅ **Tips/Donations** - JSON-LD with payment details

### Vocabularies Used

✅ **schema.org** - Primary vocabulary (80% of predicates)
✅ **geo:** - Geographic coordinates
✅ **dcterms:** - Metadata (dates, creators)
✅ **vcard:** - Contact information

### Container Structure

✅ Hierarchical organization by year/month/day
✅ Separate containers for each data type
✅ ACL configuration for access control

### Next Steps

1. **Issue #48:** Build PoC to test these schemas
2. **Issue #49:** Implement Go serialization/parsing
3. **Issue #52:** Write to actual Pods using these formats

---

**Status:** ✅ Complete
**Last Updated:** 2025-11-12
**Related Issues:** #47, #48, #49, #52
**Files:** See `rdf-schemas/examples/` directory
