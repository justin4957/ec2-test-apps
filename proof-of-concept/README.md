# Solid Authentication Proof-of-Concept

This is a standalone proof-of-concept demonstrating Solid Pod authentication and basic data operations for the Location Tracker project.

## Overview

This PoC validates the authentication flow and data operations that will be integrated into the Location Tracker application. It demonstrates:

- ✅ OIDC authentication with any Solid provider
- ✅ DPoP token handling (automatic via Inrupt libraries)
- ✅ Reading WebID profiles
- ✅ Writing location data to Pods
- ✅ Reading location data from Pods
- ✅ Proper error handling

## Quick Start

### Option 1: Open Directly in Browser

Simply open `index.html` in any modern web browser:

```bash
# macOS
open proof-of-concept/index.html

# Linux
xdg-open proof-of-concept/index.html

# Windows
start proof-of-concept/index.html
```

### Option 2: Serve via Local Web Server

For a better development experience:

```bash
# Using Python
python3 -m http.server 8080

# Using Node.js
npx http-server -p 8080

# Using PHP
php -S localhost:8080
```

Then navigate to: `http://localhost:8080/proof-of-concept/`

## Testing Instructions

### Prerequisites

You need a Solid Pod to test with. Choose one of:

1. **SolidCommunity.net** (Recommended for testing)
   - Navigate to https://solidcommunity.net/register
   - Create a free Pod account
   - Use `https://solidcommunity.net` as your provider

2. **Inrupt PodSpaces** (Commercial option)
   - Navigate to https://start.inrupt.com
   - Sign up for a free Pod
   - Use `https://inrupt.net` as your provider

3. **Local Community Solid Server** (For development)
   - See `SOLID_DEV_SETUP.md` for installation instructions
   - Use `http://localhost:3000` as your provider

### Step-by-Step Test Flow

#### 1. Choose Provider
- Open the application in your browser
- Select a provider from the quick links or enter a custom URL
- The default is `https://solidcommunity.net`

#### 2. Authenticate
- Click "Login with Solid"
- You will be redirected to your Solid provider
- Log in with your credentials
- Authorize the application
- You'll be redirected back with authentication complete

#### 3. Read WebID Profile
- After login, the profile is automatically read
- View your name and WebID in the profile card
- Check the profile data section for full details
- The green indicator shows the step is complete

#### 4. Write Location Data
- Click "Write Test Location"
- This writes a test location (San Francisco, CA) to your Pod
- Location is stored at: `/private/location-tracker/test-location-[timestamp].ttl`
- You can verify the file in your Pod browser

#### 5. Read Location Data
- Click "Read Location Data"
- The application reads back the location you just wrote
- View parsed coordinates and metadata
- Expand "View Raw Turtle Data" to see the RDF format

#### 6. Logout
- Click "Logout" to end the session
- All indicators will reset
- You can login again with the same or different provider

### Verification Checklist

After completing the test flow, verify:

- [ ] Successfully logged in with a Solid provider
- [ ] Profile data displayed correctly
- [ ] Location file created in Pod at `/private/location-tracker/`
- [ ] Location data readable and parsed correctly
- [ ] Activity log shows all operations
- [ ] No errors in browser console
- [ ] Logout works and clears session

## Pod Structure

The application creates the following structure in your Pod:

```
/private/
  └── location-tracker/
      └── test-location-[timestamp].ttl
```

Example file name: `test-location-1699824000000.ttl`

## Viewing Your Pod Data

### Via Pod Browser

Each provider has a web interface to browse your Pod:

- **SolidCommunity.net**: https://solidcommunity.net/[username]/
- **Inrupt PodSpaces**: https://pod.inrupt.com/[username]/
- **Local CSS**: http://localhost:3000/[username]/

Navigate to `/private/location-tracker/` to see the files created by this PoC.

### Via Direct URL

The application displays the direct URL to each file created. You can:
- Click the link to view in your browser (if logged in)
- Use a Solid data browser like Penny (https://penny.vincenttunru.com/)
- Use the Solid Data Browser extension

## RDF Data Format

Location data is stored in Turtle format following the schema defined in issue #47:

```turtle
@prefix schema: <http://schema.org/> .
@prefix geo: <http://www.w3.org/2003/01/geo/wgs84_pos#> .
@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .
@prefix dcterms: <http://purl.org/dc/terms/> .

<#location> a schema:Place, geo:Point ;
    geo:lat "37.7749"^^xsd:decimal ;
    geo:long "-122.4194"^^xsd:decimal ;
    schema:accuracy "10.5"^^xsd:decimal ;
    dcterms:created "2025-11-12T18:30:00Z"^^xsd:dateTime ;
    schema:identifier "poc-test-device" .
```

## Troubleshooting

### Login Redirect Loop

**Problem**: Clicking login causes a redirect loop

**Solution**:
- Ensure you're using a valid provider URL
- Try clearing browser cache and cookies
- Check browser console for CORS errors

### "No storage location found"

**Problem**: Write operation fails with this error

**Solution**:
- Your WebID profile needs a `pim:storage` triple
- Most providers set this automatically
- Try a different provider if issue persists

### CORS Errors

**Problem**: Console shows CORS policy errors

**Solution**:
- Ensure you're accessing via `http://` or `https://` (not `file://`)
- Use a local web server (see "Option 2" above)
- Some providers require HTTPS for production apps

### Authentication Fails

**Problem**: "Invalid client" or "Unauthorized" errors

**Solution**:
- Check that the provider URL is correct
- Ensure provider is accessible (not behind firewall)
- Try a different provider to isolate the issue

### Can't Read Data Back

**Problem**: Write succeeds but read fails

**Solution**:
- Wait a few seconds after writing before reading
- Check Pod browser to confirm file exists
- Verify the file URL in the activity log
- Check browser console for detailed error messages

## Security Notes

### Data Stored Locally
- Session information is stored in browser localStorage
- Credentials are never stored (handled by provider)
- DPoP keys are generated in browser and never leave

### Permissions
- All data written to `/private/` is only accessible to you
- No data is shared with the Location Tracker server
- You can revoke application access at any time

### HTTPS Requirements
- Production applications must use HTTPS
- This PoC works with HTTP for local development
- Solid providers require HTTPS for OAuth callbacks in production

## Browser Compatibility

Tested and working on:
- ✅ Chrome/Chromium 90+
- ✅ Firefox 88+
- ✅ Safari 14+
- ✅ Edge 90+

## Libraries Used

- **@inrupt/solid-client-authn-browser** v2.0.0 - Authentication
- **@inrupt/solid-client** v2.0.0 - Data operations

Both loaded via CDN for simplicity. Production applications should use npm packages.

## Next Steps

After validating this PoC:

1. **Issue #49**: Implement Go Solid client library
2. **Issue #50**: Add Solid authentication endpoints to Location Tracker backend
3. **Issue #51**: Create data storage abstraction layer
4. **Issue #52**: Implement Pod read/write operations in production code

## Resources

- **Inrupt Documentation**: https://docs.inrupt.com/
- **Solid Protocol Spec**: https://solidproject.org/TR/protocol
- **RDF Data Models** (Issue #47): See `../SOLID_DATA_MODELS.md`
- **Authentication Research** (Issue #46): See `../SOLID_AUTHENTICATION.md`

## Support

For issues or questions:
1. Check the activity log in the application
2. Review browser console for detailed errors
3. Consult `LEARNINGS.md` for common gotchas
4. See `SOLID_DEV_SETUP.md` for development environment setup

## License

Part of the Location Tracker project. See main repository for license information.
