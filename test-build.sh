#!/bin/bash

set -e

echo "🔨 Testing Go builds..."

# Linux x86_64
echo "Building Linux x86_64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o vx-linux-x86_64 .

# Linux ARM64
echo "Building Linux ARM64..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o vx-linux-aarch64 .

# macOS x86_64
echo "Building macOS x86_64..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o vx-macos-x86_64 .

# macOS ARM64
echo "Building macOS ARM64..."
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -v -trimpath -ldflags="-s -w" -o vx-macos-aarch64 .

echo "✅ All builds successful!"
ls -lh vx-*

# Test one of the binaries
echo "Testing binary..."
./vx-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/aarch64/;s/arm64/aarch64/') --version
