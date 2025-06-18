#!/bin/bash
# SPIRE Setup Script
# AIDEV-NOTE: Installs and configures SPIRE server and agent with proper permissions

set -euo pipefail

# AIDEV-NOTE: Configuration variables
SPIRE_VERSION="${SPIRE_VERSION:-1.12.2}"
SPIRE_ARCH="${SPIRE_ARCH:-linux-amd64-musl}"
SPIRE_INSTALL_DIR="/usr/local/bin"
SPIRE_CONFIG_DIR="/etc/spire"
SPIRE_DATA_DIR="/var/lib/spire"
SPIRE_RUN_DIR="/run/spire"

# AIDEV-NOTE: Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# AIDEV-NOTE: Check if running as root
if [[ $EUID -ne 0 ]]; then
   log_error "This script must be run as root"
   exit 1
fi

# AIDEV-NOTE: Create system users
create_users() {
    log_info "Creating SPIRE system users..."
    
    # Create spire-server user
    if ! id -u spire-server >/dev/null 2>&1; then
        useradd --system --shell /bin/false --home-dir /nonexistent --comment "SPIRE Server" spire-server
        log_info "Created spire-server user"
    fi
    
    # Create spire-agent user
    if ! id -u spire-agent >/dev/null 2>&1; then
        useradd --system --shell /bin/false --home-dir /nonexistent --comment "SPIRE Agent" spire-agent
        log_info "Created spire-agent user"
    fi
}

# AIDEV-NOTE: Create directory structure
create_directories() {
    log_info "Creating SPIRE directory structure..."
    
    # Server directories
    mkdir -p "${SPIRE_CONFIG_DIR}/server"
    mkdir -p "${SPIRE_DATA_DIR}/server/"{data,keys}
    mkdir -p "${SPIRE_RUN_DIR}"
    
    # Agent directories
    mkdir -p "${SPIRE_CONFIG_DIR}/agent"
    mkdir -p "${SPIRE_DATA_DIR}/agent/"{data,keys}
    
    # Set ownership
    chown -R spire-server:spire-server "${SPIRE_DATA_DIR}/server"
    chown -R spire-agent:spire-agent "${SPIRE_DATA_DIR}/agent"
    chown root:root "${SPIRE_CONFIG_DIR}"
    chown root:root "${SPIRE_RUN_DIR}"
    
    # Set permissions
    chmod 755 "${SPIRE_CONFIG_DIR}"
    chmod 755 "${SPIRE_RUN_DIR}"
    chmod 700 "${SPIRE_DATA_DIR}/server/keys"
    chmod 700 "${SPIRE_DATA_DIR}/agent/keys"
    
    log_info "Directory structure created"
}

# AIDEV-NOTE: Download and install SPIRE binaries
install_spire() {
    log_info "Installing SPIRE ${SPIRE_VERSION}..."
    
    # Check if already installed
    if command -v spire-server >/dev/null 2>&1; then
        local installed_version=$(spire-server --version 2>&1 | grep -oP 'v\K[0-9.]+' || echo "unknown")
        if [[ "$installed_version" == "$SPIRE_VERSION" ]]; then
            log_info "SPIRE ${SPIRE_VERSION} already installed"
            return 0
        else
            log_warn "SPIRE ${installed_version} installed, upgrading to ${SPIRE_VERSION}"
        fi
    fi
    
    # Download SPIRE
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    local download_url="https://github.com/spiffe/spire/releases/download/v${SPIRE_VERSION}/spire-${SPIRE_VERSION}-${SPIRE_ARCH}.tar.gz"
    log_info "Downloading SPIRE from ${download_url}"
    
    if ! curl -sSfL "$download_url" -o spire.tar.gz; then
        log_error "Failed to download SPIRE"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Extract and install
    tar xzf spire.tar.gz
    cp spire-*/bin/spire-server "${SPIRE_INSTALL_DIR}/"
    cp spire-*/bin/spire-agent "${SPIRE_INSTALL_DIR}/"
    chmod +x "${SPIRE_INSTALL_DIR}/spire-server" "${SPIRE_INSTALL_DIR}/spire-agent"
    
    # Cleanup
    cd - >/dev/null
    rm -rf "$temp_dir"
    
    log_info "SPIRE ${SPIRE_VERSION} installed successfully"
}

