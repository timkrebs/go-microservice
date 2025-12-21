#!/bin/bash
set -e

echo "==================================="
echo "Cloudflare Tunnel Setup Script"
echo "==================================="
echo ""

# Check if cloudflared is installed
if ! command -v cloudflared &> /dev/null; then
    echo "cloudflared is not installed"
    echo ""
    echo "Install it with:"
    echo "  macOS:   brew install cloudflare/cloudflare/cloudflared"
    echo "  Linux:   wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb && sudo dpkg -i cloudflared-linux-amd64.deb"
    exit 1
fi

echo "✓ cloudflared is installed"
echo ""

# Step 1: Login
echo "Step 1: Authenticate with Cloudflare"
echo "----------------------------------------"
echo "This will open your browser. Select the 'qnt9.com' domain."
read -p "Press Enter to continue..."
cloudflared tunnel login
echo ""

# Step 2: Create tunnel
echo "Step 2: Create the tunnel"
echo "----------------------------------------"
TUNNEL_NAME="image-processor-tunnel"
echo "Creating tunnel: $TUNNEL_NAME"

if cloudflared tunnel info $TUNNEL_NAME &> /dev/null; then
    echo "Tunnel '$TUNNEL_NAME' already exists"
    read -p "Do you want to use the existing tunnel? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Please delete the existing tunnel first with: cloudflared tunnel delete $TUNNEL_NAME"
        exit 1
    fi
else
    cloudflared tunnel create $TUNNEL_NAME
fi

# Get tunnel ID
TUNNEL_ID=$(cloudflared tunnel info $TUNNEL_NAME 2>/dev/null | grep -E "^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}" | awk '{print $1}')

# If that didn't work, try to extract from list
if [ -z "$TUNNEL_ID" ]; then
    TUNNEL_ID=$(cloudflared tunnel list 2>/dev/null | grep "$TUNNEL_NAME" | awk '{print $1}')
fi

echo ""
echo "✓ Tunnel created/exists with ID: $TUNNEL_ID"
echo ""

if [ -z "$TUNNEL_ID" ]; then
    echo "Failed to extract tunnel ID"
    echo "Please run: cloudflared tunnel list"
    echo "And manually note your tunnel ID"
    exit 1
fi

# Step 3: Create Kubernetes secret
echo "Step 3: Create Kubernetes secret with credentials"
echo "----------------------------------------"
CREDS_FILE="$HOME/.cloudflared/$TUNNEL_ID.json"

if [ ! -f "$CREDS_FILE" ]; then
    echo "Credentials file not found at: $CREDS_FILE"
    exit 1
fi

echo "Creating/updating secret in Kubernetes..."
kubectl create secret generic cloudflared-credentials \
  --from-file=credentials.json=$CREDS_FILE \
  -n image-processor \
  --dry-run=client -o yaml | kubectl apply -f -

echo "✓ Secret created/updated"
echo ""

# Step 4: Configure DNS
echo "Step 4: Configure DNS"
echo "----------------------------------------"
echo "Setting up DNS routes..."
cloudflared tunnel route dns $TUNNEL_NAME app.qnt9.com
cloudflared tunnel route dns $TUNNEL_NAME api.qnt9.com
echo "✓ DNS configured"
echo ""

# Step 5: Update ConfigMap
echo "Step 5: Update ConfigMap with tunnel ID"
echo "----------------------------------------"
CONFIGMAP_FILE="deployments/kubernetes/base/cloudflared/configmap.yaml"

if [ -f "$CONFIGMAP_FILE" ]; then
    # Use sed to replace the tunnel line
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        sed -i '' "s/tunnel: image-processor-tunnel/tunnel: $TUNNEL_ID/" "$CONFIGMAP_FILE"
    else
        # Linux
        sed -i "s/tunnel: image-processor-tunnel/tunnel: $TUNNEL_ID/" "$CONFIGMAP_FILE"
    fi
    echo "✓ ConfigMap updated with tunnel ID"
else
    echo "ConfigMap file not found at: $CONFIGMAP_FILE"
fi
echo ""

# Step 6: Deploy
echo "Step 6: Deploy to Kubernetes"
echo "----------------------------------------"
echo "Committing changes..."
git add "$CONFIGMAP_FILE"
git commit -m "feat: configure tunnel ID for cloudflared" || echo "No changes to commit"
echo ""

echo "Pushing to repository..."
git push
echo ""

echo "Syncing ArgoCD..."
kubectl patch application image-processor-dev -n argocd --type merge -p '{"operation":{"sync":{"revision":"HEAD"}}}'
echo ""

echo "==================================="
echo "Setup Complete!"
echo "==================================="
echo ""
echo "Waiting for pods to start..."
sleep 10

kubectl get pods -n image-processor -l app=cloudflared

echo ""
echo "Your application will be available at:"
echo "Frontend: https://app.qnt9.com"
echo "API:      https://api.qnt9.com/api/v1/health"
echo ""
echo "DNS propagation may take 1-2 minutes."
echo ""
echo "To check tunnel status:"
echo "  cloudflared tunnel info $TUNNEL_NAME"
echo ""
echo "To check logs:"
echo "  kubectl logs -n image-processor -l app=cloudflared -f"
