#!/bin/bash
#
# OCPP Emulator Deployment Script for Ubuntu
#
# This script deploys:
# - Backend service binary with systemd
# - Config file to /etc/conf/ocpp-emu.yml
# - Web app to /var/www/html/ocpp-emu
#
# Usage:
#   sudo ./deploy-ubuntu.sh [OPTIONS]
#
# Options:
#   --version VERSION    Version to download (e.g., v1.0.0), or 'local' for local build
#   --binary PATH        Path to local binary (optional, uses download if not specified)
#   --web PATH           Path to local web dist directory (optional)
#   --config PATH        Path to custom config file (optional)
#   --skip-service       Don't install/restart systemd service
#   --skip-web           Don't deploy web files
#   --uninstall          Remove installation
#
# Examples:
#   sudo ./deploy-ubuntu.sh --version v1.0.0
#   sudo ./deploy-ubuntu.sh --binary ./ocpp-emu --web ./web/dist
#   sudo ./deploy-ubuntu.sh --uninstall
#

set -e

# Configuration
GITHUB_REPO="ruslan-hut/ocpp-emu"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/conf"
CONFIG_FILE="${CONFIG_DIR}/ocpp-emu.yml"
WEB_DIR="/var/www/html/ocpp-emu"
SERVICE_NAME="ocpp-emu"
SERVICE_USER="ocpp-emu"
LOG_DIR="/var/log/ocpp-emu"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
VERSION=""
LOCAL_BINARY=""
LOCAL_WEB=""
LOCAL_CONFIG=""
SKIP_SERVICE=false
SKIP_WEB=false
UNINSTALL=false

# Logging functions (output to stderr to not interfere with function returns)
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --binary)
            LOCAL_BINARY="$2"
            shift 2
            ;;
        --web)
            LOCAL_WEB="$2"
            shift 2
            ;;
        --config)
            LOCAL_CONFIG="$2"
            shift 2
            ;;
        --skip-service)
            SKIP_SERVICE=true
            shift
            ;;
        --skip-web)
            SKIP_WEB=true
            shift
            ;;
        --uninstall)
            UNINSTALL=true
            shift
            ;;
        -h|--help)
            head -30 "$0" | tail -25
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Uninstall function
uninstall() {
    log_info "Uninstalling OCPP Emulator..."

    # Stop and disable service
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        log_info "Stopping service..."
        systemctl stop ${SERVICE_NAME}
    fi

    if systemctl is-enabled --quiet ${SERVICE_NAME} 2>/dev/null; then
        log_info "Disabling service..."
        systemctl disable ${SERVICE_NAME}
    fi

    # Remove files
    log_info "Removing files..."
    rm -f /etc/systemd/system/${SERVICE_NAME}.service
    rm -f ${INSTALL_DIR}/ocpp-emu
    rm -rf ${WEB_DIR}
    rm -rf ${LOG_DIR}

    # Don't remove config by default
    log_warn "Config file preserved at ${CONFIG_FILE}"
    log_warn "Remove manually if needed: sudo rm -rf ${CONFIG_DIR}"

    # Remove user
    if id ${SERVICE_USER} &>/dev/null; then
        log_info "Removing service user..."
        userdel ${SERVICE_USER} 2>/dev/null || true
    fi

    systemctl daemon-reload

    log_info "Uninstallation complete"
    exit 0
}

# Create service user
create_user() {
    if ! id ${SERVICE_USER} &>/dev/null; then
        log_info "Creating service user: ${SERVICE_USER}"
        useradd --system --no-create-home --shell /bin/false ${SERVICE_USER}
    else
        log_info "Service user already exists: ${SERVICE_USER}"
    fi
}

# Create directories
create_directories() {
    log_info "Creating directories..."

    mkdir -p ${CONFIG_DIR}
    mkdir -p ${WEB_DIR}
    mkdir -p ${LOG_DIR}

    chown ${SERVICE_USER}:${SERVICE_USER} ${LOG_DIR}
}

# Download binary from GitHub releases
download_binary() {
    local version=$1
    local arch=$(detect_arch)
    local url="https://github.com/${GITHUB_REPO}/releases/download/${version}/ocpp-emu-linux-${arch}"

    log_info "Downloading binary from ${url}..."

    if command -v curl &>/dev/null; then
        curl -fsSL -o /tmp/ocpp-emu "${url}"
    elif command -v wget &>/dev/null; then
        wget -q -O /tmp/ocpp-emu "${url}"
    else
        log_error "Neither curl nor wget found"
        exit 1
    fi

    chmod +x /tmp/ocpp-emu
    echo "/tmp/ocpp-emu"
}

