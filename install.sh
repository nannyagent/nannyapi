#!/bin/bash
#
# NannyAPI Installation Script
# Supports: Linux (amd64, arm64)
#

set -euo pipefail

# Configuration
BINARY_NAME="nannyapi"
INSTALL_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/${BINARY_NAME}.service"
WORKING_DIR="/var/lib/nannyapi"
GITHUB_REPO="nannyagent/nannyapi"
VERSION="${NANNYAPI_VERSION:-latest}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)     OS="linux";;
        *)
            log_error "Only Linux is supported. Detected: $(uname -s)"
            ;;
    esac
    log_info "Detected OS: ${OS}"
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   ARCH="amd64";;
        aarch64|arm64)  ARCH="arm64";;
        *)
            log_error "Unsupported architecture: $(uname -m). Only amd64 and arm64 are supported."
            ;;
    esac
    log_info "Detected architecture: ${ARCH}"
}

# Check if running as root
check_root() {
    if [[ "$EUID" -ne 0 ]]; then
        log_error "This script must be run as root. Use: sudo $0"
    fi
}

# Get the latest release version from GitHub
get_latest_version() {
    if [[ "${VERSION}" == "latest" ]]; then
        log_info "Fetching latest version from GitHub..."
        VERSION=$(curl -sL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ -z "${VERSION}" ]]; then
            log_error "Failed to fetch latest version. Check your internet connection."
        fi
    fi
    log_info "Version to install: ${VERSION}"
}

# Construct download URL
get_download_url() {
    local version_no_v="${VERSION#v}"
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${BINARY_NAME}_${version_no_v}_${OS}_${ARCH}.tar.gz"
    log_info "Download URL: ${DOWNLOAD_URL}"
}

# Download and extract binary
download_binary() {
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf ${tmp_dir}" EXIT

    log_info "Downloading ${BINARY_NAME}..."

    if command -v curl &> /dev/null; then
        curl -fsSL "${DOWNLOAD_URL}" -o "${tmp_dir}/${BINARY_NAME}.tar.gz" || log_error "Download failed. Check if release exists for ${OS}/${ARCH}."
    elif command -v wget &> /dev/null; then
        wget -q "${DOWNLOAD_URL}" -O "${tmp_dir}/${BINARY_NAME}.tar.gz" || log_error "Download failed. Check if release exists for ${OS}/${ARCH}."
    else
        log_error "Neither curl nor wget found. Please install one of them."
    fi

    log_info "Extracting archive..."
    tar -xzf "${tmp_dir}/${BINARY_NAME}.tar.gz" -C "${tmp_dir}"

    # Find the binary (might be in root or subdirectory)
    local binary_path
    binary_path=$(find "${tmp_dir}" -name "${BINARY_NAME}" -type f | head -1)
    if [[ -z "${binary_path}" ]]; then
        log_error "Binary not found in archive"
    fi

    # Backup existing binary
    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        local backup_name="${BINARY_NAME}.bak.$(date +%Y%m%d%H%M%S)"
        log_info "Backing up existing binary to ${INSTALL_DIR}/${backup_name}"
        mv "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${backup_name}"
    fi

    # Install binary
    log_info "Installing binary to ${INSTALL_DIR}/${BINARY_NAME}"
    mkdir -p "${INSTALL_DIR}"
    cp "${binary_path}" "${INSTALL_DIR}/${BINARY_NAME}"
    chmod +x "${INSTALL_DIR}/${BINARY_NAME}"
}

