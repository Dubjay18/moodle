# DigitalOcean Deployment Guide

## Quick Start

### Option 1: App Platform (Recommended)

1. **Push to GitHub**:
   ```bash
   git add .
   git commit -m "Add deployment configs"
   git push origin main
   ```

2. **Install doctl CLI**:
   ```bash
   # macOS
   brew install doctl
   
   # Linux
   curl -sL https://github.com/digitalocean/doctl/releases/download/v1.100.0/doctl-1.100.0-linux-amd64.tar.gz | tar -xzv
   sudo mv doctl /usr/local/bin
   ```

3. **Authenticate with DigitalOcean**:
   ```bash
   doctl auth init
   # Enter your API token from https://cloud.digitalocean.com/account/api/tokens
   ```

4. **Deploy**:
   ```bash
   # Edit app.yaml first - update GitHub repo and API keys
   doctl apps create --spec app.yaml
   ```

### Option 2: Manual via Web Interface

1. Go to [DigitalOcean App Platform](https://cloud.digitalocean.com/apps)
2. Click "Create App" → "GitHub" → select this repo
3. Configure:
   - Service Type: Web Service
   - Build Command: `go build -o main ./cmd/api`
   - Run Command: `./main`
   - Port: 8080
4. Add environment variables (see .env.prod template)
5. Deploy

## Environment Setup

### Required API Keys

1. **Supabase** (https://supabase.com/dashboard):
   - Create project
   - Go to Settings → API
   - Copy JWT Secret and project URL

2. **TMDb** (https://www.themoviedb.org/settings/api):
   - Request API key
   - Copy API key

3. **Google Gemini** (https://aistudio.google.com/app/apikey):
   - Create API key
   - Copy API key

### Update Configuration

Edit `app.yaml` and replace:
- `Dubjay18/moodle` with your GitHub username/repo
- `your-project.supabase.co` with your Supabase project URL
- `your-tmdb-key` with your TMDb API key
- `your-gemini-key` with your Gemini API key

## Commands

```bash
# Check app status
doctl apps list

# View logs
doctl apps logs YOUR-APP-ID --follow

# Update app
doctl apps update YOUR-APP-ID --spec app.yaml

# Delete app
doctl apps delete YOUR-APP-ID
```

## Costs

- Basic App Platform: $5-12/month
- Managed Postgres (1GB): $15/month
- **Total: ~$20-27/month**

## Troubleshooting

- Check logs: `doctl apps logs YOUR-APP-ID`
- Verify environment variables in DO dashboard
- Ensure migrations run successfully
- Check database connectivity
