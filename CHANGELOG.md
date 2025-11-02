# Changelog

## 2025-11-02 - Twilio SMS Integration & DynamoDB Fix

### Added

#### Twilio SMS Integration
- **New Feature**: SMS-to-Error-Log integration via Twilio webhooks
  - Endpoint: `/api/twilio/sms` receives SMS messages from Twilio
  - Messages stored as pending "user experience notes"
  - Automatically attached to next error log
  - Displayed in UI with green highlighting and 💬 icon

- **Files Added**:
  - `TWILIO_INTEGRATION.md` - Complete setup and usage documentation
  - `test-twilio-integration.sh` - Automated testing script
  - `DEPLOYMENT_TROUBLESHOOTING.md` - Comprehensive troubleshooting guide
  - `CHANGELOG.md` - This file

#### Enhanced Deployment
- **Validation**: Pre-deployment checks for critical environment variables
  - Script now exits early if `TRACKER_PASSWORD` is missing
  - Clear error messages guide users to fix configuration

- **Post-Deployment Validation**:
  - Automatic container status checks
  - DynamoDB connectivity verification
  - Environment variable validation
  - Clear success/warning indicators

- **Dual Port Exposure**:
  - Port 8081: HTTP endpoint for Twilio webhooks
  - Port 8082: HTTPS endpoint for browser access (self-signed cert)

### Changed

#### Location Tracker
- **ErrorLog struct**: Added `UserExperienceNote` field (location-tracker/main.go:47)
- **Global state**: Added `pendingUserExperienceNote` with mutex for thread-safe access
- **Endpoint added**: `handleTwilioWebhook()` processes incoming SMS (location-tracker/main.go:476)
- **Error logging**: Modified to auto-attach pending notes (location-tracker/main.go:413-420)
- **UI display**: Added green-highlighted section for user notes (location-tracker/main.go:1160-1165)
- **Dockerfile**: Fixed to properly use go.mod for dependency management

#### Deployment Script (deploy-to-ec2.sh)
- Added environment variable validation before deployment
- Added post-deployment health checks
- Exposed both HTTP (8081) and HTTPS (8082) ports
- Enhanced output with Twilio webhook URL
- Added reference to troubleshooting documentation

#### Documentation
- **TWILIO_INTEGRATION.md**: Updated with HTTP endpoint instructions
- **TWILIO_INTEGRATION.md**: Added SSL certificate explanation
- **TWILIO_INTEGRATION.md**: Enhanced troubleshooting section
- **README.md**: Added Twilio SMS Integration feature section
- **README.md**: Updated architecture diagram

### Fixed

#### DynamoDB Storage Issue
**Problem**: After manual redeployment, error logs stopped being saved to DynamoDB

**Root Cause**:
- Location-tracker requires `TRACKER_PASSWORD` environment variable
- Manual deployment didn't pass environment variables
- Container entered crash-loop (restart cycle)
- Service couldn't receive or save error logs

**Resolution**:
1. Fixed immediate issue by properly redeploying with environment variables
2. Enhanced deployment script to validate critical variables before deployment
3. Added post-deployment validation to catch similar issues
4. Created comprehensive troubleshooting documentation

**Prevention**:
- Deployment script now validates `TRACKER_PASSWORD` exists before attempting deployment
- Post-deployment checks verify container is running (not restarting)
- Checks confirm DynamoDB data is loaded
- Validates environment variables are set in running containers

#### Network Connectivity
**Problem**: Error-generator couldn't reach location-tracker after redeployment

**Resolution**: Restart error-generator after location-tracker redeployment to refresh DNS

### Technical Details

#### Twilio Webhook Flow
```
User sends SMS → Twilio → POST /api/twilio/sms → Store message
                                                        ↓
Error Generator → POST /api/errorlogs → Attach note → DynamoDB → UI Display
```

#### Data Structure
```go
type ErrorLog struct {
    ID                  string
    Message             string
    GifURL              string
    Slogan              string
    SongTitle           string
    SongArtist          string
    SongURL             string
    UserExperienceNote  string    // NEW
    Timestamp           time.Time
}
```

#### Port Configuration
- **8080**: Internal HTTP (within Docker network)
- **8081**: External HTTP (for Twilio webhooks - no SSL required)
- **8082**: External HTTPS (for browser access - self-signed cert)
- **8443**: Internal HTTPS (within Docker network)

### Deployment Instructions

1. **Update Twilio Webhook** (if using):
   ```
   http://ec2-54-226-246-133.compute-1.amazonaws.com:8081/api/twilio/sms
   ```

2. **Deploy Updated Services**:
   ```bash
   ./deploy-to-ec2.sh
   ```

3. **Verify Deployment**:
   - Check validation output at end of deployment
   - All services should show ✅ running
   - DynamoDB should show ✅ data loaded
   - TRACKER_PASSWORD should show ✅ configured

### Breaking Changes

None. This is a backwards-compatible addition.

### Migration Notes

Existing deployments will continue to work. To enable Twilio SMS integration:

1. Configure Twilio webhook with HTTP endpoint (port 8081)
2. Ensure security group allows inbound traffic on port 8081
3. Redeploy using updated deployment script

### Known Issues

1. **Single pending note**: Only one SMS message stored at a time. If multiple SMS received before error, only latest is attached.
2. **No SMS authentication**: Webhook doesn't validate Twilio signature. Consider adding for production.
3. **In-memory storage**: Pending note lost if service restarts before attachment.

### Next Steps

See TWILIO_INTEGRATION.md "Future Enhancements" for planned improvements:
- Queue multiple pending notes
- Add Twilio signature validation
- Persist pending notes to DynamoDB
- SMS reply confirmations
- Note-to-specific-error-type routing
