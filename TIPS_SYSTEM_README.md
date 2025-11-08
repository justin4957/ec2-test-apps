# Anonymous "Not Spy Work" Tips System

Complete implementation of an anonymous tip submission system with OpenAI content moderation, reversible hashing, rate limiting, and DynamoDB persistence.

## 🎯 Features

### Core Functionality
- ✅ **Anonymous Tip Submission**: Users can submit tips via HTML form
- ✅ **Reversible Anonymous IDs**: AES-256 encrypted user IDs (format: `user_abc123def456`)
- ✅ **OpenAI Content Moderation**: Two-stage filtering with Moderation API + GPT-4 redaction
- ✅ **Rate Limiting**: 10 tips per hour per user (configurable)
- ✅ **Abuse Prevention**: Ban system with DynamoDB persistence
- ✅ **DynamoDB Storage**: Full persistence of tips and ban records
- ✅ **Error Log Integration**: Tips automatically attached to error logs (similar to Twilio SMS)

### Safety Features
- 🔒 **PII Redaction**: Automatic removal of emails, phone numbers, addresses, SSNs, credit cards, IPs, URLs
- 🚫 **Content Filtering**: OpenAI Moderation API blocks hate speech, violence, sexual content, self-harm
- ⏱️ **Rate Limiting**: Prevents spam (10 submissions per hour per anonymous user)
- 🔨 **Ban System**: Temporary/permanent bans for repeat abusers
- 🔐 **Encrypted Metadata**: User metadata encrypted with AES-256-GCM
- 🕵️ **Reversible Hashing**: Admin can reverse hash for abuse investigation

## 📂 File Structure

```
location-tracker/
├── main.go                    # Main server with tip endpoints & HTML
├── identity.go                # Reversible anonymous ID generation
├── content_moderation.go      # OpenAI moderation integration
├── rate_limiter.go            # Rate limiting & ban management
└── location-tracker           # Compiled binary

Root:
├── create-tips-tables.sh      # DynamoDB table creation script
└── TIPS_SYSTEM_README.md      # This file
```

## 🚀 Setup Instructions

### 1. Prerequisites

```bash
# Install Go 1.21+
# Install AWS CLI v2
# Configure AWS credentials
aws configure
```

### 2. Generate Encryption Key

```bash
# Generate 32-byte (64 hex chars) encryption key
openssl rand -hex 32
# Example output: a3f8c2e1d4b7a9f6c5e8d2b4a7f9c6e3d8b5a2f7c4e9d6b3a8f5c2e7d4b9a6f3
```

### 3. Set Environment Variables

```bash
# Required
export TRACKER_PASSWORD=your_password_here
export OPENAI_API_KEY=sk-...  # For content moderation
export TIP_ENCRYPTION_KEY=a3f8c2e1d4b7a9f6c5e8d2b4a7f9c6e3d8b5a2f7c4e9d6b3a8f5c2e7d4b9a6f3

# Optional
export USE_HTTPS=true
export HTTP_PORT=8080
export HTTPS_PORT=8443
```

### 4. Create DynamoDB Tables

```bash
# Make script executable
chmod +x create-tips-tables.sh

# Create tables
./create-tips-tables.sh

# Verify tables
aws dynamodb list-tables --region us-east-1
```

**Tables Created:**
- `location-tracker-anonymous-tips` (stores tips with GSI for querying)
- `location-tracker-banned-users` (stores banned user hashes)

### 5. Build and Run

```bash
cd location-tracker
go mod tidy
go build -o location-tracker .
./location-tracker
```

Server runs on:
- HTTP: `http://localhost:8080` (for webhooks)
- HTTPS: `https://localhost:8443` (for browser access)

## 📡 API Endpoints

### POST `/api/tips`
Submit an anonymous tip.

**Request:**
```json
{
  "tip_content": "I saw someone not doing spy work at the office today."
}
```

**Response (Success):**
```json
{
  "status": "success",
  "tip_id": "1234567890123456789",
  "user_hash": "user_abc123def456",
  "moderated": false,
  "reason": ""
}
```

