#!/bin/sh
set -e

REPO="ilyas-bkgo/devcheck"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "❌ Error: Unsupported operating system: $OS"
    exit 1
    ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "❌ Error: Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Determine installation directory
if [ -w "/usr/local/bin" ]; then
  BIN_DIR="/usr/local/bin"
  USE_SUDO=""
elif command -v sudo >/dev/null 2>&1; then
  BIN_DIR="/usr/local/bin"
  USE_SUDO="sudo"
else
  BIN_DIR="$HOME/.local/bin"
  USE_SUDO=""
  mkdir -p "$BIN_DIR"
fi

echo "🔍 Fetching latest devcheck release..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
  echo "❌ Error: Could not fetch latest release version from GitHub."
  exit 1
fi

VERSION="${LATEST_TAG#v}"
FILE_NAME="devcheck_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${FILE_NAME}"

echo "📦 Downloading devcheck ${LATEST_TAG} (${OS}/${ARCH})..."
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$FILE_NAME"

echo "⚙️ Installing devcheck to ${BIN_DIR}..."
tar -xzf "$TMP_DIR/$FILE_NAME" -C "$TMP_DIR" devcheck

if [ -n "$USE_SUDO" ]; then
  $USE_SUDO mv "$TMP_DIR/devcheck" "$BIN_DIR/devcheck"
  $USE_SUDO chmod +x "$BIN_DIR/devcheck"
else
  mv "$TMP_DIR/devcheck" "$BIN_DIR/devcheck"
  chmod +x "$BIN_DIR/devcheck"
fi

echo ""
echo "✨ devcheck installed successfully to ${BIN_DIR}/devcheck!"
echo "🚀 Run 'devcheck init' to get started."
