#!/bin/bash
# MCP Registry Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/mcp-reg/mcp-sub-registry/main/scripts/install.sh | bash
#        GITHUB_TOKEN=xxx curl -fsSL ... | GITHUB_TOKEN=xxx bash   # For private repos
#
# Options:
#   --version VERSION   Install specific version (e.g., v1.0.0)
#   --help              Show this help message

set -e

REPO="mcp-reg/mcp-sub-registry"
BINARY_NAME="mcp-registry"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION=""
TEMP_DIR=""

# Build auth header args for curl
AUTH_ARGS=()
if [ -n "$GITHUB_TOKEN" ]; then
    AUTH_ARGS=(-H "Authorization: token $GITHUB_TOKEN")
fi

# Cleanup on exit
cleanup() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        rm -rf "$TEMP_DIR"
    fi
}
trap cleanup EXIT

# Print error and exit
error() {
    echo "Error: $1" >&2
    exit 1
}

# Show help
show_help() {
    cat <<EOF
MCP Registry Installer

Usage:
  curl -fsSL https://raw.githubusercontent.com/mcp-reg/mcp-sub-registry/main/scripts/install.sh | bash
  curl -fsSL ... | bash -s -- --version v1.0.0
  GITHUB_TOKEN=xxx curl -fsSL ... | GITHUB_TOKEN=xxx bash   # For private repos

Options:
  --version VERSION   Install specific version (default: latest)
  --help              Show this help message

Environment:
  INSTALL_DIR         Installation directory (default: \$HOME/.local/bin)
  GITHUB_TOKEN        GitHub personal access token (required for private repos)

EOF
    exit 0
}

# Parse arguments
while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --help|-h)
            show_help
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Check dependencies
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        error "curl is required but not installed"
    fi
}

# Detect OS and architecture
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)   OS="linux" ;;
        darwin)  OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *)       error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *)             error "Unsupported architecture: $ARCH" ;;
    esac

    if [ "$OS" = "windows" ]; then
        BINARY="${BINARY_NAME}-${OS}-${ARCH}.exe"
        INSTALLED_BINARY="${BINARY_NAME}.exe"
    else
        BINARY="${BINARY_NAME}-${OS}-${ARCH}"
        INSTALLED_BINARY="${BINARY_NAME}"
    fi
}

# Get latest release version
get_latest_version() {
    local latest
    latest=$(curl -fsSL "${AUTH_ARGS[@]}" "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ -z "$latest" ]; then
        error "Failed to fetch latest version"
    fi
    echo "$latest"
}

# Check if already installed with same version
check_existing() {
    if [ -x "${INSTALL_DIR}/${INSTALLED_BINARY}" ]; then
        local installed_version
        installed_version=$("${INSTALL_DIR}/${INSTALLED_BINARY}" --version 2>/dev/null | head -1 || echo "")
        if [ -n "$installed_version" ] && echo "$installed_version" | grep -q "$VERSION"; then
            echo "${BINARY_NAME} ${VERSION} is already installed"
            exit 0
        fi
    fi
}

# Verify SHA256 checksum
verify_checksum() {
    local file="$1"
    local checksum_file="$2"

    if ! command -v sha256sum &> /dev/null && ! command -v shasum &> /dev/null; then
        echo "Warning: sha256sum/shasum not found, skipping checksum verification"
        return 0
    fi

    local expected
    expected=$(cat "$checksum_file" | awk '{print $1}')

    local actual
    if command -v sha256sum &> /dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    else
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    fi

    if [ "$expected" != "$actual" ]; then
        error "Checksum verification failed\nExpected: $expected\nActual: $actual"
    fi

    echo "Checksum verified"
}

# Download release asset (handles both public and private repos)
download_asset() {
    local asset_name="$1"
    local output_path="$2"

    # Try direct download first (works for public repos)
    local direct_url="https://github.com/${REPO}/releases/download/${VERSION}/${asset_name}"
    if curl -fsSL "${AUTH_ARGS[@]}" "$direct_url" -o "$output_path" 2>/dev/null; then
        return 0
    fi

    # For private repos, use API to get asset URL
    if [ -n "$GITHUB_TOKEN" ]; then
        local asset_id
        asset_id=$(curl -fsSL "${AUTH_ARGS[@]}" "https://api.github.com/repos/${REPO}/releases/tags/${VERSION}" | \
            grep -B3 "\"name\": \"${asset_name}\"" | grep '"id"' | head -1 | sed -E 's/.*: ([0-9]+).*/\1/')

        if [ -n "$asset_id" ]; then
            if curl -fsSL "${AUTH_ARGS[@]}" -H "Accept: application/octet-stream" \
                "https://api.github.com/repos/${REPO}/releases/assets/${asset_id}" -o "$output_path"; then
                return 0
            fi
        fi
    fi

    return 1
}

# Download and install
install() {
    check_dependencies
    detect_platform

    # Get version
    if [ -z "$VERSION" ]; then
        echo "Fetching latest version..."
        VERSION=$(get_latest_version)
    fi

    # Ensure version starts with 'v'
    case "$VERSION" in
        v*) ;;
        *)  VERSION="v${VERSION}" ;;
    esac

    echo "Installing ${BINARY_NAME} ${VERSION} for ${OS}/${ARCH}..."

    # Check existing installation
    check_existing

    # Create temp directory
    TEMP_DIR=$(mktemp -d)

    # Download binary
    echo "Downloading ${BINARY}..."
    if ! download_asset "${BINARY}" "${TEMP_DIR}/${BINARY}"; then
        error "Failed to download binary. Check if version ${VERSION} exists."
    fi

    # Download and verify checksum
    echo "Verifying checksum..."
    if download_asset "${BINARY}.sha256" "${TEMP_DIR}/${BINARY}.sha256"; then
        verify_checksum "${TEMP_DIR}/${BINARY}" "${TEMP_DIR}/${BINARY}.sha256"
    else
        echo "Warning: Checksum file not found, skipping verification"
    fi

    # Install
    mkdir -p "$INSTALL_DIR"
    mv "${TEMP_DIR}/${BINARY}" "${INSTALL_DIR}/${INSTALLED_BINARY}"
    chmod +x "${INSTALL_DIR}/${INSTALLED_BINARY}"

    echo ""
    echo "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${INSTALLED_BINARY}"

    # PATH warning
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo ""
        echo "Add to your PATH by running:"
        echo ""
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
        echo "Or add that line to your ~/.bashrc or ~/.zshrc"
    fi

    echo ""
    echo "Run '${BINARY_NAME} --help' to get started"
}

install