**Response (Rejected):**
```json
{
  "status": "rejected",
  "reason": "Content flagged for: hate, violence"
}
```

**Response (Rate Limited):**
```json
{
  "status": "rate_limited",
  "reason": "Too many submissions. Please try again later.",
  "reset_time": "2025-01-15T10:30:00Z"
}
```

**Headers:**
- `X-RateLimit-Remaining`: Number of submissions left this hour
- `X-RateLimit-Reset`: Unix timestamp when limit resets

### GET `/api/tips`
Retrieve recent approved tips (public, no auth required).

**Response:**
```json
{
  "tips": [
    {
      "id": "1234567890123456789",
      "tip_content": "Original text",
      "moderated_content": "Redacted text with [EMAIL_REDACTED]",
      "user_hash": "user_abc123def456",
      "moderation_status": "redacted",
      "moderation_reason": "Sensitive information redacted",
      "keywords": ["office", "work", "suspicious"],
      "timestamp": "2025-01-15T10:15:00Z"
    }
  ],
  "count": 1
}
```

### GET `/api/tips/:id`
Retrieve a specific tip by ID.

**Response:**
```json
{
  "id": "1234567890123456789",
  "tip_content": "Original text",
  "moderated_content": "Moderated text",
  "user_hash": "user_abc123def456",
  "user_metadata": "base64_encrypted_metadata",
  "moderation_status": "approved",
  "timestamp": "2025-01-15T10:15:00Z",
  "ip_address": "192.168.1.1"
}
```

## 🎨 Frontend Integration

The tip submission form is automatically included in the main HTML interface:

```html
<!-- Form appears after login/cryptogram solution -->
<div id="tracker">
  <!-- Tip Submission Form -->
  <h3>🕵️ Report Not Spy Work</h3>
  <textarea id="tip-content" maxlength="1000"></textarea>
  <button onclick="submitTip()">📝 Submit Anonymous Tip</button>
</div>
```

**JavaScript API:**
```javascript
// Submit a tip
async function submitTip() {
  const res = await fetch('/api/tips', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({tip_content: content})
  });
  const result = await res.json();
  // Handle result.status: success, rejected, rate_limited, banned
}
```

## 🔐 Security Architecture

### Anonymous Identity System

```
User Request → Extract Metadata → Encrypt with AES-256 → Generate SHA-256 Hash → Display ID
                    ↓                      ↓                        ↓
              IP + UserAgent         Base64 Encoded           user_abc123def456
                    ↓
         Store encrypted metadata in DB (reversible for investigation)
```

**Encryption Details:**
- Algorithm: AES-256-GCM
- Key: 32 bytes (256 bits)
- Nonce: 12 bytes (randomly generated per encryption)
- Output: Base64-encoded ciphertext

**Hash Generation:**
```
SHA-256(encrypted_metadata) → first 6 bytes → hex encode → "user_" + hex
```

### Content Moderation Pipeline

```
User Input
    ↓
Basic Validation (length, format)
    ↓
OpenAI Moderation API (check ToS violations)
    ↓ (if passed)
Pattern-Based PII Redaction (emails, phones, SSNs, etc.)
    ↓
Store in DynamoDB with status: approved/rejected/redacted
```

**Redaction Patterns:**
- Email: `[EMAIL_REDACTED]`
- Phone: `[PHONE_REDACTED]`
- SSN: `[SSN_REDACTED]`
- Credit Card: `[CARD_REDACTED]`
- Address: `[ADDRESS_REDACTED]`
- IP Address: `[IP_REDACTED]`
- URL: `[URL_REDACTED]`

### Rate Limiting

```
User submits tip
    ↓
Check submissions in last hour (stored in memory)
    ↓
If < 10: Allow + Record timestamp
If ≥ 10: Reject with reset_time
```

**Implementation:**
- In-memory map: `user_hash → []timestamp`
- Cleanup: Every 5 minutes
- Limit: 10 per hour (configurable via `tipRateLimit`)

