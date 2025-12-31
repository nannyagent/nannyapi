#!/bin/bash

set -e

BINARY_NAME="nannyapi"
INSTALL_DIR="/usr/local/bin"
SERVICE_FILE="/etc/systemd/system/${BINARY_NAME}.service"
# TODO: Update this URL to the actual release URL
DOWNLOAD_URL="https://github.com/nannyagent/nannyapi/releases/latest/download/${BINARY_NAME}"
BACKUP_SUFFIX=".bkp.$(date +%d%m%y)"
WORKING_DIR="/var/lib/nannyapi"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

echo "Starting installation of ${BINARY_NAME}..."

# Create install directory if it doesn't exist
mkdir -p "${INSTALL_DIR}"

# Backup existing binary if it exists
if [ -f "${INSTALL_DIR}/${BINARY_NAME}" ]; then
    echo "Backing up existing binary to ${INSTALL_DIR}/${BINARY_NAME}${BACKUP_SUFFIX}..."
    cp "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}${BACKUP_SUFFIX}"
fi

# Download latest binary
echo "Downloading latest binary from ${DOWNLOAD_URL}..."
# Note: In a real scenario, we might need to handle architecture (amd64/arm64)
# For now, we assume the URL points to the correct binary or a redirect
curl -L -o "${INSTALL_DIR}/${BINARY_NAME}.new" "${DOWNLOAD_URL}"
chmod +x "${INSTALL_DIR}/${BINARY_NAME}.new"
mv "${INSTALL_DIR}/${BINARY_NAME}.new" "${INSTALL_DIR}/${BINARY_NAME}"

# Print version
echo "Installed version:"
# PocketBase apps usually support --version or version command
"${INSTALL_DIR}/${BINARY_NAME}" --version || echo "Version check failed"

# Create systemd service if it doesn't exist
if [ ! -f "${SERVICE_FILE}" ]; then
    echo "Creating systemd service at ${SERVICE_FILE}..."

    # Create working directory
    mkdir -p "${WORKING_DIR}"

    cat <<EOF > "${SERVICE_FILE}"
[Unit]
Description=NannyAPI Service
After=network.target

[Service]
Type=simple
User=root
# Adjust arguments as needed (e.g., --http="0.0.0.0:8090")
EnvironmentFile=${WORKING_DIR}/.env
ExecStart=${INSTALL_DIR}/${BINARY_NAME} serve --dir="${WORKING_DIR}/pb_data" --publicDir="${WORKING_DIR}/pb_public"
Restart=on-failure
WorkingDirectory=${WORKING_DIR}
StandardOutput=journal
StandardError=journal
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF
    echo "Systemd service created."
else
    echo "Systemd service already exists. Skipping creation."
fi

# Reload systemd daemon
systemctl daemon-reload

echo "Installation complete."
echo "NOTE: Docker installation is currently a template and will be fully supported in future releases."
echo "NOTE: The service has NOT been started."
echo "IMPORTANT: Please run migration scripts and verify configuration before starting the service."
echo "To start the service: systemctl start ${BINARY_NAME}"
