# Solid Development Environment Setup

**Issue:** #45
**Phase:** Phase 1.1 - Foundation & Research
**Status:** ✅ Complete

---

## Overview

This document provides step-by-step instructions for setting up a complete Solid Project development environment for the Location Tracker application.

---

## Prerequisites

- **Node.js**: v18.x or higher
- **npm**: v9.x or higher
- **Go**: 1.21 or higher (for backend development)
- **Git**: Latest version
- **Modern web browser**: Chrome, Firefox, or Safari

---

## 1. Test Pod Accounts

### 1.1 Inrupt PodSpaces Account

**Provider:** Inrupt PodSpaces (Commercial provider)

**Steps to create:**
1. Visit https://signup.pod.inrupt.com/
2. Click "Create an account"
3. Fill in your details:
   - Email address
   - Password (strong password required)
   - Pod name (will become your Pod URL)
4. Verify your email address
5. Complete the setup wizard

**Your Pod URL will be:** `https://[your-pod-name].inrupt.net/`

**Features:**
- ✅ Reliable uptime
- ✅ Fast performance
- ✅ Commercial support
- ✅ Free tier available
- ✅ GDPR compliant

**Testing:**
```bash
# Test Pod accessibility
curl https://[your-pod-name].inrupt.net/profile/card
```

### 1.2 SolidCommunity.net Account

**Provider:** SolidCommunity.net (Community-run)

**Steps to create:**
1. Visit https://solidcommunity.net/
2. Click "Register"
3. Fill in registration form:
   - Username
   - Name
   - Email
   - Password
4. Confirm email verification
5. Your Pod is ready!

**Your Pod URL will be:** `https://[username].solidcommunity.net/`

**Features:**
- ✅ Community-driven
- ✅ Open source
- ✅ Free forever
- ✅ Good for testing
- ⚠️ Lower SLA than commercial

**Testing:**
```bash
# Test Pod accessibility
curl https://[username].solidcommunity.net/profile/card
```

### 1.3 Pod Credentials Management

**⚠️ IMPORTANT SECURITY NOTICE:**

**DO NOT commit Pod credentials to the repository!**

Create a local file `~/.solid-pods` to track your test accounts:

```bash
# ~/.solid-pods
# Test Pod Accounts for Location Tracker Development
# DO NOT COMMIT THIS FILE

[Inrupt PodSpaces]
Email: your-email@example.com
Pod URL: https://your-pod-name.inrupt.net/
WebID: https://your-pod-name.inrupt.net/profile/card#me
Created: 2025-11-12
Notes: Primary test account

[SolidCommunity.net]
Username: your-username
Email: your-email@example.com
Pod URL: https://your-username.solidcommunity.net/
WebID: https://your-username.solidcommunity.net/profile/card#me
Created: 2025-11-12
Notes: Secondary test account
```

**Secure this file:**
```bash
chmod 600 ~/.solid-pods
```

---

## 2. Local Solid Server (Community Solid Server)

### 2.1 Installation

Install Community Solid Server (CSS) globally:

```bash
npm install -g @solid/community-server
```

Verify installation:
```bash
community-solid-server --version
```

### 2.2 Configuration

Create a local configuration directory:

```bash
mkdir -p ~/solid-dev/css-config
cd ~/solid-dev/css-config
```

Create a basic configuration file `config.json`:

```json
{
  "@context": "https://linkedsoftwaredependencies.org/bundles/npm/@solid/community-server/^7.0.0/components/context.jsonld",
  "import": [
    "css:config/app/main/default.json",
    "css:config/app/init/initialize-root.json",
    "css:config/http/handler/default.json",
    "css:config/http/middleware/websockets.json",
    "css:config/http/server-factory/https.json",
    "css:config/http/static/default.json",
    "css:config/identity/access/public.json",
    "css:config/identity/email/default.json",
    "css:config/identity/handler/default.json",
    "css:config/identity/ownership/token.json",
    "css:config/identity/pod/static.json",
    "css:config/ldp/authentication/dpop-bearer.json",
    "css:config/ldp/authorization/webacl.json",
    "css:config/ldp/handler/default.json",
    "css:config/ldp/metadata-parser/default.json",
    "css:config/ldp/metadata-writer/default.json",
    "css:config/ldp/modes/default.json",
    "css:config/storage/backend/memory.json",
    "css:config/util/auxiliary/acl.json",
    "css:config/util/identifiers/suffix.json",
    "css:config/util/index/default.json",
    "css:config/util/logging/winston.json",
    "css:config/util/representation-conversion/default.json",
    "css:config/util/resource-locker/memory.json",
    "css:config/util/variables/default.json"
  ],
  "@graph": [
    {
      "comment": "A single-pod server that uses a memory backend and supports registration."
    }
  ]
}
```

