#!/bin/bash
set -e

# This script downloads the latest mcp-debug .tar.gz release archive by
# querying the GitHub API for the latest tag, constructing the correct filename,
# and extracting the binary. It avoids dependencies like jq.

# Helper function for logging
info() {
    echo "[INFO] $1"
}

# Check for required tools
for tool in curl grep sed tar tr uname; do
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
VERSION=$(echo "$LATEST_TAG" | sed 's/^v//')

# Construct the download URL for the .tar.gz archive
ARCHIVE_NAME="mcp-debug_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/giantswarm/mcp-debug/releases/download/${LATEST_TAG}/${ARCHIVE_NAME}"

info "Constructed download URL: $DOWNLOAD_URL"

# Download the archive
curl -L -o "${ARCHIVE_NAME}" "${DOWNLOAD_URL}"
info "Download complete."

# Un-tar the archive to extract the binary and then clean up
info "Extracting binary from ${ARCHIVE_NAME}..."
tar -xzf "${ARCHIVE_NAME}" mcp-debug
rm "${ARCHIVE_NAME}"
info "Extraction complete."

chmod +x mcp-debug
info "Made binary executable."

info "Installation successful! The 'mcp-debug' binary is now in your current directory."
info "You can run it with: ./mcp-debug"
info "For system-wide access, move it to a directory in your PATH, e.g.:"
info "sudo mv mcp-debug /usr/local/bin/mcp-debug" 