### Ban Management

```
Admin/System detects abuse
    ↓
Ban user: BanUser(user_hash, duration, reason)
    ↓
Store in DynamoDB: location-tracker-banned-users
    ↓
All future submissions blocked until expiry
```

## 📊 DynamoDB Schema

### Table: `location-tracker-anonymous-tips`

**Primary Key:**
- Partition Key: `id` (String) - Nanosecond timestamp

**Global Secondary Indexes:**
1. **user_hash-timestamp-index**
   - Partition Key: `user_hash`
   - Sort Key: `timestamp`
   - Use: Query all tips from a specific user

2. **moderation_status-timestamp-index**
   - Partition Key: `moderation_status`
   - Sort Key: `timestamp`
   - Use: Query approved/rejected/redacted tips

**Attributes:**
```
id                 String    Primary key
tip_content        String    Original content
moderated_content  String    After PII redaction
user_hash          String    Anonymous ID
user_metadata      String    Encrypted metadata (reversible)
moderation_status  String    approved/rejected/redacted
moderation_reason  String    Why rejected/redacted
keywords           []String  Extracted keywords
timestamp          String    ISO 8601 timestamp
ip_address         String    For abuse detection
```

### Table: `location-tracker-banned-users`

**Primary Key:**
- Partition Key: `user_hash` (String)

**Attributes:**
```
user_hash   String    Anonymous user ID
ban_expiry  String    ISO 8601 timestamp
reason      String    Why banned
banned_at   String    When banned
banned_by   String    Who banned (admin username or "system")
```