### 2.3 SSL Certificate Setup for HTTPS

Community Solid Server requires HTTPS for production-like testing. Generate a self-signed certificate:

```bash
# Create SSL directory
mkdir -p ~/solid-dev/ssl
cd ~/solid-dev/ssl

# Generate self-signed certificate (valid for 1 year)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/C=US/ST=State/L=City/O=Development/CN=localhost"
```

**Add certificate to trusted certificates:**

**macOS:**
```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/solid-dev/ssl/cert.pem
```

**Linux:**
```bash
sudo cp ~/solid-dev/ssl/cert.pem /usr/local/share/ca-certificates/solid-dev.crt
sudo update-ca-certificates
```

**Windows:**
```powershell
# Import certificate to Trusted Root Certification Authorities
Import-Certificate -FilePath ~/solid-dev/ssl/cert.pem -CertStoreLocation Cert:\LocalMachine\Root
```

### 2.4 Running the Local Server

Start Community Solid Server:

```bash
community-solid-server \
  --config ~/solid-dev/css-config/config.json \
  --port 3000 \
  --httpsKey ~/solid-dev/ssl/key.pem \
  --httpsCert ~/solid-dev/ssl/cert.pem \
  --baseUrl https://localhost:3000/ \
  --seedConfig ~/solid-dev/css-config/seed.json
```

**Create a startup script for convenience:**

`~/solid-dev/start-css.sh`:
```bash
#!/bin/bash
echo "🚀 Starting Community Solid Server..."
community-solid-server \
  --config ~/solid-dev/css-config/config.json \
  --port 3000 \
  --httpsKey ~/solid-dev/ssl/key.pem \
  --httpsCert ~/solid-dev/ssl/cert.pem \
  --baseUrl https://localhost:3000/ \
  --loggingLevel debug
```

Make executable:
```bash
chmod +x ~/solid-dev/start-css.sh
```

Run:
```bash
~/solid-dev/start-css.sh
```

### 2.5 Verify Local Server

Test the local server:

```bash
# Check server is running
curl -k https://localhost:3000/.well-known/openid-configuration

# Should return OIDC configuration JSON
```

Access in browser: https://localhost:3000/

**Expected:** You should see the Community Solid Server interface.

---

## 3. Browser Configuration

### 3.1 Trust Self-Signed Certificate

If you see certificate warnings in your browser:

**Chrome/Edge:**
1. Navigate to https://localhost:3000/
2. Click "Advanced"
3. Click "Proceed to localhost (unsafe)"

**Firefox:**
1. Navigate to https://localhost:3000/
2. Click "Advanced"
3. Click "Accept the Risk and Continue"

**Safari:**
1. Navigate to https://localhost:3000/
2. Click "Show Details"
3. Click "visit this website"

### 3.2 Browser Extensions (Optional)

**Penny** - Solid Pod browser extension:
- Chrome: https://chrome.google.com/webstore/detail/penny/
- Firefox: https://addons.mozilla.org/en-US/firefox/addon/penny/

**Benefits:**
- View Pod contents directly
- Manage permissions
- Debug authentication issues

---

## 4. Development Tools

### 4.1 Solid Libraries (for testing)

Install Solid client libraries globally for CLI testing:

```bash
npm install -g @inrupt/solid-client @inrupt/solid-client-authn-node
```

### 4.2 RDF Tools

Install RDF validation and debugging tools:

```bash
# For validating Turtle syntax
npm install -g rdf-validate-shacl

# For pretty-printing RDF
npm install -g rdf-toolkit
```

### 4.3 HTTP Debugging

Install tools for debugging HTTP requests to Pods:

**httpie** (better curl):
```bash
# macOS
brew install httpie

# Linux
apt-get install httpie

# Test with a Pod
http GET https://[your-pod].inrupt.net/profile/card
```

**Postman** - GUI HTTP client:
- Download from: https://www.postman.com/downloads/

---

## 5. Project-Specific Setup

### 5.1 Environment Variables

The project already has Solid environment variables configured in `.env.ec2`:

```bash
# Solid Project Integration (Beta)
SOLID_ENABLED=true
SOLID_CLIENT_ID=2f585e37-e435-41b8-b9bf-9d338a1b945a
SOLID_CLIENT_SECRET=[your-secret]
SOLID_REDIRECT_URI=https://notspies.org/api/solid/callback
```

For local development, create `.env.local`:

```bash
SOLID_ENABLED=true
SOLID_CLIENT_ID=location-tracker-dev
SOLID_CLIENT_SECRET=
SOLID_REDIRECT_URI=https://localhost:8080/api/solid/callback
```

