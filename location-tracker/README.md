# 📍 Personal Location Tracker

Educational personal security project to track your own devices and share location with trusted people.

## Purpose

- Track location of **YOUR OWN** devices (phone, laptop, etc.)
- Share access with trusted friends/family via password
- Educational demonstration of geolocation APIs
- Personal security tool (e.g., "Find My Device" alternative)

## ⚠️ Legal & Ethical Use ONLY

**This tool is ONLY for:**
- ✅ Tracking devices YOU own
- ✅ With EXPLICIT consent from anyone tracked
- ✅ Educational purposes
- ✅ Personal security

**NEVER use for:**
- ❌ Tracking others without consent (ILLEGAL)
- ❌ Stalking or surveillance
- ❌ Any malicious purposes

## How It Works

```
1. You open the tracker URL on any device
2. Login with shared password
3. Click "Share Location" to update your position
4. Anyone with the password can see all shared locations
5. Locations auto-expire after 24 hours
```

## Features

- 🔒 Password-protected access
- 🔐 HTTPS support with automatic self-signed certificates
- 📱 Works on any device (phone, tablet, laptop)
- 🗺️ Direct links to Google Maps
- ⏱️ Real-time updates (auto-refresh every 10s)
- 🧹 Auto-cleanup (locations >24h deleted)
- 📊 Shows location accuracy and age
- 💾 In-memory storage (no database needed)

## Local Testing

### HTTP Mode (Default)
```bash
# Set password
export TRACKER_PASSWORD=your_secure_password_here

# Run
go run main.go

# Open browser
open http://localhost:8080
```

**Note:** Geolocation may not work in HTTP mode on some browsers. Use HTTPS mode below for full functionality.

### HTTPS Mode (Recommended)
```bash
# Set password and enable HTTPS
export TRACKER_PASSWORD=your_secure_password_here
export USE_HTTPS=true

# Run (will auto-generate self-signed certificate)
go run main.go

# Open browser (accept certificate warning)
open https://localhost:8443
```

**Browser Certificate Warning:** Self-signed certificates will trigger a security warning. This is expected for local testing. Click "Advanced" and proceed to the site.

### HTTPS with Custom Certificates
```bash
# Use your own certificates
export TRACKER_PASSWORD=your_secure_password_here
export USE_HTTPS=true
export CERT_FILE=/path/to/server.crt
export KEY_FILE=/path/to/server.key

go run main.go
```

## Docker Deployment

### HTTP Mode
```bash
# Build
docker build -t location-tracker .

# Run
docker run -d \
  --name location-tracker \
  -p 8082:8080 \
  -e TRACKER_PASSWORD=your_secure_password_here \
  --restart unless-stopped \
  location-tracker
```

### HTTPS Mode
```bash
# Build
docker build -t location-tracker .

# Run with HTTPS (auto-generates self-signed certificate)
docker run -d \
  --name location-tracker \
  -p 8443:8443 \
  -e TRACKER_PASSWORD=your_secure_password_here \
  -e USE_HTTPS=true \
  --restart unless-stopped \
  location-tracker

# OR with custom certificates
docker run -d \
  --name location-tracker \
  -p 8443:8443 \
  -e TRACKER_PASSWORD=your_secure_password_here \
  -e USE_HTTPS=true \
  -e CERT_FILE=/certs/server.crt \
  -e KEY_FILE=/certs/server.key \
  -v /path/to/certs:/certs:ro \
  --restart unless-stopped \
  location-tracker
```

## EC2 Deployment

### Add to existing EC2 instance:

```bash
# 1. Update .env.ec2 with password
echo "TRACKER_PASSWORD=your_secure_password_here" >> .env.ec2

# 2. Build and push to ECR
cd location-tracker
docker buildx build --platform linux/amd64 -t location-tracker .

aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 310829530225.dkr.ecr.us-east-1.amazonaws.com

# Create ECR repository
aws ecr create-repository --repository-name location-tracker --region us-east-1

docker tag location-tracker:latest \
  310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest

docker push 310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest

# 3. Deploy to EC2
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@your-ec2-instance.amazonaws.com << 'EOF'
  aws ecr get-login-password --region us-east-1 | \
    docker login --username AWS --password-stdin 310829530225.dkr.ecr.us-east-1.amazonaws.com

  docker pull 310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest

  docker stop location-tracker 2>/dev/null || true
  docker rm location-tracker 2>/dev/null || true

  docker run -d \
    --name location-tracker \
    --restart unless-stopped \
    -p 8082:8080 \
    -e TRACKER_PASSWORD=your_secure_password_here \
    310829530225.dkr.ecr.us-east-1.amazonaws.com/location-tracker:latest
EOF

# 4. Open port 8082 in security group
aws ec2 authorize-security-group-ingress \
  --group-id sg-0b854584c1f195ecf \
  --protocol tcp --port 8082 --cidr 0.0.0.0/0 \
  --region us-east-1
```