# AIDEV-NOTE: Generate initial trust bundle
generate_trust_bundle() {
    log_info "Generating initial trust bundle..."
    
    local bundle_path="${SPIRE_CONFIG_DIR}/agent/bundle.crt"
    
    # AIDEV-NOTE: For production, this should be the actual CA certificate
    # For now, create a placeholder that will be replaced during bootstrap
    if [[ ! -f "$bundle_path" ]]; then
        cat > "$bundle_path" << 'EOF'
# SPIRE Server CA Bundle
# AIDEV-NOTE: This file will be populated with the actual CA certificate
# during the agent bootstrap process
EOF
        chmod 644 "$bundle_path"
        log_warn "Created placeholder trust bundle at ${bundle_path}"
        log_warn "You must replace this with the actual server CA certificate"
    fi
}

# AIDEV-NOTE: Install systemd service files
install_systemd_services() {
    log_info "Installing systemd service files..."
    
    # Copy service files if they exist in the repo
    local script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local systemd_dir="${script_dir}/../systemd"
    
    if [[ -d "$systemd_dir" ]]; then
        cp "${systemd_dir}/spire-server.service" /etc/systemd/system/
        cp "${systemd_dir}/spire-agent.service" /etc/systemd/system/
        
        # Copy configuration files
        cp "${script_dir}/../server/spire-server.conf" "${SPIRE_CONFIG_DIR}/server/"
        cp "${script_dir}/../agent/spire-agent.conf" "${SPIRE_CONFIG_DIR}/agent/"
        
        log_info "Service files installed"
    else
        log_error "Systemd service files not found at ${systemd_dir}"
        exit 1
    fi
    
    # Reload systemd
    systemctl daemon-reload
}

# AIDEV-NOTE: Create environment-specific configuration
create_environment_config() {
    local environment="${1:-dev}"
    log_info "Creating environment configuration for: ${environment}"
    
    # Create drop-in directory
    mkdir -p "/etc/systemd/system/spire-server.service.d"
    mkdir -p "/etc/systemd/system/spire-agent.service.d"
    
    # Server environment config
    cat > "/etc/systemd/system/spire-server.service.d/environment.conf" << EOF
[Service]
# AIDEV-NOTE: Environment-specific configuration for ${environment}
Environment="UNKEY_SPIRE_TRUST_DOMAIN=${environment}.unkey.app"
Environment="UNKEY_SPIRE_LOG_LEVEL=${SPIRE_LOG_LEVEL:-INFO}"
Environment="UNKEY_SPIRE_DB_TYPE=sqlite3"
Environment="UNKEY_SPIRE_DB_CONNECTION=/var/lib/spire/server/data/datastore.sqlite3"
EOF
    
    # Agent environment config
    cat > "/etc/systemd/system/spire-agent.service.d/environment.conf" << EOF
[Service]
# AIDEV-NOTE: Environment-specific configuration for ${environment}
Environment="UNKEY_SPIRE_TRUST_DOMAIN=${environment}.unkey.app"
Environment="UNKEY_SPIRE_LOG_LEVEL=${SPIRE_LOG_LEVEL:-INFO}"
Environment="UNKEY_SPIRE_SERVER_ADDRESS=127.0.0.1"
EOF
    
    systemctl daemon-reload
    log_info "Environment configuration created"
}

# AIDEV-NOTE: Main setup function
main() {
    log_info "Starting SPIRE setup..."
    
    # Parse arguments
    local environment="${1:-dev}"
    
    # Run setup steps
    create_users
    create_directories
    install_spire
    generate_trust_bundle
    install_systemd_services
    create_environment_config "$environment"
    
    log_info "SPIRE setup completed successfully!"
    log_info ""
    log_info "Next steps:"
    log_info "1. Review and update configuration files in ${SPIRE_CONFIG_DIR}"
    log_info "2. Generate and distribute the trust bundle to agents"
    log_info "3. Start SPIRE server: systemctl start spire-server"
    log_info "4. Generate join tokens for agents"
    log_info "5. Start SPIRE agent: systemctl start spire-agent"
    log_info ""
    log_info "For production environments, ensure you:"
    log_info "- Use PostgreSQL instead of SQLite"
    log_info "- Configure AWS KMS for key management"
    log_info "- Set up proper TLS certificates"
    log_info "- Configure monitoring and alerting"
}

# Run main function
main "$@"