### 5.2 Test the Current PoC

The Solid PoC is already deployed and functional:

```bash
# Test the providers endpoint
curl https://notspies.org/api/solid/providers | jq

# Test the login initiation
curl -X POST https://notspies.org/api/solid/login \
  -H "Content-Type: application/json" \
  -d '{"issuer_url":"https://solidcommunity.net"}' | jq
```

### 5.3 Local Development

Run the location-tracker locally:

```bash
cd location-tracker

# Build
go build -o location-tracker .

# Run with Solid enabled
SOLID_ENABLED=true \
SOLID_CLIENT_ID=location-tracker-dev \
SOLID_REDIRECT_URI=https://localhost:8080/api/solid/callback \
./location-tracker
```

Access at: https://localhost:8080

---

## 6. Verification Checklist

### ✅ Environment Setup Complete

- [ ] Inrupt PodSpaces account created and tested
- [ ] SolidCommunity.net account created and tested
- [ ] Community Solid Server installed
- [ ] SSL certificates generated and trusted
- [ ] Local Solid server runs on https://localhost:3000
- [ ] Browser can access local server without warnings
- [ ] Pod credentials documented in `~/.solid-pods`
- [ ] HTTP debugging tools installed (httpie or Postman)
- [ ] Project builds successfully with `go build`
- [ ] Can access production PoC at https://notspies.org
- [ ] SOLID_DEV_SETUP.md exists and is complete

---

## 7. Next Steps

After completing this setup:

1. **Issue #46:** Research and document Solid authentication flows
   - Study OIDC + DPoP specifications
   - Test authentication with your Pod accounts
   - Document the complete flow

2. **Issue #47:** Research and design RDF data models
   - Learn Turtle and JSON-LD formats
   - Design Location and ErrorLog schemas
   - Create example RDF files

3. **Issue #48:** Build authentication proof-of-concept
   - Create standalone HTML PoC
   - Test with local Solid server
   - Document learnings

---

## 8. Troubleshooting

### Issue: "Cannot connect to Community Solid Server"

**Solution:**
```bash
# Check if port 3000 is available
lsof -i :3000

# Kill any process using port 3000
kill -9 [PID]

# Restart server
~/solid-dev/start-css.sh
```

### Issue: "Certificate not trusted"

**Solution:**
```bash
# Re-add certificate to trusted store
# macOS:
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/solid-dev/ssl/cert.pem

# Restart browser
```

### Issue: "Pod not accessible"

**Solution:**
```bash
# Test DNS resolution
nslookup your-pod.inrupt.net

# Test with curl (ignore SSL)
curl -k https://your-pod.inrupt.net/profile/card

# Check Pod provider status
# Inrupt: https://status.inrupt.com/
# SolidCommunity: https://solidcommunity.net/
```

### Issue: "Go build fails"

**Solution:**
```bash
# Update Go dependencies
cd location-tracker
go mod tidy
go mod download

# Rebuild
go build -o location-tracker .
```

---

## 9. Resources

### Official Documentation
- **Solid Protocol Spec:** https://solidproject.org/TR/protocol
- **Solid OIDC Spec:** https://solidproject.org/TR/oidc
- **WebID Spec:** https://www.w3.org/2005/Incubator/webid/spec/
- **Community Solid Server:** https://github.com/CommunitySolidServer/CommunitySolidServer

### Libraries
- **Inrupt JavaScript Client:** https://docs.inrupt.com/developer-tools/javascript/client-libraries/
- **Inrupt JavaScript Auth:** https://docs.inrupt.com/developer-tools/javascript/client-libraries/authentication/

### Learning Resources
- **Solid Tutorial:** https://solidproject.org/developers/tutorials/getting-started
- **RDF Primer:** https://www.w3.org/TR/rdf-primer/
- **Turtle Spec:** https://www.w3.org/TR/turtle/

### Community
- **Solid Forum:** https://forum.solidproject.org/
- **Solid Gitter Chat:** https://gitter.im/solid/chat
- **Solid GitHub:** https://github.com/solid

---

## 10. Summary

**Status:** ✅ Development environment ready

**What we have:**
- 2 test Pod accounts (Inrupt + SolidCommunity)
- Local Solid server running on HTTPS
- SSL certificates trusted in browser
- Development tools installed
- Project configured for Solid development

**What's next:**
- Research authentication flows (#46)
- Design RDF data models (#47)
- Build PoC application (#48)

---

**Last Updated:** 2025-11-12
**Maintained By:** Development Team
**Related Issues:** #45, #46, #47, #48
