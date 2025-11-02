# Deployment Troubleshooting Guide

## Common Issues and Solutions

### Issue: DynamoDB Not Saving Error Logs After Deployment

**Symptoms:**
- Error logs stop appearing in DynamoDB after deployment
- Last records in database are from previous deployment
- Container shows `❌ TRACKER_PASSWORD environment variable must be set!` errors
- Container is in `Restarting` status

**Root Cause:**
The location-tracker service requires the `TRACKER_PASSWORD` environment variable to start. If this variable is not passed during deployment, the service will crash-loop and cannot receive or save error logs.

**Solution:**

1. **Always use the deploy-to-ec2.sh script** which properly loads environment variables from `.env.ec2`

2. **If manual deployment is needed**, ensure environment variables are passed:
   ```bash
   # Load environment variables first
   source .env.ec2

   # Then deploy with variables
   ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@$EC2_HOST \
     TRACKER_PASSWORD="${TRACKER_PASSWORD}" \
     GOOGLE_MAPS_API_KEY="${GOOGLE_MAPS_API_KEY}" \
     bash << 'EOF'
       docker run -d \
         --name location-tracker \
         -e TRACKER_PASSWORD="${TRACKER_PASSWORD}" \
         -e GOOGLE_MAPS_API_KEY="${GOOGLE_MAPS_API_KEY}" \
         ...other flags...
   EOF
   ```

3. **Verify container is running**:
   ```bash
   ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@$EC2_HOST 'docker ps --filter name=location-tracker'
   ```

   Should show `Up X seconds` not `Restarting`

4. **Check logs for successful startup**:
   ```bash
   ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@$EC2_HOST 'docker logs location-tracker | grep "✅"'
   ```

   Should show:
   ```
   ✅ Location tracker starting...
   💾 DynamoDB persistence enabled
   ✅ Loaded X error logs from DynamoDB into memory
   ```

### Issue: Error Generator Can't Reach Location Tracker

**Symptoms:**
- Error generator logs show: `no such host` or `dial tcp: lookup location-tracker`
- Errors not appearing in location-tracker

**Root Cause:**
After redeploying location-tracker, the Docker network DNS may not update immediately for running containers.

**Solution:**

Restart error-generator after redeploying location-tracker:
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@$EC2_HOST 'docker restart error-generator'
```

Or use the full deployment script which restarts all services in order.

### Issue: Twilio Webhook Returns 502 (Certificate Invalid)

**Symptoms:**
- Twilio dashboard shows: `Certificate Invalid - Could not find path to certificate`
- Error code: 11237
- HTTP response: 502

**Root Cause:**
Twilio requires a valid SSL certificate from a trusted Certificate Authority. Self-signed certificates are not accepted.

**Solution:**

Use the HTTP endpoint for Twilio webhooks:

1. **Ensure HTTP port is exposed** (port 8081):
   ```bash
   docker run -d \
     --name location-tracker \
     -p 8081:8080 \
     -p 8082:8443 \
     ...other flags...
   ```

2. **Configure Twilio webhook with HTTP URL**:
   ```
   http://ec2-54-226-246-133.compute-1.amazonaws.com:8081/api/twilio/sms
   ```

3. **Ensure security group allows inbound on port 8081**:
   ```bash
   aws ec2 authorize-security-group-ingress \
     --group-id sg-0b854584c1f195ecf \
     --protocol tcp \
     --port 8081 \
     --cidr 0.0.0.0/0 \
     --region us-east-1
   ```

**For Production:**
Consider using AWS Certificate Manager (ACM) with an Application Load Balancer for valid SSL certificates.

## Validation Checklist

After any deployment, verify:

- [ ] All containers are running (not restarting):
  ```bash
  docker ps --filter name=slogan-server --filter name=error-generator --filter name=location-tracker
  ```

- [ ] Location-tracker loaded DynamoDB data:
  ```bash
  docker logs location-tracker | grep "Loaded.*from DynamoDB"
  ```

- [ ] Error-generator is sending errors:
  ```bash
  docker logs error-generator | grep "Sent error log to location tracker"
  ```

- [ ] Errors are being saved to DynamoDB:
  ```bash
  docker logs location-tracker | grep "Error log saved to DynamoDB"
  ```

- [ ] Services are on the same Docker network:
  ```bash
  docker network inspect ec2-test-network
  ```

## Emergency Recovery

If services are broken and you need to restore quickly:

```bash
# 1. Run the full deployment script
./deploy-to-ec2.sh

# 2. Verify all services started
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@$EC2_HOST << 'EOF'
  docker ps
  docker logs --tail 10 slogan-server
  docker logs --tail 10 location-tracker
  docker logs --tail 10 error-generator
EOF
```

## Preventing Issues

1. **Always use .env.ec2**
   - Never commit `.env.ec2` to git (it's in .gitignore)
   - Always load it before deployment: `source .env.ec2`
   - Verify required variables are set: `echo $TRACKER_PASSWORD`

2. **Use the deployment script**
   - `./deploy-to-ec2.sh` handles all environment variables correctly
   - Includes proper wait times between service starts
   - Validates services after deployment

3. **Monitor logs after deployment**
   - Check for crash-loops immediately
   - Verify DynamoDB connectivity
   - Confirm inter-service communication

4. **Test end-to-end flow**
   - Wait for next error generation cycle (~255 seconds)
   - Verify error appears in DynamoDB
   - Check error displays in UI

## Support

If issues persist:
1. Check logs: `docker logs <container-name>`
2. Verify environment variables: `docker inspect <container-name> | grep -A 20 Env`
3. Check network connectivity: `docker exec error-generator ping location-tracker`
4. Review security groups: `aws ec2 describe-security-groups --group-ids sg-0b854584c1f195ecf`
