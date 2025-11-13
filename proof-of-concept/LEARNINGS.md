# Key Learnings from Solid Authentication PoC

This document captures important insights, gotchas, and best practices discovered while building the Solid authentication proof-of-concept.

## Table of Contents

1. [Authentication Flow](#authentication-flow)
2. [Session Management](#session-management)
3. [Pod Operations](#pod-operations)
4. [RDF Data Handling](#rdf-data-handling)
5. [Common Pitfalls](#common-pitfalls)
6. [Security Considerations](#security-considerations)
7. [Provider Differences](#provider-differences)
8. [Performance Insights](#performance-insights)

---

## Authentication Flow

### Key Insights

#### 1. OAuth Redirect Flow is Mandatory
- There's no "username/password" login option directly in the client library
- Must redirect to the Solid provider's login page
- Application loses state during redirect (must be restored)
- Cannot skip the redirect even for testing

**Implication**: Plan your UX around the redirect. Consider:
- Saving application state to localStorage before redirect
- Handling the callback URL gracefully
- Providing clear messaging about the redirect

#### 2. `handleIncomingRedirect()` Must Always Be Called
```javascript
await session.handleIncomingRedirect({
    restorePreviousSession: true
});
```

This must run on **every page load**, not just after redirect. It:
- Detects if returning from OAuth provider
- Restores previous authenticated sessions
- Completes the token exchange

**Gotcha**: If you forget this on page load, users will have to re-authenticate every time.

#### 3. DPoP is Handled Automatically
- No need to manually generate DPoP proof tokens
- Library creates ephemeral keypairs in the browser
- Keys stored in IndexedDB, never leave the browser
- Proofs automatically attached to every authenticated request

**Implication**: You don't need to implement RFC 9449 yourself, but you should understand it for troubleshooting.

#### 4. Client Registration is Dynamic
```javascript
await session.login({
    oidcIssuer: issuer,
    redirectUrl: window.location.href,
    clientName: 'Location Tracker PoC'
});
```

- No pre-registration required for most providers
- Client metadata registered dynamically on first login
- `clientName` is what users see in authorization screen
- `redirectUrl` must exactly match current URL (including query params)

**Gotcha**: In production, use a fixed redirect URL and handle routing appropriately.

---

## Session Management

### Key Insights

#### 1. Sessions Persist Across Page Loads
- Session info stored in localStorage and IndexedDB
- Tokens automatically refreshed when expired
- `restorePreviousSession: true` enables this

**Best Practice**: Always check `session.info.isLoggedIn` on page load.

#### 2. Multiple Sessions Not Supported by Default
- One session per browser/origin
- Logging in with different provider overwrites previous session
- For multi-account support, need separate browser profiles or custom storage

#### 3. Logout Clears All Local State
```javascript
await session.logout();
```

This:
- Clears localStorage
- Clears IndexedDB
- Does NOT revoke tokens at provider (most providers)
- Does NOT notify the server

**Implication**: For backend integration, track sessions separately and invalidate on logout.

#### 4. Token Expiration Handling
- Access tokens typically expire in 1-24 hours
- Library handles refresh automatically
- Refresh tokens can be valid for weeks/months
- No manual intervention needed

**Gotcha**: If refresh fails (e.g., provider down), user must re-authenticate.

---

## Pod Operations

### Key Insights

#### 1. Storage Location from WebID Profile
```javascript
const storage = solidClient.getUrl(profileThing, 'http://www.w3.org/ns/pim/space#storage');
```

- Every WebID must have `pim:storage` triple
- This points to the root of the user's storage
- Usually ends with trailing slash: `https://alice.solidcommunity.net/`
- All data operations relative to this URL

**Gotcha**: Some test Pods might not have `pim:storage` set. This will break write operations.

#### 2. Container Creation is Idempotent
```javascript
await solidClient.createContainerAt(containerUrl, { fetch: session.fetch });
```

- Safe to call even if container exists
- May throw error or succeed silently depending on provider
- Always wrap in try/catch
- Check container existence first if precision needed

**Best Practice**: Wrap in try/catch and log "created or exists" message.

#### 3. File URLs Must Include Extensions
```turtle
https://alice.solidcommunity.net/private/location-tracker/test-location.ttl
```

- File extension determines content-type handling
- `.ttl` for Turtle
- `.jsonld` for JSON-LD
- `.json` for plain JSON (not RDF)

**Gotcha**: Omitting extension may cause Pod to reject or mishandle file.

#### 4. Content-Type Header is Critical
```javascript
await solidClient.overwriteFile(
    locationUrl,
    new Blob([locationData], { type: 'text/turtle' }),
    {
        fetch: session.fetch,
        contentType: 'text/turtle'
    }
);
```

- Must match the actual data format
- Common types:
  - `text/turtle`
  - `application/ld+json`
  - `application/n-triples`
- Mismatched content-type causes parse errors

#### 5. Private vs Public Containers
- `/private/` - Only owner can access (default for sensitive data)
- `/public/` - World-readable
- Custom ACLs can be set per container/file

**Security Note**: Always use `/private/` for location data unless explicitly shared.

#### 6. Authenticated Fetch is Required
```javascript
const dataset = await solidClient.getSolidDataset(url, {
    fetch: session.fetch  // ← Critical!
});
```

- Must use `session.fetch`, not regular `fetch()`
- `session.fetch` automatically adds authentication headers and DPoP proofs
- Regular `fetch()` will get 401 Unauthorized for private resources

**Gotcha**: Forgetting `fetch: session.fetch` is the #1 cause of authentication errors.

---

## RDF Data Handling

### Key Insights

#### 1. Turtle is Human-Friendly, Machine-Parseable
```turtle
<#location> a schema:Place, geo:Point ;
    geo:lat "37.7749"^^xsd:decimal ;
    geo:long "-122.4194"^^xsd:decimal .
```

**Pros**:
- Easy to read and debug
- Compact representation
- Good for simple, flat data structures

**Cons**:
- Harder to generate programmatically (string templates)
- Whitespace-sensitive
- Easy to make syntax errors

**Best Practice**: Use Turtle for simple data (locations, settings). Use JSON-LD for complex nested structures.

#### 2. JSON-LD for Complex Structures
```json
{
  "@context": { "@vocab": "http://schema.org/" },
  "@type": "Report",
  "associatedMedia": [...]
}
```

**Pros**:
- Native JavaScript object handling
- Easy to generate and parse
- Better for nested data (error logs, transactions)

**Cons**:
- More verbose
- Context must be carefully managed
- Type coercion can be tricky

#### 3. Thing URLs Include Fragment Identifiers
```javascript
const thing = solidClient.getThing(dataset, `${url}#location`);
```

- The `#location` is the fragment identifier
- Refers to a specific resource within the file
- Multiple Things can exist in one file
- Fragment is part of the Thing's identity

**Gotcha**: Forgetting the `#` will result in "Thing not found" errors.

#### 4. Typed Getters vs Generic Getters
```javascript
// Typed - returns number or null
const lat = solidClient.getDecimal(thing, 'http://www.w3.org/2003/01/geo/wgs84_pos#lat');

// Generic - returns all values
const allLats = solidClient.getAll(thing, 'http://www.w3.org/2003/01/geo/wgs84_pos#lat');
```

- Typed getters: `getDecimal()`, `getInteger()`, `getDatetime()`, `getStringNoLocale()`
- Return `null` if property doesn't exist
- Automatic type conversion
- Safer than generic getters

**Best Practice**: Always use typed getters when you know the datatype.

#### 5. Predicate URIs Must Be Full URIs
```javascript
// ✅ Correct
solidClient.getDecimal(thing, 'http://www.w3.org/2003/01/geo/wgs84_pos#lat')

// ❌ Wrong
solidClient.getDecimal(thing, 'geo:lat')
```

- No prefix expansion in JavaScript API
- Must use full URI form
- Define constants for commonly used predicates

**Best Practice**:
```javascript
const PREDICATES = {
    GEO_LAT: 'http://www.w3.org/2003/01/geo/wgs84_pos#lat',
    GEO_LONG: 'http://www.w3.org/2003/01/geo/wgs84_pos#long',
    SCHEMA_NAME: 'http://schema.org/name'
};
```

#### 6. Datatype Literals Need Explicit Types
```turtle
geo:lat "37.7749"^^xsd:decimal ;
```

- The `^^xsd:decimal` is the datatype
- Required for non-string values
- Common types:
  - `xsd:decimal` - floating point
  - `xsd:integer` - whole numbers
  - `xsd:dateTime` - ISO 8601 timestamps
  - `xsd:boolean` - true/false

**Gotcha**: Omitting datatype causes values to be treated as strings.

---

## Common Pitfalls

### 1. CORS Issues with `file://` Protocol
**Problem**: Opening HTML directly from filesystem triggers CORS errors

**Solution**: Use a local web server (Python http.server, Node http-server, etc.)

### 2. State Lost During OAuth Redirect
**Problem**: In-memory state is cleared when redirecting to provider

**Solution**: Save state to localStorage before calling `session.login()`

### 3. Callback URL Mismatch
**Problem**: "Invalid redirect_uri" error

**Solution**: Ensure `redirectUrl` exactly matches current URL:
```javascript
redirectUrl: window.location.href  // Includes path, query, hash
```

### 4. Reading Before Container Exists
**Problem**: 404 error when trying to read from a container that hasn't been created

**Solution**: Create container structure on first write, or during onboarding

### 5. Case Sensitivity in URLs
**Problem**: Accessing `/Private/` instead of `/private/`

**Solution**: Solid Pods are case-sensitive. Always use lowercase for standard containers.

### 6. Forgetting to Await Async Operations
```javascript
// ❌ Wrong
const dataset = solidClient.getSolidDataset(url, { fetch: session.fetch });

// ✅ Correct
const dataset = await solidClient.getSolidDataset(url, { fetch: session.fetch });
```

**All Solid operations are asynchronous**. Always use `await` or `.then()`.

### 7. Not Checking if Logged In
```javascript
// ✅ Always check first
if (!session.info.isLoggedIn) {
    console.error('User not authenticated');
    return;
}
```

Attempting operations without authentication causes confusing errors.

---

## Security Considerations

### 1. Private Data Must Use `/private/` Container
- Location data is sensitive
- Always write to `/private/location-tracker/`
- Never use `/public/` unless explicitly sharing

### 2. DPoP Prevents Token Theft
- Even if access token is intercepted, attacker can't use it
- Requires the private key which never leaves browser
- Provides stronger security than Bearer tokens

### 3. No Server-Side Secrets in Browser Code
- WebID is public (safe to expose)
- Access tokens handled by library (in memory)
- Never embed API keys or secrets in frontend

### 4. Logout is Client-Side Only
- `session.logout()` clears browser state
- Does NOT revoke tokens at provider
- For complete logout, redirect to provider's logout endpoint

### 5. HTTPS Required in Production
- OAuth providers reject non-HTTPS redirect URLs
- DPoP requires secure context (HTTPS or localhost)
- Plan for SSL certificates in deployment

### 6. Trust the Provider
- Provider controls authentication
- Provider can read all Pod data
- Choose providers carefully (privacy policy, terms of service)

---

## Provider Differences

### SolidCommunity.net
- **Type**: Free community provider
- **Pods**: Unlimited (within reason)
- **Storage**: Limited (check provider terms)
- **Registration**: Email-based, no verification required
- **Stability**: Good for development, not production-grade
- **Best For**: Testing, demos, learning

### Inrupt PodSpaces
- **Type**: Commercial provider
- **Pods**: Multiple per account (paid tiers)
- **Storage**: 2GB free, paid tiers available
- **Registration**: Email verification required
- **Stability**: Production-ready
- **Best For**: Production applications

### Community Solid Server (localhost)
- **Type**: Self-hosted
- **Pods**: Unlimited
- **Storage**: Limited by disk space
- **Registration**: Email/password (no real email sent)
- **Stability**: Excellent for development
- **Best For**: Offline development, CI/CD, testing

### Provider Selection Strategy
- **Development**: Use localhost CSS or SolidCommunity.net
- **Staging**: Use Inrupt PodSpaces free tier
- **Production**: Use Inrupt PodSpaces paid or self-host CSS

---

## Performance Insights

### 1. Fetches are Not Cached by Default
- Each `getSolidDataset()` makes a network request
- No automatic caching in the library
- Implement application-level caching if needed

**Best Practice**: Cache datasets in memory for the session duration.

### 2. Write Operations are Expensive
- Each write requires:
  - Authentication
  - DPoP proof generation
  - Network round-trip
  - Server-side RDF parsing

**Best Practice**: Batch writes when possible. Update datasets locally and write once.

### 3. Container Traversal is Slow
- No built-in "get all files in container recursively"
- Must fetch container, iterate children, fetch each
- Can be hundreds of requests for deep hierarchies

**Best Practice**: Design flat container structures. Use date-based organization.

### 4. WebID Dereference Can Be Cached
```javascript
const profile = await solidClient.getSolidDataset(webId, { fetch: session.fetch });
```

- WebID profile rarely changes
- Safe to cache for session duration
- Reduces redundant fetches

### 5. Solid is Not a Database
- Not optimized for querying
- No indexes or query language (LDP provides minimal querying)
- Better for document storage than relational queries

**Implication**: For complex queries, consider:
- Maintaining indexes in DynamoDB
- Using SPARQL endpoints (advanced)
- Caching and querying locally

---

## Recommendations for Production Integration

### 1. Hybrid Architecture
- Frontend: Use `@inrupt/solid-client-authn-browser` for authentication
- Backend: Validate WebID and maintain sessions in Go
- Data: Store in Solid Pods, cache indexes in DynamoDB

### 2. Progressive Enhancement
- Start with password auth, add Solid as optional
- Allow users to migrate gradually
- Maintain backward compatibility

### 3. Error Handling
- Always wrap Solid operations in try/catch
- Provide meaningful user messages
- Log errors with context (URL, operation, WebID)

### 4. Offline Support
- Cache Pod data in IndexedDB
- Queue writes when offline
- Sync when connection restored

### 5. Testing Strategy
- Unit tests: Mock `session.fetch`
- Integration tests: Use local CSS
- E2E tests: Use test Pods on SolidCommunity.net

---

## Key Takeaways

1. **OAuth redirect is unavoidable** - Design UX around it
2. **DPoP is automatic** - Don't try to implement it yourself
3. **Always use `session.fetch`** - Regular fetch won't work
4. **Turtle for simple, JSON-LD for complex** - Choose format wisely
5. **Pod operations are slow** - Cache aggressively
6. **Private by default** - Use `/private/` for sensitive data
7. **Provider differences matter** - Test across multiple providers
8. **Solid is not a database** - Don't treat it like one

---

## Next Steps

Based on these learnings:

1. **Issue #49**: Implement Go library for WebID validation
2. **Issue #50**: Create hybrid auth endpoints (password + Solid)
3. **Issue #51**: Build storage abstraction layer
4. **Issue #52**: Implement caching strategy for Pod operations

---

## Resources

- **Solid Protocol**: https://solidproject.org/TR/protocol
- **Inrupt Docs**: https://docs.inrupt.com/
- **DPoP RFC 9449**: https://www.rfc-editor.org/rfc/rfc9449.html
- **RDF Primer**: https://www.w3.org/TR/rdf-primer/
- **Our Auth Research**: `../SOLID_AUTHENTICATION.md`
- **Our Data Models**: `../SOLID_DATA_MODELS.md`

---

*This document will be updated as we integrate Solid into the production application and discover additional insights.*
