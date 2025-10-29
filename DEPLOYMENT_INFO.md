# EC2 Test Apps - Deployment Information

## Deployment Summary

**Status**: ✅ Successfully Deployed
**Date**: October 28, 2025
**Region**: us-east-1

---

## EC2 Instance Details

- **Instance ID**: `i-04bd2369c252bee39`
- **Instance Type**: t2.micro
- **Public DNS**: `ec2-54-226-246-133.compute-1.amazonaws.com`
- **Public IP**: 54.226.246.133
- **SSH Key**: `~/.ssh/ec2-test-apps-key.pem`
- **Security Group**: `sg-0b854584c1f195ecf` (docker-app-sg)
- **IAM Role**: EC2-ECR-Access (for pulling from ECR)

### SSH Access
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com
```

---

## ECR Repositories

### Slogan Server
- **URI**: `310829530225.dkr.ecr.us-east-1.amazonaws.com/slogan-server:latest`
- **Digest**: sha256:147fed4ac66696cda08e59b65c3857ecdd2a6a34a52063c58ca15efd6924cd45

### Error Generator
- **URI**: `310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest`
- **Digest**: sha256:162918b704e2bdbf733c1a5c4f358d434fd40ae0da2ca2fc86859f4bed131461

---

## Running Containers

Both containers are running on a shared Docker network (`ec2-test-network`) for inter-container communication:

### Slogan Server
- **Container Name**: `slogan-server`
- **Port**: 8080 (exposed publicly)
- **Status**: Running
- **Restart Policy**: unless-stopped
- **AI Integration**: OpenAI GPT-4o-mini ✅
- **Environment**:
  - `OPENAI_API_KEY=***configured***` ✅
- **Slogan Generation**: AI-powered with fallback to 115 pre-generated slogans

### Error Generator
- **Container Name**: `error-generator`
- **Environment**:
  - `SLOGAN_SERVER_URL=http://slogan-server:8080`
  - `ERROR_INTERVAL_SECONDS=60`
  - `GIPHY_API_KEY=***configured***` ✅
- **Status**: Running
- **Restart Policy**: unless-stopped
- **GIF Source**: Real GIFs from Giphy API

---

## Testing the Deployment

### Test the Slogan Server Endpoint
```bash
curl -X POST http://ec2-54-226-246-133.compute-1.amazonaws.com:8080/error-log \
  -H "Content-Type: application/json" \
  -d '{"message": "Test error", "gif_url": "https://giphy.com/test"}'
```

**Example Response**:
```json
{"emoji":"🚬","slogan":"Security misconfiguration: Artistic freedom"}
```

### View Container Logs

**Slogan Server Logs**:
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
  'docker logs -f slogan-server'
```

**Error Generator Logs**:
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
  'docker logs -f error-generator'
```

**View Latest Error Logs**:
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
  'docker logs --tail 30 error-generator'
```

---

## Application Behavior

The system is working as designed:

1. **Error Generator** sends a simulated error log every 60 seconds
2. Each error includes a random message and a **real GIF URL from Giphy**
3. **Slogan Server** uses **OpenAI GPT-4o-mini** to generate a custom sardonic slogan:
   - Analyzes the error message
   - Extracts context from the GIF URL
   - Generates a unique, darkly humorous advertising slogan
   - Falls back to 115 pre-generated slogans if OpenAI fails
4. Responds with:
   - A cigarette emoji (🚬)
   - An AI-generated sardonic slogan (or fallback)
5. **Error Generator** logs the response in a formatted display

**Sample Output** (with OpenAI + Giphy):
```
=== ERROR LOG ===
Error: DeadlockDetected: Thread pool exhausted
GIF: https://giphy.com/gifs/girl-sad-white-ShPv5tt0EM396
Response: 🚬 DeadlockDetected: Embrace the bliss of perpetual stagnation!
================
```

**Sample AI-Generated Slogans:**
- "FileNotFoundException: Embrace the mystery of absence!"
- "SegmentationFault: When your code self-destructs, embrace the chaos!"
- "DeadlockDetected: Embrace the bliss of perpetual stagnation!"

### API Configuration

The deployment is configured with both Giphy and OpenAI API keys stored in `.env.ec2` (git-ignored).

**Giphy API**: Fetches real, contextual GIFs for each error message
**OpenAI API**: Generates unique, sardonic slogans using GPT-4o-mini

To update or change the API keys, see [GIPHY_API_SETUP.md](GIPHY_API_SETUP.md) for detailed instructions.

---

## Security Group Configuration

### Inbound Rules
- **SSH (22)**: From your current IP (172.56.114.251/32)
- **HTTP (8080)**: From anywhere (0.0.0.0/0) - for testing the slogan server
- **HTTP (80)**: From anywhere (0.0.0.0/0) - not currently used

---

## Management Commands

### Restart Containers
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
  'docker restart slogan-server error-generator'
```

### Stop Containers
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
  'docker stop slogan-server error-generator'
```

### Check Container Status
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
  'docker ps'
```

### Update Deployment
To deploy new versions, rebuild and push the images, then run:
```bash
cd /Users/coolbeans/Development/dev/ec2-test-apps
./deploy-to-ec2.sh
```

---

## Cost Considerations

**Running Costs** (approximate):
- **EC2 t2.micro**: ~$0.0116/hour (~$8.50/month)
- **ECR Storage**: $0.10/GB/month (minimal for these small images)
- **Data Transfer**: Varies based on usage

**To Stop Costs**:
```bash
# Stop the instance (can be restarted later)
aws ec2 stop-instances --instance-ids i-04bd2369c252bee39 --region us-east-1

# Or terminate completely (cannot be recovered)
aws ec2 terminate-instances --instance-ids i-04bd2369c252bee39 --region us-east-1
```

---

## Troubleshooting

### Containers Not Communicating
If error-generator can't reach slogan-server, ensure they're on the same network:
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
  'docker network inspect ec2-test-network'
```

### Can't Access from Outside
Check security group allows port 8080:
```bash
aws ec2 describe-security-groups --group-ids sg-0b854584c1f195ecf --region us-east-1
```

### SSH Connection Issues
Update security group with your current IP:
```bash
MY_IP=$(curl -s https://checkip.amazonaws.com)
aws ec2 authorize-security-group-ingress \
  --group-id sg-0b854584c1f195ecf \
  --protocol tcp \
  --port 22 \
  --cidr ${MY_IP}/32 \
  --region us-east-1
```

---

## Success Metrics

✅ ECR repositories created
✅ Docker images built for linux/amd64
✅ Images pushed to ECR
✅ EC2 instance created with proper IAM role
✅ Containers deployed and running
✅ Inter-container networking configured
✅ Port 8080 publicly accessible
✅ Error generator successfully sending requests every 60 seconds
✅ Slogan server responding with emoji + random slogans

**The deployment is fully operational!** 🚬
