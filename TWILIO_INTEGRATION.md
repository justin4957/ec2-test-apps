# Twilio SMS Integration

This feature allows users to send SMS messages via Twilio that will be attached to the next error log as a "user experience note".

## How It Works

1. **SMS Reception**: When a user sends an SMS to your Twilio phone number, Twilio forwards the message to the `/api/twilio/sms` webhook endpoint
2. **Message Storage**: The SMS body is stored temporarily in memory as a pending user experience note
3. **Attachment**: When the next error log arrives from the error-generator service, the pending note is automatically attached to it
4. **Clearing**: After attachment, the pending note is cleared, ready for the next SMS

## Architecture

### Components Modified

- **location-tracker/main.go**:
  - Added `UserExperienceNote` field to `ErrorLog` struct
  - Added `pendingUserExperienceNote` global variable with mutex for thread-safe access
  - Added `handleTwilioWebhook()` endpoint handler
  - Modified `handleErrorLogs()` to attach pending notes to incoming errors

### Data Flow

```
User sends SMS → Twilio → POST /api/twilio/sms → Store message
                                                        ↓
Error Generator → POST /api/errorlogs → Attach note → Store error → Display in UI
```

## Setup Instructions

### 1. Configure Twilio Webhook

**IMPORTANT:** Twilio requires either:
- A valid SSL certificate from a trusted Certificate Authority (CA), OR
- An HTTP endpoint (no SSL)

Self-signed certificates will NOT work with Twilio webhooks.

**For EC2 Deployment (Recommended):**

Use the HTTP endpoint on port 8081:

In your Twilio Console:
1. Go to your phone number configuration
2. Under "Messaging", set the webhook URL for incoming messages:
   ```
   http://your-ec2-dns:8081/api/twilio/sms
   ```
   Example:
   ```
   http://ec2-54-226-246-133.compute-1.amazonaws.com:8081/api/twilio/sms
   ```
3. Set HTTP method to `POST`
4. Set webhook format to `application/x-www-form-urlencoded` (default)

**For Production with Valid SSL:**
```
https://your-domain.com/api/twilio/sms
```

**Note:** The deployment script (`deploy-to-ec2.sh`) automatically exposes both:
- Port 8081 (HTTP) for Twilio webhooks
- Port 8082 (HTTPS with self-signed cert) for browser access

### 2. Test Locally

#### Test the webhook endpoint directly:
```bash
# For local testing (HTTP)
curl -X POST http://localhost:8080/api/twilio/sms \
  -d "Body=The+app+is+really+slow+today" \
  -d "From=%2B15551234567" \
  -d "MessageSid=SM1234567890abcdef"

# For EC2 deployment (HTTP - port 8081)
curl -X POST http://your-ec2-dns:8081/api/twilio/sms \
  -d "Body=The+app+is+really+slow+today" \
  -d "From=%2B15551234567" \
  -d "MessageSid=SM1234567890abcdef"
```

#### Expected response:
```xml
<?xml version="1.0" encoding="UTF-8"?><Response></Response>
```

#### Check the logs:
```
📱 Received SMS from +15551234567 (SID: SM1234567890abcdef): The app is really slow today
💬 Stored user experience note, will attach to next error log
```

### 3. Test End-to-End

1. Send an SMS to your Twilio number with your feedback:
   ```
   "User reported slow page loads on checkout"
   ```

2. Wait for the next error to be generated (or trigger one manually)

3. Check the location-tracker UI or API to see the error with the attached note:
   ```json
   {
     "id": "1699123456789000000",
     "message": "ConnectionTimeoutException: Unable to reach database",
     "gif_url": "https://giphy.com/gifs/...",
     "slogan": "Keep Calm and Restart",
     "song_title": "Fix You",
     "song_artist": "Coldplay",
     "user_experience_note": "User reported slow page loads on checkout",
     "timestamp": "2025-11-02T10:30:45Z"
   }
   ```

### 4. Deploy to EC2

The webhook endpoint is automatically available when you deploy the location-tracker service to EC2. Make sure:
- HTTPS is enabled (`USE_HTTPS=true`)
- Port 8443 is open in your security group
- Twilio can reach your EC2 instance publicly

## API Reference

### POST /api/twilio/sms

Receives SMS webhook from Twilio.

**Request** (application/x-www-form-urlencoded):
```
MessageSid=SM1234567890abcdef
Body=The actual SMS text content
From=+15551234567
To=+15559876543
```

**Response** (application/xml):
```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response></Response>
```