# Verify installation
verify_installation() {
    if [[ -x "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        log_success "Binary installed successfully"
        log_info "Version: $(${INSTALL_DIR}/${BINARY_NAME} --version 2>/dev/null || echo 'Unable to determine version')"
    else
        log_error "Installation verification failed"
    fi
}

# Create systemd service
create_systemd_service() {
    if [[ -f "${SERVICE_FILE}" ]]; then
        log_warn "Systemd service already exists at ${SERVICE_FILE}"
        read -p "Overwrite? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Skipping service creation"
            return
        fi
    fi

    log_info "Creating systemd service..."

    # Create working directory
    mkdir -p "${WORKING_DIR}"
    mkdir -p "${WORKING_DIR}/pb_data"

    # Create .env file if it doesn't exist
    if [[ ! -f "${WORKING_DIR}/.env" ]]; then
        cat > "${WORKING_DIR}/.env" << 'ENVEOF'
# NannyAPI Environment Configuration
# See documentation for all available options

# Frontend URL for device authorization
FRONTEND_URL=http://localhost:8080

# Auto-migrate database (set to false in production)
PB_AUTOMIGRATE=true

# Optional: GitHub OAuth
# GITHUB_CLIENT_ID=
# GITHUB_CLIENT_SECRET=

# Optional: Google OAuth
# GOOGLE_CLIENT_ID=
# GOOGLE_CLIENT_SECRET=

# Optional: ClickHouse for TensorZero
# CLICKHOUSE_URL=
ENVEOF
        log_info "Created default .env at ${WORKING_DIR}/.env"
    fi

    # Create systemd service file
    cat > "${SERVICE_FILE}" << EOF
[Unit]
Description=NannyAPI Server
Documentation=https://nannyai.dev/docs/api_reference
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${WORKING_DIR}
EnvironmentFile=${WORKING_DIR}/.env
ExecStart=${INSTALL_DIR}/${BINARY_NAME} serve --http="0.0.0.0:8090" --dir="${WORKING_DIR}/pb_data"
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${BINARY_NAME}

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${WORKING_DIR}
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd
    systemctl daemon-reload
    log_success "Systemd service created at ${SERVICE_FILE}"
}

# Print post-installation instructions
print_instructions() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_success "NannyAPI ${VERSION} installed successfully!"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Next Steps:"
    echo ""
    echo "  1. Edit configuration:"
    echo "     ${YELLOW}sudo nano ${WORKING_DIR}/.env${NC}"
    echo ""
    echo "  2. Create admin user:"
    echo "     ${YELLOW}sudo ${INSTALL_DIR}/${BINARY_NAME} superuser upsert admin@example.com YourSecurePassword123!${NC}"
    echo ""
    echo "  3. Start the service:"
    echo "     ${YELLOW}sudo systemctl start ${BINARY_NAME}${NC}"
    echo ""
    echo "  4. Enable on boot:"
    echo "     ${YELLOW}sudo systemctl enable ${BINARY_NAME}${NC}"
    echo ""
    echo "  5. Check status:"
    echo "     ${YELLOW}sudo systemctl status ${BINARY_NAME}${NC}"
    echo ""
    echo "  6. View logs:"
    echo "     ${YELLOW}sudo journalctl -u ${BINARY_NAME} -f${NC}"
    echo ""
    echo "Documentation: https://github.com/${GITHUB_REPO}"
    echo "Admin UI (after start): http://localhost:8090/_/"
    echo ""
}

# Uninstall function
uninstall() {
    log_warn "Uninstalling NannyAPI..."

    if systemctl is-active --quiet "${BINARY_NAME}" 2>/dev/null; then
        systemctl stop "${BINARY_NAME}"
    fi
    if [[ -f "${SERVICE_FILE}" ]]; then
        systemctl disable "${BINARY_NAME}" 2>/dev/null || true
        rm -f "${SERVICE_FILE}"
        systemctl daemon-reload
    fi

    if [[ -f "${INSTALL_DIR}/${BINARY_NAME}" ]]; then
        rm -f "${INSTALL_DIR}/${BINARY_NAME}"
        log_success "Binary removed"
    fi

    log_warn "Data directory ${WORKING_DIR} was NOT removed. Remove manually if needed."
    log_success "NannyAPI uninstalled"
    exit 0
}

# Main installation flow
main() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════════════╗"
    echo "║                    NannyAPI Installation Script                    ║"
    echo "╚════════════════════════════════════════════════════════════════════╝"
    echo ""

    # Handle uninstall flag
    if [[ "${1:-}" == "--uninstall" ]] || [[ "${1:-}" == "-u" ]]; then
        detect_os
        check_root
        uninstall
    fi

    # Installation flow
    detect_os
    detect_arch
    check_root
    get_latest_version
    get_download_url
    download_binary
    verify_installation
    create_systemd_service
    print_instructions
}

# Run main with all arguments
main "$@"
