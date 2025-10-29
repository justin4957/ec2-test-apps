# Giphy API Configuration Guide

This guide explains how to configure the error-generator to use real GIFs from Giphy instead of placeholder URLs.

## Getting a Giphy API Key

1. **Sign up for Giphy Developers**
   - Go to https://developers.giphy.com/
   - Click "Create an App"
   - Select "API" (not SDK)
   - Choose the type of app (select "App" for general use)
   - Fill in the required information
   - Accept the terms and create your app

2. **Get Your API Key**
   - Once created, you'll see your API key
   - Copy the key (it looks like: `AbCdEfGhIjKlMnOpQrStUvWxYz123456`)

## Configuration Methods

### Method 1: Using .env.ec2 File (Recommended)

1. **Create the environment file**:
   ```bash
   cd ec2-test-apps
   cp .env.ec2.example .env.ec2
   ```

2. **Edit .env.ec2** and add your API key:
   ```bash
   GIPHY_API_KEY=your_actual_api_key_here
   ERROR_INTERVAL_SECONDS=60
   ```

3. **Deploy to EC2**:
   ```bash
   ./deploy-to-ec2.sh
   ```

The script will automatically:
- Load environment variables from `.env.ec2`
- Pass the API key to the error-generator container
- Start fetching real GIFs from Giphy

### Method 2: Export Environment Variable

```bash
export GIPHY_API_KEY=your_actual_api_key_here
./deploy-to-ec2.sh
```

### Method 3: Inline with Deployment

```bash
GIPHY_API_KEY=your_actual_api_key_here ./deploy-to-ec2.sh
```

### Method 4: Manual Container Update (Already Running)

If containers are already deployed and you just want to add the API key:

```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com

# Stop and remove existing error-generator
docker stop error-generator
docker rm error-generator

# Restart with API key
docker run -d \
    --name error-generator \
    --restart unless-stopped \
    --network ec2-test-network \
    -e SLOGAN_SERVER_URL=http://slogan-server:8080 \
    -e ERROR_INTERVAL_SECONDS=60 \
    -e GIPHY_API_KEY=your_actual_api_key_here \
    310829530225.dkr.ecr.us-east-1.amazonaws.com/error-generator:latest
```

## Verifying Giphy Integration

### Check Logs
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
    'docker logs --tail 20 error-generator'
```

**With API Key** (look for):
```
Loaded 25 GIF URLs from Giphy (search term: fail)
With GIF: https://giphy.com/gifs/fail-black-and-white-bob-dylan-li0dswKqIZNpm
```

**Without API Key** (you'll see):
```
GIPHY_API_KEY not set, using placeholder GIFs
With GIF: https://giphy.com/gifs/error-placeholder-1
```

## How It Works

1. **GIF Batch Loading**: The error-generator loads 25 GIFs at startup from Giphy API
2. **Search Terms**: Randomly chooses from: "error", "fail", "glitch", "broken", "oops"
3. **Rate Limiting**: With 60-second intervals and 25 GIFs per batch, you get 25 minutes before needing to reload
4. **Auto-Reload**: When the cache is exhausted, it automatically fetches a new batch with a different search term

## Rate Limits

**Giphy Free Tier**:
- 42,000 requests per day
- No rate limit per second specified

**This App's Usage**:
- Loads 25 GIFs every ~25 minutes (at 60-second intervals)
- Approximately 58 API calls per day
- Well within free tier limits

## Security Best Practices

### ✅ DO:
- Store API key in `.env.ec2` (ignored by git)
- Use environment variables
- Keep API keys out of source code
- Rotate keys periodically

### ❌ DON'T:
- Commit `.env.ec2` to git (already in .gitignore)
- Hardcode API keys in scripts
- Share API keys publicly
- Use production keys in public repos

## Troubleshooting

### No Real GIFs Loading

**Check 1**: Verify API key is set
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
    'docker inspect error-generator | grep GIPHY_API_KEY'
```

**Check 2**: Test API key directly
```bash
curl "https://api.giphy.com/v1/gifs/search?api_key=YOUR_KEY&q=error&limit=5"
```

**Check 3**: View container logs
```bash
ssh -i ~/.ssh/ec2-test-apps-key.pem ec2-user@ec2-54-226-246-133.compute-1.amazonaws.com \
    'docker logs error-generator 2>&1 | grep -i giphy'
```

### API Key Not Working

**Error: 401 Unauthorized**
- API key is invalid or expired
- Get a new key from https://developers.giphy.com/

**Error: 429 Too Many Requests**
- You've exceeded rate limits
- Wait or upgrade your Giphy plan

**Error: 403 Forbidden**
- API key doesn't have correct permissions
- Check your Giphy dashboard settings

## Configuration Changes

### Change Error Interval

In `.env.ec2`:
```bash
ERROR_INTERVAL_SECONDS=30  # Send errors every 30 seconds
```

Then redeploy:
```bash
./deploy-to-ec2.sh
```

### Remove API Key

1. Remove from `.env.ec2` or set to empty:
   ```bash
   GIPHY_API_KEY=
   ```

2. Redeploy:
   ```bash
   ./deploy-to-ec2.sh
   ```

The app will fallback to placeholder GIF URLs.

## Current Status

✅ **Giphy API Key Configured**: Yes
✅ **Real GIFs Loading**: Yes
✅ **API Key Secure**: Yes (in .env.ec2, not in git)
✅ **Deployment Script Updated**: Yes (supports .env.ec2)

## Next Steps

1. Monitor logs to see real GIF URLs
2. Adjust ERROR_INTERVAL_SECONDS if desired
3. Enjoy your absurdly paid-for error log advertising machine! 🚬

---

**Need Help?**
- Check logs: `docker logs error-generator`
- Test endpoint: `curl http://ec2-54-226-246-133.compute-1.amazonaws.com:8080/error-log`
- Giphy API Docs: https://developers.giphy.com/docs/api/
