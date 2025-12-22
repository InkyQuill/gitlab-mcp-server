#!/bin/bash

# Setup script for macOS/Linux
# Installs prerequisites and optionally runs the MCP installer

set -e

GO_VERSION_MIN="1.23"
BINARY_NAME="gitlab-mcp-server"
INSTALLER_BINARY="bin/install"

echo "=== GitLab MCP Server Setup ==="
echo ""

# Check for Go
echo "Checking for Go..."
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed."
    echo "Please install Go ${GO_VERSION_MIN} or later from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "Go found: $(go version)"

# Check Go version (simple check - assumes version format like 1.23.1)
MAJOR_MINOR=$(echo "$GO_VERSION" | cut -d. -f1,2)
REQUIRED_MAJOR_MINOR=$(echo "$GO_VERSION_MIN" | cut -d. -f1,2)

if [ "$(printf '%s\n' "$REQUIRED_MAJOR_MINOR" "$MAJOR_MINOR" | sort -V | head -n1)" != "$REQUIRED_MAJOR_MINOR" ]; then
    echo "Error: Go version must be ${GO_VERSION_MIN} or later (found ${GO_VERSION})"
    exit 1
fi

# Download dependencies
echo ""
echo "Downloading dependencies..."
go mod download

echo ""
echo "Prerequisites installed successfully!"
echo ""

# Ask if user wants to run installer
read -p "Do you want to configure MCP servers now? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Building installer..."
    mkdir -p bin
    go build -o "$INSTALLER_BINARY" ./cmd/install
    
    if [ -f "$INSTALLER_BINARY" ]; then
        echo "Running installer..."
        "$INSTALLER_BINARY"
    else
        echo "Error: Failed to build installer"
        exit 1
    fi
else
    echo "You can run the installer later with: make install-mcp"
fi