# Download web files from GitHub releases
download_web() {
    local version=$1
    local url="https://github.com/${GITHUB_REPO}/releases/download/${version}/ocpp-emu-web.tar.gz"

    log_info "Downloading web files from ${url}..."

    if command -v curl &>/dev/null; then
        curl -fsSL -o /tmp/ocpp-emu-web.tar.gz "${url}"
    elif command -v wget &>/dev/null; then
        wget -q -O /tmp/ocpp-emu-web.tar.gz "${url}"
    else
        log_error "Neither curl nor wget found"
        exit 1
    fi

    echo "/tmp/ocpp-emu-web.tar.gz"
}

# Install binary
install_binary() {
    local binary_path=$1

    log_info "Installing binary to ${INSTALL_DIR}/ocpp-emu..."

    # Stop service if running
    if systemctl is-active --quiet ${SERVICE_NAME} 2>/dev/null; then
        log_info "Stopping existing service..."
        systemctl stop ${SERVICE_NAME}
    fi

    cp "${binary_path}" ${INSTALL_DIR}/ocpp-emu
    chmod 755 ${INSTALL_DIR}/ocpp-emu
    chown root:root ${INSTALL_DIR}/ocpp-emu

    # Verify
    if ${INSTALL_DIR}/ocpp-emu --version 2>/dev/null || true; then
        log_info "Binary installed successfully"
    fi
}

# Install config
install_config() {
    local config_path=$1

    if [[ -f ${CONFIG_FILE} ]]; then
        log_warn "Config file already exists at ${CONFIG_FILE}"
        log_warn "Creating backup at ${CONFIG_FILE}.bak"
        cp ${CONFIG_FILE} ${CONFIG_FILE}.bak
    fi

    log_info "Installing config to ${CONFIG_FILE}..."

    if [[ -n "${config_path}" && -f "${config_path}" ]]; then
        cp "${config_path}" ${CONFIG_FILE}
    else
        # Create default config
        cat > ${CONFIG_FILE} << 'EOF'
# OCPP Emulator Configuration
# See documentation for all options

server:
  host: "0.0.0.0"
  port: 8080

mongodb:
  uri: "mongodb://localhost:27017"
  database: "ocpp_emu"

csms:
  default_url: "ws://localhost:9000/ocpp"
  connect_timeout: 30s
  ping_interval: 30s
  reconnect_delay: 5s
  max_reconnect_delay: 5m

logging:
  level: "info"
  format: "json"

auth:
  enabled: true
  jwt_secret: "change-me-in-production-use-at-least-32-characters"
  jwt_expiry: 24h
  users:
    - username: "admin"
      # Default password: admin (change in production!)
      # Generate new hash: go run cmd/tools/hashpw/main.go "your-password"
      password_hash: "$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqcBcLZ5HmT1h.CyD3ZL0K3Qb5IKi"
      role: "admin"
      enabled: true
    - username: "viewer"
      # Default password: viewer
      password_hash: "$2a$10$XQxBtJXKQZPveuJqNk3Tq.vI6X8Jn5qzJHp8RqKL0K3Qb5IKiViewer"
      role: "viewer"
      enabled: true
  api_keys: []
EOF
    fi

    chmod 640 ${CONFIG_FILE}
    chown root:${SERVICE_USER} ${CONFIG_FILE}

    log_info "Config installed"
    log_warn "Review and update ${CONFIG_FILE} before starting the service"
}

