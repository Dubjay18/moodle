# Secure DigitalOcean Deployment with Secrets Management

## 🔐 Security Overview

The `app.yaml` file now uses variable references instead of hardcoded secrets. You'll need to set these secrets through DigitalOcean's interface or CLI.

## 📋 Required Secrets

### Via DigitalOcean Dashboard

1. **Go to your App in DigitalOcean**:
   ```
   https://cloud.digitalocean.com/apps
   ```

2. **Navigate to Settings → Environment Variables**

3. **Add these secrets**:

   ```bash
   # Supabase Configuration
   SUPABASE_URL=https://your-project-id.supabase.co
   SUPABASE_ANON_KEY=your_supabase_anon_key_here
   SUPABASE_JWKS_URL=https://your-project-id.supabase.co/auth/v1/keys
   SUPABASE_JWT_ISSUER=https://your-project-id.supabase.co/auth/v1
   
   # External APIs
   TMDB_API_KEY=your_tmdb_api_key_here
   GEMINI_API_KEY=your_gemini_api_key_here
   
   # Frontend URL (update for production)
   CLIENT_URL=https://your-frontend-domain.com
   ```

### Via doctl CLI

```bash
# Set secrets via command line
doctl apps update YOUR_APP_ID --spec - <<EOF
name: moodle-api
services:
- name: api
  envs:
  - key: SUPABASE_URL
    value: "https://your-project-id.supabase.co"
    type: SECRET
  - key: SUPABASE_ANON_KEY
    value: "your_supabase_anon_key_here"
    type: SECRET
  - key: SUPABASE_JWKS_URL
    value: "https://your-project-id.supabase.co/auth/v1/keys"
    type: SECRET
  - key: SUPABASE_JWT_ISSUER
    value: "https://your-project-id.supabase.co/auth/v1"
    type: SECRET
  - key: TMDB_API_KEY
    value: "your_tmdb_api_key_here"
    type: SECRET
  - key: GEMINI_API_KEY
    value: "your_gemini_api_key_here"
    type: SECRET
  - key: CLIENT_URL
    value: "https://your-frontend-domain.com"
    type: GENERAL
EOF
```

## 🚀 Deployment Steps

### 1. Create App (First Time)

```bash
# Install doctl
curl -sL https://github.com/digitalocean/doctl/releases/download/v1.100.0/doctl-1.100.0-linux-amd64.tar.gz | tar -xzv
sudo mv doctl /usr/local/bin

# Authenticate
doctl auth init

# Create app from spec
doctl apps create --spec app.yaml
```

### 2. Set Secrets in Dashboard

1. Go to DigitalOcean Apps dashboard
2. Find your `moodle-api` app
3. Go to **Settings** → **Environment Variables**
4. Add each secret as **Encrypted** type
5. Redeploy the app

### 3. Alternative: Use Environment File

Create a secure environment file for doctl:

```bash
# Create secrets.env (never commit this!)
cat > secrets.env << 'EOF'
SUPABASE_URL=https://your-project-id.supabase.co
SUPABASE_ANON_KEY=your_supabase_anon_key_here
SUPABASE_JWKS_URL=https://your-project-id.supabase.co/auth/v1/keys
SUPABASE_JWT_ISSUER=https://your-project-id.supabase.co/auth/v1
TMDB_API_KEY=your_tmdb_api_key_here
GEMINI_API_KEY=your_gemini_api_key_here
CLIENT_URL=https://your-frontend-domain.com
EOF

# Add to .gitignore
echo "secrets.env" >> .gitignore
```

### 4. Deploy Script

```bash
#!/bin/bash
# deploy.sh

set -e

APP_NAME="moodle-api"

# Check if app exists
if doctl apps list | grep -q "$APP_NAME"; then
    echo "Updating existing app..."
    APP_ID=$(doctl apps list --format ID,Name --no-header | grep "$APP_NAME" | awk '{print $1}')
    doctl apps update "$APP_ID" --spec app.yaml
else
    echo "Creating new app..."
    doctl apps create --spec app.yaml
fi

echo "Deployment initiated. Check status with:"
echo "doctl apps list"
```

## 🔄 CI/CD with GitHub Actions

Update `.github/workflows/deploy.yml`:

```yaml
name: Deploy to DigitalOcean

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Install doctl
      uses: digitalocean/action-doctl@v2
      with:
        token: ${{ secrets.DIGITALOCEAN_ACCESS_TOKEN }}
    
    - name: Update app secrets
      run: |
        # Set secrets via GitHub Actions secrets
        doctl apps update ${{ secrets.APP_ID }} --spec - <<EOF
        name: moodle-api
        services:
        - name: api
          envs:
          - key: SUPABASE_URL
            value: "${{ secrets.SUPABASE_URL }}"
            type: SECRET
          - key: SUPABASE_ANON_KEY
            value: "${{ secrets.SUPABASE_ANON_KEY }}"
            type: SECRET
          - key: TMDB_API_KEY
            value: "${{ secrets.TMDB_API_KEY }}"
            type: SECRET
          - key: GEMINI_API_KEY
            value: "${{ secrets.GEMINI_API_KEY }}"
            type: SECRET
        EOF
    
    - name: Deploy to App Platform
      uses: digitalocean/app_action@v1.1.5
      with:
        app_name: moodle-api
        token: ${{ secrets.DIGITALOCEAN_ACCESS_TOKEN }}
```

## 🛡️ Security Best Practices

### ✅ What we fixed:
- Removed hardcoded secrets from `app.yaml`
- Using DO's encrypted environment variables
- Secrets are now stored securely on DigitalOcean

### 🔐 Additional Security:
```bash
# 1. Rotate secrets regularly
# 2. Use different keys for dev/staging/prod
# 3. Monitor secret access in DO dashboard
# 4. Never commit secrets to Git

# Add to .gitignore
echo "*.env" >> .gitignore
echo "secrets.*" >> .gitignore
echo ".env.local" >> .gitignore
```

### 🚨 Emergency Secret Rotation:
```bash
# If secrets are compromised:
# 1. Generate new API keys
# 2. Update in DigitalOcean dashboard
# 3. Redeploy app
# 4. Revoke old keys

doctl apps update YOUR_APP_ID --spec app.yaml
```

## 📱 Mobile/Frontend Configuration

Your frontend should now connect to the deployed API:

```javascript
// Update your frontend config
const API_BASE_URL = 'https://your-app-name-xxxxx.ondigitalocean.app';

// The deployed API will handle CORS and auth properly
```

## ✅ Verification

After deployment, verify secrets are working:

```bash
# Check app status
doctl apps list

# View logs
doctl apps logs YOUR_APP_ID --follow

# Test endpoints
curl https://your-app-xxxxx.ondigitalocean.app/healthz
```

Your secrets are now properly secured! 🔐