## 🔧 Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `TRACKER_PASSWORD` | ✅ | - | Password for full login |
| `OPENAI_API_KEY` | ⚠️ | - | Required for AI moderation (falls back to patterns) |
| `TIP_ENCRYPTION_KEY` | ⚠️ | Auto-generated | 64 hex chars (32 bytes). Auto-generates if not set (won't persist) |
| `USE_HTTPS` | ❌ | false | Enable HTTPS mode |
| `HTTP_PORT` | ❌ | 8080 | HTTP server port |
| `HTTPS_PORT` | ❌ | 8443 | HTTPS server port |

### In-Code Configuration

```go
// In main.go global variables
tipMaxLength       = 1000  // Max characters per tip
tipRateLimit       = 10    // Tips per hour per user

// Modify these values to adjust limits
```

## 🧪 Testing

### Manual Testing

```bash
# Start server
./location-tracker

# Submit a tip (in another terminal)
curl -X POST http://localhost:8080/api/tips \
  -H "Content-Type: application/json" \
  -d '{"tip_content":"I saw Bob eating lunch instead of spying. Very suspicious."}'

# Expected response:
# {"status":"success","tip_id":"1234...","user_hash":"user_abc123...","moderated":false}

# Test PII redaction
curl -X POST http://localhost:8080/api/tips \
  -H "Content-Type: application/json" \
  -d '{"tip_content":"Contact me at test@example.com or 555-123-4567"}'

# Expected: Content redacted with [EMAIL_REDACTED] and [PHONE_REDACTED]

# Test rate limiting (submit 11 times rapidly)
for i in {1..11}; do
  curl -X POST http://localhost:8080/api/tips \
    -H "Content-Type: application/json" \
    -d "{\"tip_content\":\"Test tip $i\"}"
done

# Expected: 11th submission should be rate_limited
```

### Browser Testing

1. Open `https://localhost:8443` (accept self-signed cert)
2. Login with password or solve cryptogram
3. Scroll to "🕵️ Report Not Spy Work" section
4. Enter a tip and submit
5. Check that anonymous ID is displayed
6. Generate an error (via error-generator)
7. Verify tip appears in the error log

### DynamoDB Verification

```bash
# View all tips
aws dynamodb scan --table-name location-tracker-anonymous-tips \
  --region us-east-1

# Query tips by user hash
aws dynamodb query --table-name location-tracker-anonymous-tips \
  --index-name user_hash-timestamp-index \
  --key-condition-expression "user_hash = :hash" \
  --expression-attribute-values '{":hash":{"S":"user_abc123def456"}}' \
  --region us-east-1

# View approved tips only
aws dynamodb query --table-name location-tracker-anonymous-tips \
  --index-name moderation_status-timestamp-index \
  --key-condition-expression "moderation_status = :status" \
  --expression-attribute-values '{":status":{"S":"approved"}}' \
  --region us-east-1
```

## 🛠️ Administration

### Ban a User

```go
// In Go code or admin endpoint
banManager.BanUser("user_abc123def456", 24*time.Hour, "Spam")
// Bans for 24 hours
```

### Unban a User

```go
banManager.UnbanUser("user_abc123def456")
```

### Reverse Anonymous Hash

```go
// Admin-only: Decrypt user metadata
metadata, err := identityManager.ReverseHash(encryptedMetadata)
// Returns: IP address, User-Agent, timestamp
```

### Check Rate Limit Status

```go
remaining := rateLimiter.GetRemainingQuota("user_abc123def456")
fmt.Printf("User has %d tips remaining this hour\n", remaining)
```

## 📈 Monitoring & Logs

### Log Messages

```
📝 Anonymous tip submitted: 1234567890123456789 (status: approved, user: user_abc123def456)
💾 Anonymous tip saved to DynamoDB: 1234567890123456789
📝 Attached 3 anonymous tips to error log
⚠️  Content flagged for: hate, violence
⚠️  OpenAI Moderation API failed: <error>
```

### Metrics to Monitor

- Tips per hour (rate limit effectiveness)
- Rejection rate (content quality)
- Redaction rate (PII detection)
- Ban count (abuse level)
- OpenAI API errors (moderation health)

## 🎯 Integration with Error Logs

Tips are automatically attached to error logs:

```go
// When error log is created (POST /api/errorlogs)
errorLog.AnonymousTips = []string{"tip_id_1", "tip_id_2", "tip_id_3"}

// Frontend fetches tip details and displays alongside error
```

**Display Example:**
```
📝 Error Log
Message: Database connection timeout
Slogan: "When your database takes a coffee break"

🕵️ Anonymous Not-Spy-Work Tips
  user_abc123def456 | 2025-01-15 10:15 AM
  "I saw Bob eating lunch instead of spying. Very suspicious."

  user_xyz789abc123 | 2025-01-15 10:12 AM  [Redacted]
  "Contact info: [EMAIL_REDACTED]"
```

## 🔮 Future Enhancements

- [ ] Admin dashboard for reviewing flagged content
- [ ] Email notifications for high-priority tips
- [ ] Sentiment analysis on tip content
- [ ] Tip voting/ranking system
- [ ] Export tips to CSV/JSON
- [ ] Webhook notifications when tips are submitted
- [ ] Multi-language support
- [ ] Image attachments (with OCR + moderation)

## 📝 License

Part of the ec2-test-apps project.

## 🤝 Contributing

When modifying the system:
1. Update `TIPS_SYSTEM_README.md` with changes
2. Test all safety features (rate limiting, moderation, bans)
3. Verify DynamoDB persistence
4. Check that tips display correctly in error logs
5. Run `go build` to ensure compilation

## ❓ Troubleshooting

### "TIP_ENCRYPTION_KEY must be 32 bytes"
**Solution:** Generate a new key with `openssl rand -hex 32` and set as environment variable.

### "OpenAI Moderation API failed"
**Solution:** Check `OPENAI_API_KEY` is valid. System will fall back to pattern-based filtering.

### "DynamoDB table not accessible"
**Solution:** Run `./create-tips-tables.sh` and verify AWS credentials with `aws sts get-caller-identity`.

### Tips not appearing in error logs
**Solution:** Tips are attached only to NEW error logs. Generate a new error after submitting tips.

### Rate limit not working
**Solution:** Clear browser cookies to reset anonymous ID. Each browser session gets a unique hash.

---

**Built with ❤️ for anonymous reporting of definitely-not-spy-work activities.**