# Install web files
install_web() {
    local web_path=$1

    log_info "Installing web files to ${WEB_DIR}..."

    # Clear existing files
    rm -rf ${WEB_DIR}/*

    if [[ -d "${web_path}" ]]; then
        # Copy from directory
        cp -r "${web_path}"/* ${WEB_DIR}/
    elif [[ -f "${web_path}" && "${web_path}" == *.tar.gz ]]; then
        # Extract from tarball
        tar -xzf "${web_path}" -C ${WEB_DIR}
    elif [[ -f "${web_path}" && "${web_path}" == *.zip ]]; then
        # Extract from zip
        unzip -q "${web_path}" -d ${WEB_DIR}
    else
        log_error "Invalid web path: ${web_path}"
        exit 1
    fi

    # Set permissions
    chown -R www-data:www-data ${WEB_DIR}
    chmod -R 755 ${WEB_DIR}

    log_info "Web files installed"
}

# Install systemd service
install_service() {
    log_info "Installing systemd service..."

    cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=OCPP Charging Station Emulator
Documentation=https://github.com/${GITHUB_REPO}
After=network.target mongodb.service
Wants=mongodb.service

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
ExecStart=${INSTALL_DIR}/ocpp-emu --config ${CONFIG_FILE}
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
ReadWritePaths=${LOG_DIR}

# Environment
Environment=GIN_MODE=release

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=${SERVICE_NAME}

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable ${SERVICE_NAME}

    log_info "Systemd service installed and enabled"
}

# Start service
start_service() {
    log_info "Starting service..."
    systemctl start ${SERVICE_NAME}

    sleep 2

    if systemctl is-active --quiet ${SERVICE_NAME}; then
        log_info "Service started successfully"
        systemctl status ${SERVICE_NAME} --no-pager
    else
        log_error "Service failed to start"
        journalctl -u ${SERVICE_NAME} --no-pager -n 20
        exit 1
    fi
}

# Print nginx config suggestion
print_nginx_config() {
    cat << EOF

${GREEN}=== Nginx Configuration ===${NC}

Add this to your nginx configuration to serve the web app and proxy API requests:

server {
    listen 80;
    server_name your-domain.com;  # Update with your domain

    # Web app (static files)
    root ${WEB_DIR};
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API proxy
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 86400;
    }
}

EOF
}

# Main installation
main() {
    check_root

    if [[ "${UNINSTALL}" == "true" ]]; then
        uninstall
    fi

    log_info "Starting OCPP Emulator deployment..."

    # Determine binary source
    local binary_path=""
    if [[ -n "${LOCAL_BINARY}" ]]; then
        if [[ ! -f "${LOCAL_BINARY}" ]]; then
            log_error "Binary not found: ${LOCAL_BINARY}"
            exit 1
        fi
        binary_path="${LOCAL_BINARY}"
    elif [[ -n "${VERSION}" ]]; then
        binary_path=$(download_binary "${VERSION}")
    else
        log_error "Please specify --version or --binary"
        exit 1
    fi

    # Create user and directories
    create_user
    create_directories

    # Install binary
    install_binary "${binary_path}"

    # Install config
    if [[ ! -f ${CONFIG_FILE} ]] || [[ -n "${LOCAL_CONFIG}" ]]; then
        install_config "${LOCAL_CONFIG}"
    else
        log_info "Keeping existing config at ${CONFIG_FILE}"
    fi

    # Install web files
    if [[ "${SKIP_WEB}" != "true" ]]; then
        local web_path=""
        if [[ -n "${LOCAL_WEB}" ]]; then
            if [[ ! -e "${LOCAL_WEB}" ]]; then
                log_error "Web files not found: ${LOCAL_WEB}"
                exit 1
            fi
            web_path="${LOCAL_WEB}"
        elif [[ -n "${VERSION}" ]]; then
            web_path=$(download_web "${VERSION}")
        fi

        if [[ -n "${web_path}" ]]; then
            install_web "${web_path}"
        fi
    fi

    # Install and start service
    if [[ "${SKIP_SERVICE}" != "true" ]]; then
        install_service
        start_service
    fi

    # Print summary
    echo ""
    log_info "=== Deployment Complete ==="
    echo ""
    echo "Binary:  ${INSTALL_DIR}/ocpp-emu"
    echo "Config:  ${CONFIG_FILE}"
    echo "Web:     ${WEB_DIR}"
    echo "Logs:    journalctl -u ${SERVICE_NAME} -f"
    echo ""
    echo "Service commands:"
    echo "  sudo systemctl status ${SERVICE_NAME}"
    echo "  sudo systemctl restart ${SERVICE_NAME}"
    echo "  sudo systemctl stop ${SERVICE_NAME}"
    echo ""

    print_nginx_config

    log_warn "Remember to:"
    log_warn "  1. Update ${CONFIG_FILE} with your settings"
    log_warn "  2. Change default passwords!"
    log_warn "  3. Configure nginx or another reverse proxy"
    log_warn "  4. Ensure MongoDB is running"
}

main