Access at: `http://your-ec2-instance.amazonaws.com:8082`

## Browser Permissions & HTTPS

Modern browsers **require HTTPS** for geolocation APIs to work (except on localhost). The application now supports HTTPS natively with automatic self-signed certificate generation.

### HTTPS Configuration Options

**Option 1: Built-in Self-Signed Certificates (Easiest)**
- Set `USE_HTTPS=true` environment variable
- Application auto-generates certificates on first run
- Perfect for local testing and development
- You'll need to accept the browser's security warning

**Option 2: Custom Certificates (Production)**
- Set `USE_HTTPS=true`
- Provide `CERT_FILE` and `KEY_FILE` paths
- Use certificates from Let's Encrypt, your CA, or cloud provider

**Option 3: Reverse Proxy (Advanced)**
- Keep application in HTTP mode
- Use nginx/Apache/Caddy with SSL termination
- Proxy HTTPS requests to HTTP backend

### Production HTTPS Options

**Let's Encrypt (Free)**
```bash
# Get certificate with certbot
certbot certonly --standalone -d yourdomain.com

# Run with certificates
export TRACKER_PASSWORD=your_password
export USE_HTTPS=true
export CERT_FILE=/etc/letsencrypt/live/yourdomain.com/fullchain.pem
export KEY_FILE=/etc/letsencrypt/live/yourdomain.com/privkey.pem
go run main.go
```

**Cloudflare Tunnel (Free, No Certificates Needed)**
```bash
# Install cloudflared
# Run app in HTTP mode
# Create tunnel: cloudflared tunnel --url http://localhost:8080
# Get free HTTPS URL with automatic certificates
```

## Security Considerations

### Current Implementation
- ✅ Password authentication
- ✅ HTTPS support with auto-generated certificates
- ✅ HTTP-only cookies with Secure flag (when using HTTPS)
- ✅ 2-second delay on failed login (brute force prevention)
- ✅ Auto-expiring locations (24h)
- ✅ In-memory storage (no persistent data)

### Production Enhancements (Optional)
- 🔑 Use bcrypt for password hashing
- 📝 Add session management with tokens
- 🚦 Rate limiting on login endpoint
- 🌐 IP whitelisting
- 📊 Logging and monitoring
- 💾 Database storage for persistence
- 👥 Multi-user support
- 🔔 Geofencing alerts

## API Endpoints

### POST /api/login
Login with password
```json
{
  "password": "your_password"
}
```

### POST /api/location
Share your location (requires auth)
```json
{
  "latitude": 37.7749,
  "longitude": -122.4194,
  "accuracy": 20.0,
  "device_id": "device_abc123"
}
```

### GET /api/location
Get all locations (requires auth)
```json
{
  "device_abc123": {
    "latitude": 37.7749,
    "longitude": -122.4194,
    "accuracy": 20.0,
    "timestamp": "2025-10-29T12:00:00Z",
    "device_id": "device_abc123",
    "user_agent": "Mozilla/5.0..."
  }
}
```

### GET /api/health
Health check (no auth required)
```json
{
  "status": "ok"
}
```

## Privacy & Data

- **Storage**: In-memory only (resets on restart)
- **Retention**: 24 hours maximum
- **Sharing**: Only with people who have the password
- **Encryption**: Passwords should be strong; consider HTTPS in production
- **Deletion**: Automatic after 24h, or restart the service

## Use Cases

1. **Personal Device Tracking**: Know where you left your laptop
2. **Family Safety**: Share location during travel
3. **Group Coordination**: Friends meeting up at an event
4. **Emergency Contact**: Let trusted people find you in emergency
5. **Educational**: Learn about geolocation APIs and web security

## Limitations

- No persistent storage (restarts clear data)
- Single global password (everyone sees all locations)
- Self-signed certificates require browser security exception
- No mobile app (web-based only)
- Browser must allow geolocation

## Future Enhancements

Ideas for educational extensions:
- 📱 Mobile app (React Native)
- 🗺️ Embedded map view (Leaflet.js)
- 📍 Location history and trails
- 🔔 Geofencing notifications
- 👥 Multiple user accounts
- 💬 Location notes/messages
- 🔋 Battery status tracking
- 🚗 Speed and movement tracking

## License

Educational purposes only. Use responsibly and legally.
