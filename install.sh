#!/bin/bash
set -e

# 1. Define your configuration parameters
REPO_OWNER="codot-product" # Change this to your actual GitHub username/org
REPO_NAME="codot-gateway"
BINARY_NAME="codot-gateway"
INSTALL_DIR="/usr/local/bin"

echo "🤖 Initializing Codot Gateway Installer..."

# 2. Detect System OS Layout
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# 3. Map System Metrics to your Cross-Compiled Binary Names
case "$OS" in
    darwin)
        if [ "$ARCH" = "arm64" ]; then
            SUFFIX="mac-arm64"
        else
            SUFFIX="mac-intel"
        fi
        ;;
    linux)
        if [ "$ARCH" = "x86_64" ] || [ "$ARCH" = "amd64" ]; then
            SUFFIX="linux-amd64"
        else
            echo "❌ Unsupported Linux Architecture: $ARCH"
            exit 1
        fi
        ;;
    *)
        echo "❌ Unsupported Operating System: $OS"
        exit 1
        ;;
esac

# 4. Construct the GitHub Target Asset Download Path
DOWNLOAD_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/latest/download/${BINARY_NAME}-${SUFFIX}"
TMP_FILE="/tmp/${BINARY_NAME}"

echo "📥 Downloading compiled asset for your machine architecture..."
echo "🔗 Source: $DOWNLOAD_URL"

# Execute network fetch cleanly via curl
if ! curl -sSL -fail -o "$TMP_FILE" "$DOWNLOAD_URL"; then
    echo "❌ Download failed! Please confirm that the release version is published and public."
    exit 1
fi

# 5. Make the asset executable and transition it to the system PATH globally
chmod +x "$TMP_FILE"

echo "🔐 Deploying binary asset destination path to $INSTALL_DIR..."
# If the target directory requires admin access, run elevation check gracefully
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
    echo "🔑 Administrative privileges required to link binary to system path. Prompting sudo..."
    sudo mv "$TMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

echo ""
echo "=================================================================="
echo "🎉 CODOT GATEWAY SUCCESSFULLY INSTALLED!"
echo "👉 Run 'codot-gateway' from any terminal path window to begin."
echo "=================================================================="
echo ""