**Status Codes**:
- `200 OK`: Message received and stored
- `400 Bad Request`: Invalid request or empty message body
- `405 Method Not Allowed`: Only POST is supported

## Database Schema

The `user_experience_note` field is automatically persisted to DynamoDB when available:

```go
type ErrorLog struct {
    ID                  string    `dynamodbav:"id"`
    Message             string    `dynamodbav:"message"`
    GifURL              string    `dynamodbav:"gif_url"`
    Slogan              string    `dynamodbav:"slogan"`
    SongTitle           string    `dynamodbav:"song_title"`
    SongArtist          string    `dynamodbav:"song_artist"`
    SongURL             string    `dynamodbav:"song_url"`
    UserExperienceNote  string    `dynamodbav:"user_experience_note"`  // NEW
    Timestamp           time.Time `dynamodbav:"timestamp"`
}
```

## UI Display

User experience notes are displayed in the error log cards with:
- 💬 Icon for visual identification
- Green highlight background (#f0fdf4)
- Green left border (3px solid #10b981)
- Bold text for emphasis

## Limitations

1. **Single Note**: Only one pending note is stored at a time. If multiple SMS messages are sent before an error occurs, only the most recent message is attached.

2. **In-Memory Storage**: The pending note is stored in memory and will be lost if the service restarts before an error is logged.

3. **No Authentication**: The webhook endpoint does not require authentication. Consider adding Twilio signature validation for production use.

4. **One-Time Use**: Each note is attached to exactly one error and then cleared. To attach the same note to multiple errors, send the SMS multiple times.

## Future Enhancements

- [ ] Queue multiple pending notes instead of keeping only the latest
- [ ] Add Twilio request signature validation
- [ ] Persist pending notes to DynamoDB for durability across restarts
- [ ] Add SMS reply functionality to confirm note was received
- [ ] Support for attaching notes to specific error types or services
- [ ] Admin API to view/manage pending notes

## Troubleshooting

### Issue: SMS sent but note not attached

**Possible causes**:
1. Twilio webhook not configured correctly
2. Service is not publicly accessible
3. HTTPS certificate issues (use HTTP endpoint instead)
4. Note was attached to a previous error
5. Location-tracker service is not running or crashed

**Debug steps**:
```bash
# Check if webhook endpoint is accessible (use HTTP endpoint)
curl -X POST http://your-ec2-dns:8081/api/twilio/sms \
  -d "Body=test" -d "From=+15551234567" -d "MessageSid=SM123"

# Check service logs for webhook receipt
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@your-ec2-dns \
  'docker logs location-tracker | grep "Received SMS"'

# Verify service is running (not restarting)
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@your-ec2-dns \
  'docker ps --filter name=location-tracker'
```

### Issue: Twilio webhook returns 502 (Certificate Invalid)

**Error:** `Certificate Invalid - Could not find path to certificate` (Error Code: 11237)

**Solution**: This means you're using the HTTPS endpoint with a self-signed certificate. **Use the HTTP endpoint instead:**
```
http://your-ec2-dns:8081/api/twilio/sms
```

See [DEPLOYMENT_TROUBLESHOOTING.md](DEPLOYMENT_TROUBLESHOOTING.md) for detailed explanation.

### Issue: Twilio webhook times out

**Solution**: Ensure your EC2 security group allows inbound traffic on port 8081 from all IPs (0.0.0.0/0) or specifically from Twilio's IP ranges.

### Issue: Service crashes after deployment

**Symptoms:** Location-tracker shows `Restarting` status, logs show `TRACKER_PASSWORD environment variable must be set!`

**Solution**: The deployment script now validates this automatically. If deploying manually, ensure TRACKER_PASSWORD is set. See [DEPLOYMENT_TROUBLESHOOTING.md](DEPLOYMENT_TROUBLESHOOTING.md) for complete recovery steps.

### Issue: Wrong format in UI

**Solution**: User experience notes support plain text only. Special characters are HTML-escaped automatically. For formatting, use simple punctuation.

### Additional Help

For more deployment and troubleshooting information, see:
- [DEPLOYMENT_TROUBLESHOOTING.md](DEPLOYMENT_TROUBLESHOOTING.md) - Complete deployment troubleshooting guide
- [README.md](README.md) - Main project documentation

## Example Use Cases

1. **Customer Feedback**: "Payment failed after 3 attempts, very frustrating"
2. **Bug Reports**: "Search not working on mobile Safari"
3. **Performance Issues**: "Page load took over 30 seconds"
4. **Feature Requests**: "Need ability to bulk edit records"
5. **Context Notes**: "Error occurred during Black Friday sale peak"
