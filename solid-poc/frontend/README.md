# Solid Authentication Frontend

This is the frontend component of the solid-poc application, demonstrating Solid Pod authentication and basic data operations for the Location Tracker project.

**Location**: `solid-poc/frontend/` (served by `solid-poc/main.go`)

## Overview

This PoC validates the authentication flow and data operations that will be integrated into the Location Tracker application. It demonstrates:

- ✅ OIDC authentication with any Solid provider
- ✅ DPoP token handling (automatic via Inrupt libraries)
- ✅ Reading WebID profiles
- ✅ Writing location data to Pods
- ✅ Reading location data from Pods
- ✅ Proper error handling
- ✅ **Offline-first capabilities** (Issue #56)
  - Service Worker for asset caching
  - IndexedDB for local data storage
  - Operation queue for offline writes
  - Automatic sync when back online
  - Online/offline status indicators
- ✅ **Data Sharing & Permissions** (Issue #57)
  - WAC/ACP universal access control
  - Grant read/write permissions to individuals
  - Public/private access management
  - View current permissions
  - Revoke access from users
  - Generate shareable links
- ✅ **Social Features** (NEW - Issue #58)
  - FOAF (Friend of a Friend) integration
  - Add/remove friends by WebID
  - Discover friend profiles
  - Social feed of friends' shared locations
  - Client-side friend list management
  - Privacy-aware location sharing

## Quick Start

**IMPORTANT**: You need to build the JavaScript bundle first before running the application.

### Step 1: Build the Bundle

The Solid client libraries are bundled locally for better reliability:

```bash
cd solid-poc/frontend
npm install
npm run build
```

This creates `dist/solid-client-bundle.js` from the Inrupt libraries.

### Step 2: Run the solid-poc Server

The frontend is served by the solid-poc backend:

```bash
cd solid-poc
go run main.go
```

Then navigate to: `http://localhost:9090`

The server serves the frontend files and provides RDF serialization API endpoints.

**Alternative** - Standalone Frontend Server:

You can also serve just the frontend (without the backend APIs) using:

```bash
cd solid-poc/frontend
python3 -m http.server 8080
# Then navigate to: http://localhost:8080/index.html
```

**Note**: Opening `index.html` directly in a browser (`file://` protocol) will cause CORS errors and OAuth redirect issues.

## Testing Instructions

### Prerequisites

You need a Solid Pod to test with. Choose one of:

1. **SolidCommunity.net** (Recommended for testing)
   - Navigate to https://solidcommunity.net/register
   - Create a free Pod account
   - Use `https://solidcommunity.net` as your provider

2. **Inrupt PodSpaces** (Developer Preview)
   - Navigate to https://start.inrupt.com
   - Sign up for a Pod (register at https://login.inrupt.com)
   - Use `https://login.inrupt.com` as your OIDC issuer
   - Note: PodSpaces is in Developer Preview - not for production use

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

- **@inrupt/solid-client-authn-browser** v3.1.1 - Authentication
- **@inrupt/solid-client** v3.0.0 - Data operations

Both libraries are bundled locally using webpack. The bundle is generated from npm packages and served as `dist/solid-client-bundle.js`. This approach:
- ✅ Ensures consistent library versions
- ✅ Works offline after initial build
- ✅ Avoids CDN reliability issues
- ✅ Provides better performance (single bundle file)

### Rebuilding the Bundle

If you update the libraries or make changes to `src/index.mjs`:

```bash
npm run build
```

This will regenerate `dist/solid-client-bundle.js`.

## Offline-First Capabilities

### Overview

The application now works offline! All features continue to function without an internet connection:

- **Asset Caching**: The Service Worker caches static assets (HTML, JS, CSS) for offline use
- **Local Data Storage**: IndexedDB stores location data and Pod operations locally
- **Operation Queue**: Write operations are queued when offline and synced automatically when back online
- **Status Indicators**: Real-time online/offline status displayed in the UI
- **Automatic Sync**: Pending operations sync automatically when connection is restored

### How It Works

#### Service Worker (`sw.js`)

The Service Worker provides:
- **Cache-first strategy** for static assets (HTML, JS, fonts, images)
- **Network-first strategy** for API calls
- **Background sync** support for queued operations
- **Automatic cache management** and cleanup

#### IndexedDB Storage (`offline.js`)

Four object stores manage offline data:

1. **locations**: Cached location data with sync status
2. **pendingOps**: Queued operations waiting to sync
3. **profiles**: Cached WebID profile data
4. **syncMeta**: Sync metadata (last sync time, etc.)

#### Offline Write Flow

1. User clicks "Write Test Location" while offline
2. Location data is queued in IndexedDB `pendingOps` store
3. UI shows "Queued for sync" message
4. When online, auto-sync triggers automatically
5. Queued operation executes and writes to Pod
6. Operation removed from queue on success

#### Conflict Resolution

The current implementation uses **last-write-wins** strategy:
- Each location has a unique timestamp-based filename
- No conflicts occur as each write creates a new file
- Future enhancement: Detect concurrent edits to same resource

#### Retry Logic

Failed operations are automatically retried:
- **Max retries**: 5 attempts per operation
- **Retry interval**: Triggered on each online event or manual sync
- **Failure handling**: After 5 failures, operation marked as "failed" (not retried)

### Using Offline Features

#### 1. Monitor Online/Offline Status

Two status indicators show in the header:
- **Solid Authentication**: Connection to Pod provider
- **Online Status**: Internet connectivity

#### 2. Write Data Offline

1. Disconnect from internet (airplane mode or disable WiFi)
2. Click "Write Test Location"
3. Location is queued for sync (shown in UI)
4. Reconnect to internet
5. Auto-sync runs and saves location to Pod

#### 3. Manual Sync

The "Offline Sync Status" section provides:
- **Sync Stats**: View pending/failed operations
- **Sync Now**: Manually trigger sync
- **View Queue**: See all pending operations
- **Clear Cache**: Reset offline storage

#### 4. Test Offline Mode

**Test Scenario:**
```
1. Login to Solid Pod (online)
2. Disconnect internet
3. Write test location → Queued
4. Write another location → Queued
5. Reconnect internet
6. Auto-sync runs → Both locations saved
7. Check Pod → See both files
```

### Sync Statistics

The sync status section shows:
- 📋 **Pending Operations**: Operations waiting to sync
- ❌ **Failed Operations**: Operations that failed after retries
- 💾 **Cached Locations**: Total locations in local cache
- ✅ **Synced Locations**: Locations successfully synced to Pod
- ⏱️ **Last Sync**: Timestamp of last successful sync

### Browser Storage

**IndexedDB Database**: `SolidPoCDB`

You can inspect offline data in browser DevTools:
- **Chrome/Edge**: DevTools → Application → IndexedDB → SolidPoCDB
- **Firefox**: DevTools → Storage → IndexedDB → SolidPoCDB
- **Safari**: Develop → Show Web Inspector → Storage → IndexedDB

**Service Worker Cache**: `solid-poc-v1-static`, `solid-poc-v1-dynamic`

View cached assets:
- **Chrome/Edge**: DevTools → Application → Cache Storage
- **Firefox**: DevTools → Storage → Cache Storage

### Known Limitations

1. **No conflict detection**: Concurrent writes to same file not handled
2. **Cache size limits**: Browser may evict cache if storage full
3. **No differential sync**: Full operations replayed (not deltas)
4. **Authentication required**: Must login once online before offline use works
5. **No background sync on iOS**: Safari doesn't support Background Sync API

### Clearing Offline Data

**Manual Clear:**
1. Click "Clear Cache" in Offline Sync Status section
2. Confirms before clearing all offline data

**Browser Clear:**
- Chrome/Edge: Settings → Privacy → Clear browsing data → Cached images and files
- Firefox: Settings → Privacy → Clear Data → Cached Web Content
- Safari: Develop → Empty Caches

**Programmatic Clear:**
```javascript
// Clear all offline data
await offlineStorage.clearAll();

// Clear Service Worker cache
const caches = await caches.keys();
caches.forEach(cache => caches.delete(cache));
```

## Data Sharing & Permissions

### Overview

The application supports full access control for your location data using Solid's Universal Access API. This works with both WAC (Web Access Control) and ACP (Access Control Policies), depending on your Pod provider.

### Features

#### 1. View Current Permissions

Navigate to **Settings & Preferences → Privacy** to:
- See public access status (Private, Public Read-Only, or Public Read-Write)
- View all individuals who have access to your data
- See what level of access each person has (Read, Append, Write)

#### 2. Grant Public Access

Make your location data publicly accessible:

**Make Public (Read-Only):**
- Anyone can view your location data
- No authentication required
- Cannot modify your data

**Make Public (Read-Write):**
- Anyone can view AND modify your location data
- ⚠️ Use with caution - allows anonymous writes

**Make Private:**
- Revoke all public access
- Only you and specifically granted users can access

#### 3. Grant Access to Individuals

Share with specific people by their WebID:

1. Enter their WebID (e.g., `https://alice.solidcommunity.net/profile/card#me`)
2. Choose access level:
   - **Read Access**: They can view your location data
   - **Write Access**: They can view AND modify your location data
3. Click "Grant Read Access" or "Grant Write Access"

#### 4. Revoke Access

Remove access from a specific person:

1. Enter their WebID in the "Revoke Access" section
2. Click "Revoke All Access"
3. All their permissions (read, write, append) are removed

#### 5. Generate Share Links

Create shareable links to your location data container:

1. Click "Generate Share Link"
2. Copy the generated URL
3. Share it with others (they'll need permissions to access)

**Note**: The link alone doesn't grant access - you must explicitly grant permissions using the steps above.

### Access Levels Explained

Solid supports three main access modes:

- **Read**: View data in your Pod
- **Append**: Add new data to your Pod
- **Write**: Modify or delete existing data

The UI simplifies this to:
- **Read-Only**: Read access only
- **Read-Write**: Read + Append + Write access

### Permission Inheritance

- Permissions apply to the `/private/location-tracker/` container
- All files in the container inherit these permissions
- You can override permissions on individual files (future feature)

### Testing Permissions

**Test Scenario 1: Share with a Friend**
```
1. Login to your Solid Pod
2. Go to Settings → Privacy
3. Enter your friend's WebID
4. Grant Read Access
5. Send them the share link
6. They can now view your location data
```

**Test Scenario 2: Make Data Public**
```
1. Login to your Solid Pod
2. Go to Settings → Privacy
3. Click "Make Public (Read-Only)"
4. Generate Share Link
5. Anyone with the link can view (no login required)
```

**Test Scenario 3: Revoke Access**
```
1. View "Current Access Permissions"
2. Copy the WebID of person to remove
3. Paste in "Revoke Access" section
4. Click "Revoke All Access"
5. They can no longer access your data
```

### Security Best Practices

1. **Default to Private**: Only share when necessary
2. **Audit Regularly**: Check "Current Access Permissions" periodically
3. **Use Read-Only**: Grant read access unless write is required
4. **Verify WebIDs**: Double-check WebIDs before granting access
5. **Avoid Public Write**: Never make data public with write access unless you understand the risks

### Compatibility

**Supported Pod Providers:**
- ✅ SolidCommunity.net (WAC)
- ✅ Inrupt PodSpaces (ACP)
- ✅ Community Solid Server (WAC or ACP)

**Universal Access API Benefits:**
- Works with both WAC and ACP automatically
- Provider-agnostic code
- Future-proof as Solid specification evolves

### Known Limitations

1. **Direct access only**: Shows only explicitly-set permissions, not inherited access
2. **Container-level**: Permissions apply to whole container (not individual files yet)
3. **No group support**: Can only grant to individual WebIDs (not groups)
4. **No inheritance display**: Doesn't show permissions inherited from parent containers

### Troubleshooting Permissions

**Problem**: "Failed to update access"

**Possible Causes:**
- You don't own the resource
- Pod provider doesn't support access control
- Network connection issue

**Solution:**
- Verify you're logged in with correct Pod
- Check browser console for detailed errors
- Try refreshing access info

**Problem**: Permissions not showing

**Solution:**
- Click "Refresh Access Info"
- Verify the container exists (`/private/location-tracker/`)
- Check that you've written at least one location

**Problem**: Granted access but friend can't view

**Solution:**
- Verify WebID is correct (copy from their profile)
- Ensure they're logged into their Solid Pod
- Check they're accessing the correct URL
- Confirm permissions with "Refresh Access Info"

## Social Features (FOAF)

### Overview

The application implements Solid's social capabilities using FOAF (Friend of a Friend) vocabulary. This enables decentralized friend management and social location sharing without a central server.

### Features

#### 1. Friend Management

**Add Friends:**
- Navigate to **Settings & Preferences → Social**
- Enter a friend's WebID
- Click "Add Friend"
- Friend is saved to `/private/friends.ttl` in your Pod

**Remove Friends:**
- View your friends list
- Click "Remove" next to any friend
- Confirm the removal

**Friends List Storage:**
```turtle
@prefix foaf: <http://xmlns.com/foaf/0.1/> .
@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .

<#me> a foaf:Person ;
    foaf:knows <https://friend1.solidcommunity.net/profile/card#me> ;
    foaf:knows <https://friend2.solidcommunity.net/profile/card#me> .
```

#### 2. Profile Discovery

Discover information about any Solid user:

1. Go to **Settings → Social → Discover Friend**
2. Enter their WebID
3. View their profile information:
   - Name
   - WebID
   - Storage location

This helps verify the correct WebID before adding friends.

#### 3. Social Feed

View location data shared by your friends:

**How It Works:**
1. Application reads your friends list from `/private/friends.ttl`
2. For each friend, fetches their profile to get storage URL
3. Attempts to read `/private/location-tracker/` from their Pod
4. Displays accessible locations in chronological order

**Privacy:**
- Only shows locations you have permission to view
- Friends must grant you read access to their location container
- Uses the permissions system from Issue #57

**Feed Display:**
- Shows friend's name and location name
- Displays coordinates
- Shows relative time (e.g., "2h ago", "3d ago")
- Sorted by newest first

#### 4. Refresh Feed

Click "Refresh Feed" to manually update the social feed with latest data from friends' Pods.

### Testing Social Features

**Two-User Test Scenario:**

**User A (You):**
1. Login to your Solid Pod
2. Write a test location
3. Go to Settings → Privacy
4. Grant read access to User B's WebID
5. Go to Settings → Social
6. Add User B as a friend

**User B (Friend):**
1. Login to their Solid Pod
2. Write a test location
3. Grant read access to your WebID
4. Add you as a friend
5. Click "Refresh Feed"
6. See your shared location in their feed

**Expected Result:**
- Both users see each other's shared locations
- Locations appear in social feed
- Can add/remove friends
- Profile discovery works

### Architecture

**Client-Side Only:**
- All friend management happens in browser
- Friends list stored in user's Pod (`/private/friends.ttl`)
- No server-side social graph
- Uses FOAF vocabulary for interoperability

**Data Flow:**
```
1. User adds friend → Save to /private/friends.ttl
2. Refresh feed clicked → Read friends list
3. For each friend → Fetch profile
4. For each friend → Try to read locations
5. Display accessible locations
```

**Permission Requirements:**
- Friend must grant you read access to their `/private/location-tracker/` container
- You must grant friends read access to see your locations
- Uses Universal Access API from Issue #57

### FOAF Vocabulary

The application uses these FOAF predicates:

- **`foaf:Person`** - Type of the user entity
- **`foaf:knows`** - Relationship indicating friendship/connection
- **`foaf:name`** - Person's name (read from profile)

Compatible with any Solid application using FOAF vocabulary.

### Privacy Considerations

**What Friends Can See:**
- Only locations you've granted access to
- Only data in containers with read permissions
- Your WebID and public profile info

**What Friends Cannot See:**
- Private data without explicit permissions
- Location containers you haven't shared
- Other friends in your list (unless you share friends.ttl)

**Best Practices:**
1. Only add people you know
2. Review granted permissions regularly
3. Use Privacy tab to manage access
4. Friends list is private by default

### Known Limitations

1. **No mutual friend discovery**: Can't see your friends' friends automatically
2. **Manual refresh**: Feed doesn't auto-update (click "Refresh Feed")
3. **No notifications**: No alerts when friends share new locations
4. **No groups**: Can only add individual WebIDs, not groups
5. **Permission coordination**: Both users must grant access to see each other's data

### Troubleshooting Social Features

**Problem**: Added friend but see no locations in feed

**Solution:**
- Friend may not have any location data yet
- Friend may not have granted you read access
- Verify friend's WebID is correct
- Check if friend's Pod is accessible
- Ask friend to grant you read access via Privacy tab

**Problem**: Can't add friend - "Failed to add"

**Solution:**
- Verify WebID starts with http:// or https://
- Check WebID is accessible (try Profile Discovery first)
- Ensure you're logged in
- Check browser console for detailed errors

**Problem**: Friend's name shows as WebID

**Solution:**
- Friend's profile may not have a `foaf:name` predicate
- This is normal - WebID will be used as display name
- Name will appear if friend updates their profile

## Next Steps

After validating this PoC:

1. **Issue #49**: Implement Go Solid client library
2. **Issue #50**: Add Solid authentication endpoints to Location Tracker backend
3. **Issue #51**: Create data storage abstraction layer
4. **Issue #52**: Implement Pod read/write operations in production code
5. **Phase 5**: Integrate social features into location-tracker main app
6. **Phase 5**: Integrate sharing and permissions patterns into location-tracker main app
7. **Phase 5**: Integrate offline-first patterns into location-tracker main app

## Resources

- **Inrupt Documentation**: https://docs.inrupt.com/
- **Solid Protocol Spec**: https://solidproject.org/TR/protocol
- **RDF Data Models** (Issue #47): See `../../SOLID_DATA_MODELS.md`
- **Authentication Research** (Issue #46): See `../../SOLID_AUTHENTICATION.md`
- **Backend README**: See `../README.md` for solid-poc server documentation

## Support

For issues or questions:
1. Check the activity log in the application
2. Review browser console for detailed errors
3. Consult `LEARNINGS.md` for common gotchas
4. See `SOLID_DEV_SETUP.md` for development environment setup

## License

Part of the Location Tracker project. See main repository for license information.
