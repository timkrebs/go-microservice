#!/usr/bin/env bash
# Pre-commit hook script for local validation
# This runs automatically before each commit if installed

set -e

echo "Running pre-commit checks..."

# 1. Check formatting
echo "1️Checking code formatting..."
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo "Code not formatted. Files:"
    echo "$UNFORMATTED"
    echo "Run: go fmt ./..."
    exit 1
fi
echo "Code formatting OK"

# 2. Run go vet
echo "Running go vet..."
if ! go vet ./...; then
    echo "go vet failed"
    exit 1
fi
echo "go vet passed"

# 3. Run unit tests
echo "Running unit tests..."
if ! go test -short -race ./...; then
    echo "Unit tests failed"
    exit 1
fi
echo "Unit tests passed"

# 4. Check for large files
echo "Checking for large files..."
LARGE_FILES=$(find . -type f -size +5M -not -path "./.git/*" -not -path "./vendor/*" -not -path "./bin/*" 2>/dev/null || true)
if [ -n "$LARGE_FILES" ]; then
    echo "Large files detected (>5MB):"
    echo "$LARGE_FILES"
    exit 1
fi
echo " No large files"

# 5. Check for common secrets patterns
echo "Scanning for potential secrets..."
if git diff --cached | grep -iE '(password|secret|api_key|token).*=.*["'"'"'][^"'"'"']+["'"'"']' > /dev/null; then
    echo "Warning: Possible hardcoded secrets detected in staged files"
    echo "Please review your changes carefully"
    # Not failing, just warning
fi
echo " Secret scan complete"

echo "All pre-commit checks passed!"
