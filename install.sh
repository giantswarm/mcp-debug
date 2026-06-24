#!/bin/bash
set -e

# This script downloads the latest mcp-debug binary by querying the GitHub API
# for the latest release tag and fetching the matching per-platform asset
# (mcp-debug-<os>-<arch>). It avoids dependencies like jq.

# Helper function for logging
info() {
    echo "[INFO] $1"
}

# Check for required tools
for tool in curl grep sed tr uname; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo "[ERROR] '$tool' is not installed. Please install it to continue."
        exit 1
    fi
done

# Determine OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

info "Detected OS: $OS"
info "Detected Arch: $ARCH"

# GitHub API URL for the latest release
API_URL="https://api.github.com/repos/giantswarm/mcp-debug/releases/latest"

info "Fetching latest release tag from GitHub API..."

# Get the latest version tag from the API using grep and sed
LATEST_TAG=$(curl -s "$API_URL" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "[ERROR] Could not find the latest release tag from the GitHub API."
    exit 1
fi

info "Latest version tag is $LATEST_TAG"

# Construct the download URL. Releases are built by the architect CircleCI
# pipeline (cli flavour), which attaches a raw, per-platform binary named
# mcp-debug-<os>-<arch> to the GitHub Release -- not a versioned tarball.
BINARY_NAME="mcp-debug-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/giantswarm/mcp-debug/releases/download/${LATEST_TAG}/${BINARY_NAME}"

info "Constructed download URL: $DOWNLOAD_URL"

# Download the binary directly.
curl -fL -o mcp-debug "${DOWNLOAD_URL}"
info "Download complete."

chmod +x mcp-debug
info "Made binary executable."

info "Installation successful! The 'mcp-debug' binary is now in your current directory."
info "You can run it with: ./mcp-debug"
info "For system-wide access, move it to a directory in your PATH, e.g.:"
info "sudo mv mcp-debug /usr/local/bin/mcp-debug"
