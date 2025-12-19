#!/bin/bash
set -e

echo "==================================================================="
echo "  Kubernetes Image Pull Secret Setup for GHCR"
echo "==================================================================="
echo ""
echo "You need a GitHub Personal Access Token (PAT) with 'read:packages' scope."
echo "Create one at: https://github.com/settings/tokens"
echo ""

read -p "Enter your GitHub username [timkrebs]: " GITHUB_USER
GITHUB_USER=${GITHUB_USER:-timkrebs}

echo ""
read -sp "Enter your GitHub Personal Access Token: " GITHUB_PAT
echo ""

if [ -z "$GITHUB_PAT" ]; then
  echo "Error: GitHub PAT cannot be empty"
  exit 1
fi

echo ""
echo "Creating image pull secrets in all namespaces..."
echo ""

for NS in image-processor image-processor-staging image-processor-prod; do
  echo "Namespace: $NS"
  
  # Create namespace if it doesn't exist
  kubectl create namespace $NS --dry-run=client -o yaml | kubectl apply -f -
  
  # Create docker registry secret
  kubectl create secret docker-registry ghcr-login-secret \
    --docker-server=ghcr.io \
    --docker-username=$GITHUB_USER \
    --docker-password=$GITHUB_PAT \
    --namespace=$NS \
    --dry-run=client -o yaml | kubectl apply -f -
  
  echo "   Secret created"
  echo ""
done

echo "==================================================================="
echo "All image pull secrets created successfully!"
echo ""
echo "Next steps:"
echo "1. Commit and push the deploy-gitops.yml workflow"
echo "2. Push any change to trigger CI/CD"
echo "3. Watch deployment: kubectl get pods -n image-processor -w"
echo "==================================================================="
