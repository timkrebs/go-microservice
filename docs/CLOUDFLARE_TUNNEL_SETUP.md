# Cloudflare Tunnel Setup Guide

This guide will help you expose your Image Processor application to the internet via Cloudflare Tunnel at **app.qnt9.com**.

## Prerequisites

- A Cloudflare account with the domain `qnt9.com` configured
- Access to your Kubernetes cluster
- `cloudflared` CLI installed on your local machine

## Step 1: Install cloudflared CLI (if not already installed)

### On macOS:
```bash
brew install cloudflare/cloudflare/cloudflared
```

### On Linux:
```bash
wget -q https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
sudo dpkg -i cloudflared-linux-amd64.deb
```

## Step 2: Authenticate with Cloudflare

```bash
cloudflared tunnel login
```

This will open a browser window. Select your `qnt9.com` domain.

## Step 3: Create the Tunnel

```bash
cloudflared tunnel create image-processor-tunnel
```

This creates a tunnel and generates a credentials file. Note the **Tunnel ID** from the output.

## Step 4: Locate the Credentials File

The credentials file is typically located at:
```
~/.cloudflared/<TUNNEL_ID>.json
```

## Step 5: Create Kubernetes Secret with Credentials

```bash
# Replace <TUNNEL_ID> with your actual tunnel ID
kubectl create secret generic cloudflared-credentials \
  --from-file=credentials.json=$HOME/.cloudflared/<TUNNEL_ID>.json \
  -n image-processor
```

## Step 6: Configure DNS in Cloudflare

You have two options:

### Option A: Using cloudflared CLI (Recommended)
```bash
cloudflared tunnel route dns image-processor-tunnel app.qnt9.com
cloudflared tunnel route dns image-processor-tunnel api.qnt9.com
```

### Option B: Manual DNS Configuration
1. Log in to Cloudflare Dashboard
2. Go to DNS settings for `qnt9.com`
3. Add CNAME records:
   - **Name**: `app`
   - **Target**: `<TUNNEL_ID>.cfargotunnel.com`
   - **Proxy**: Enabled (orange cloud)
   
   - **Name**: `api`
   - **Target**: `<TUNNEL_ID>.cfargotunnel.com`
   - **Proxy**: Enabled (orange cloud)

## Step 7: Update ConfigMap with Tunnel ID

Edit the file `deployments/kubernetes/base/cloudflared/configmap.yaml` and replace:
```yaml
tunnel: image-processor-tunnel
```

with:
```yaml
tunnel: <YOUR_TUNNEL_ID>
```

## Step 8: Update Kustomization

The cloudflared resources need to be added to kustomization.yaml. This will be done automatically when you commit the changes.

## Step 9: Deploy to Kubernetes

```bash
cd /Users/timkrebs/Development/Go/go-microservice

# Commit the cloudflared configuration
git add deployments/kubernetes/base/cloudflared/
git add deployments/kubernetes/base/kustomization.yaml
git commit -m "feat: add Cloudflare Tunnel for public access"
git push

# ArgoCD will automatically deploy the changes
# Or manually sync:
kubectl patch application image-processor-dev -n argocd --type merge -p '{"operation":{"sync":{"revision":"HEAD"}}}'
```

## Step 10: Verify the Tunnel

Check if cloudflared is running:
```bash
kubectl get pods -n image-processor -l app=cloudflared
kubectl logs -n image-processor -l app=cloudflared --tail=50
```

## Step 11: Test Access

Wait a minute for DNS propagation, then visit:
- **Frontend**: https://app.qnt9.com
- **API Health**: https://api.qnt9.com/api/v1/health

## Troubleshooting

### Check tunnel status:
```bash
cloudflared tunnel info image-processor-tunnel
```

### Check pod logs:
```bash
kubectl logs -n image-processor -l app=cloudflared -f
```

### Verify DNS:
```bash
dig app.qnt9.com
nslookup app.qnt9.com
```

### Common Issues:

1. **Tunnel not connecting**: Verify credentials secret exists
   ```bash
   kubectl get secret cloudflared-credentials -n image-processor
   ```

2. **DNS not resolving**: Wait 1-2 minutes for DNS propagation

3. **502 Bad Gateway**: Check that frontend/api services are running
   ```bash
   kubectl get svc -n image-processor
   kubectl get pods -n image-processor
   ```

## Security Notes

- Cloudflare Tunnel provides automatic DDoS protection
- All traffic is encrypted end-to-end
- No need to open firewall ports on your network
- Cloudflare's Web Application Firewall (WAF) can be enabled for additional protection

## Additional Configuration (Optional)

### Enable Cloudflare Access for authentication:
1. Go to Cloudflare Zero Trust dashboard
2. Configure Access policies for app.qnt9.com
3. Add authentication methods (Google, GitHub, email OTP, etc.)

### Rate Limiting:
Configure rate limiting rules in Cloudflare Dashboard → Security → Rate Limiting

### Firewall Rules:
Set up custom firewall rules in Cloudflare Dashboard → Security → WAF
