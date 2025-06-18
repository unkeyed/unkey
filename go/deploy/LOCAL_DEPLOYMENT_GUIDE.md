# Unkey Services Local Development Environment Setup

This guide provides detailed instructions for setting up a complete Unkey development environment on Linux (Fedora 42 or Ubuntu 22.04+).

## Prerequisites

### 1. System Requirements Check

Before beginning installation, verify your system meets all requirements:

```bash
cd /path/to/unkey/go/deploy
./scripts/check-system-readiness.sh
```

This script verifies:
- Operating system compatibility (Fedora 42+ or Ubuntu 22.04+)
- Required tools: Go 1.24+, Make, Git, systemd
- Container runtime: Docker or Podman
- Virtualization: Firecracker/Cloud Hypervisor, KVM support
- Port availability: 8080-8085, 9464-9467
- Disk space: minimum 5GB free
- Network connectivity

### 2. Fix Any Missing Prerequisites

If the readiness check reports missing dependencies:

**For Fedora:**
```bash
# Install development tools
sudo dnf groupinstall -y "Development Tools"
sudo dnf install -y git make golang curl wget

# Install buf for protobuf generation
sudo ./scripts/install-buf.sh

# Install Docker
sudo dnf install -y docker docker-compose-plugin
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# Install KVM support
sudo dnf install -y qemu-kvm
sudo usermod -aG kvm $USER

# Install Firecracker with jailer (required for metald)
sudo ./scripts/install-firecracker.sh

# Log out and back in for group changes to take effect
```

### 3. Firecracker Jailer Setup (REQUIRED)

The Firecracker jailer provides defense-in-depth security by isolating VMs. Complete these setup steps:

```bash
# Create the jailer user and group
sudo ./scripts/configure-jailer-user.sh --create

# Set up cgroups for Firecracker (prefers cgroup v2)
sudo ./scripts/setup-firecracker-cgroups.sh
# Or force cgroup v2 on hybrid systems:
# sudo CGROUP_VERSION=2 ./scripts/setup-firecracker-cgroups.sh

# Install sudoers configuration and set jailer capabilities
# Note: Run from the metald directory
sudo ./metald/scripts/install-sudoers.sh

# Verify jailer setup
getcap /usr/local/bin/jailer  # Should show: cap_mknod,cap_sys_admin=ep
```

**Notes**: 
- Jailer is REQUIRED for all deployments - the system will not function without it
- The setup script automatically detects and prefers cgroup v2 for better performance and security
- On systems with hybrid cgroup setups, you may need to force cgroup v2 using the CGROUP_VERSION environment variable

**For Ubuntu:**
```bash
# Install development tools
sudo apt update
sudo apt install -y build-essential git make golang curl wget

# Install buf for protobuf generation
sudo ./scripts/install-buf.sh

# Install Docker
sudo apt install -y docker.io docker-compose-v2
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER

# Install KVM support
sudo apt install -y qemu-kvm
sudo usermod -aG kvm $USER

# Install Firecracker with jailer (required for metald)
sudo ./scripts/install-firecracker.sh

# Log out and back in for group changes to take effect
```

### 3. Firecracker Jailer Setup (REQUIRED)

The Firecracker jailer provides defense-in-depth security by isolating VMs. Complete these setup steps:

```bash
# Create the jailer user and group
sudo ./scripts/configure-jailer-user.sh --create

# Set up cgroups for Firecracker (prefers cgroup v2)
sudo ./scripts/setup-firecracker-cgroups.sh
# Or force cgroup v2 on hybrid systems:
# sudo CGROUP_VERSION=2 ./scripts/setup-firecracker-cgroups.sh

# Install sudoers configuration and set jailer capabilities
# Note: Run from the metald directory
sudo ./metald/scripts/install-sudoers.sh

# Verify jailer setup
getcap /usr/local/bin/jailer  # Should show: cap_mknod,cap_sys_admin=ep
```

**Notes**: 
- Jailer is REQUIRED for all deployments - the system will not function without it
- The setup script automatically detects and prefers cgroup v2 for better performance and security
- On systems with hybrid cgroup setups, you may need to force cgroup v2 using the CGROUP_VERSION environment variable

**Note on Firecracker**: The package manager versions of Firecracker often don't include the `jailer` binary, which is required for production deployments. The `install-firecracker.sh` script downloads the official release that includes both `firecracker` and `jailer` binaries.

## Service Installation Order

Services must be installed in this specific order due to dependencies:

1. **Observability Stack** (optional but recommended) - Grafana, Loki, Tempo, Mimir
2. **SPIRE** (REQUIRED) - Service identity and mTLS for secure inter-service communication
3. **assetmanagerd** - VM asset management
4. **billaged** - Usage billing service
5. **builderd** - Container build service
6. **metald** - VM management service (depends on assetmanagerd and billaged)

### Recent Updates

- **Unified Health Endpoints**: All services now expose a standardized health endpoint at `/health` with a consistent JSON response format
- **Service Version**: All services are now at version `0.1.0`
- **Automated SPIRE Registration**: Agent registration is now fully automated with the `register-agent` command
- **Environment-based SPIRE Configuration**: Support for dev, canary, and prod environments
- **Port Changes**: SPIRE Server moved to port 8085 (from 8081) to avoid conflicts

## Complete Installation Process

All commands should be run from the `/path/to/unkey/go/deploy` directory unless otherwise specified.

### Step 1: Observability Stack (Optional but Recommended)

The OTEL-LGTM stack provides comprehensive monitoring with Grafana, Loki (logs), Tempo (traces), and Mimir (metrics).

```bash
# Start the observability stack (uses Podman or Docker)
make o11y

# Verify it's running
podman ps | grep otel-lgtm  # or docker ps | grep otel-lgtm

# Access Grafana dashboard
# URL: http://localhost:3000
# Username: admin
# Password: admin
```

**Available endpoints:**
- Grafana UI: `http://localhost:3000`
- OTLP gRPC: `localhost:4317` (for service telemetry)
- OTLP HTTP: `localhost:4318` (for service telemetry)

### Step 2: SPIRE Installation and Setup

SPIRE provides service identity and automatic mTLS between all services. The installation process now supports environment-based configurations (dev, canary, prod).

#### 2.1 Install SPIRE Components

```bash
# Install SPIRE server and agent binaries (default: dev environment)
make -C spire install

# Or specify a different environment
SPIRE_ENVIRONMENT=canary make -C spire install
SPIRE_ENVIRONMENT=prod make -C spire install

# Reload systemd to recognize new service files
sudo systemctl daemon-reload
```

#### 2.2 Start SPIRE Server

```bash
# Start the SPIRE server
make -C spire service-start-server

# Verify server is running
make -C spire service-status-server

# Check server logs if needed
make -C spire service-logs-server
```

#### 2.3 Bootstrap and Register SPIRE Agent (Automated)

```bash
# Create the trust bundle for the agent
make -C spire bootstrap-agent

# Register the agent automatically (generates join token, configures systemd, starts agent)
make -C spire register-agent

# Verify agent is running and registered
make -C spire service-status-agent
```

The `register-agent` command now provides a fully automated workflow:
- Generates join token automatically
- Configures systemd with the token
- Starts the agent with auto-registration
- Cleans up sensitive data after success
- Handles re-registration gracefully

#### 2.4 Register Unkey Services with SPIRE

```bash
# Register all Unkey services at once
make -C spire register-services

# List all registered entries to verify
make -C spire list-entries
```

### Step 3: Install Core Services

#### 3.1 AssetManagerd Installation

```bash
# Create service user
make -C assetmanagerd create-user

# Build and install the service
make -C assetmanagerd install

# Start the service
make -C assetmanagerd service-start

# Verify it's running
make -C assetmanagerd service-status

# Check logs if needed
make -C assetmanagerd service-logs
```

#### 3.2 Billaged Installation

```bash
# Build and install (creates user automatically)
make -C billaged install

# Start the service
make -C billaged service-start

# Verify it's running
make -C billaged service-status
```

#### 3.3 Builderd Installation

```bash
# Ensure Docker is running (required for builderd)
sudo systemctl status docker

# Build and install
make -C builderd install

# Start the service
make -C builderd service-start

# Verify it's running
make -C builderd service-status
```

#### 3.4 Metald Installation

```bash
# Download VM assets (required for first run)
cd metald
sudo ./scripts/setup-vm-assets.sh
cd ..

# Build and install
make -C metald install

# The jailer configuration from Step 3 is automatically applied
sudo systemctl daemon-reload

# Start the service
make -C metald service-start

# Verify it's running
make -C metald service-status

# Verify jailer is working:
sudo journalctl -u metald | grep -i jailer
```

### Step 4: Verify mTLS

All services are configured to use SPIFFE/mTLS by default. Verify it's working:

```bash
# Verify services are using mTLS (check logs)
for service in assetmanagerd billaged builderd metald; do
    echo "=== $service logs ==="
    sudo journalctl -u $service -n 20 --no-pager | grep -i "tls\|spiffe"
done

# Check SPIFFE workload API
sudo /opt/spire/bin/spire-agent healthcheck -socketPath /run/spire/sockets/agent.sock

# Verify SPIFFE SVIDs are being issued
sudo /opt/spire/bin/spire-server entry list -socketPath /run/spire/server.sock
```

## Verification and Testing

### Step 5: Verify All Services Are Running

```bash
# Check all service statuses at once
for service in spire-server spire-agent assetmanagerd billaged builderd metald; do
    echo "=== $service ==="
    sudo systemctl is-active $service && echo "✓ Running" || echo "✗ Not running"
done

# Check all service ports
echo "=== Service Ports ==="
ss -tlnp | grep -E ':(8080|8081|8082|8083|8084|8085|9464|9465|9466|9467)'
```

### Step 6: Test Service Endpoints

```bash
# Test assetmanagerd
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/ListAssets \
  -H "Content-Type: application/json" \
  -d '{}' | jq .

# Test service health endpoints (all services now have unified health response)
curl -s http://localhost:8080/health | jq .  # metald
curl -s http://localhost:8081/health | jq .  # billaged
curl -s http://localhost:8082/health | jq .  # builderd
curl -s http://localhost:8083/health | jq .  # assetmanagerd

# Expected response format for all services:
# {
#   "status": "healthy",
#   "service": "<service-name>",
#   "version": "0.1.0",
#   "uptime_seconds": 123.456
# }

# Test SPIRE health endpoints
curl -s http://localhost:8085/live && echo " SPIRE server is healthy"
curl -s http://localhost:8084/live && echo " SPIRE agent is healthy"
```

### Step 7: Verify mTLS

```bash
# Check SPIFFE SVIDs are being issued
sudo /opt/spire/bin/spire-server entry list -socketPath /run/spire/server.sock

# Test agent can fetch SVIDs
sudo /opt/spire/bin/spire-agent api fetch x509 \
  -socketPath /run/spire/sockets/agent.sock \
  -write /tmp/

# View the fetched certificates
ls -la /tmp/svid.*.pem /tmp/bundle.*.pem
```

## Service Management Commands

### Managing Individual Services

Each service supports standard systemd management commands through Make:

```bash
# Service management pattern
make -C <service-name> service-start    # Start service
make -C <service-name> service-stop     # Stop service
make -C <service-name> service-restart  # Restart service
make -C <service-name> service-status   # Check status
make -C <service-name> service-logs     # Follow logs

# Examples
make -C metald service-restart
make -C billaged service-logs
make -C assetmanagerd service-status
```

### Managing All Services

```bash
# Start all core services in order
for service in assetmanagerd billaged builderd metald; do
    make -C $service service-start
    sleep 2  # Allow service to initialize
done

# Stop all services in reverse order
for service in metald builderd billaged assetmanagerd; do
    make -C $service service-stop
done

# Check all service statuses
for service in assetmanagerd billaged builderd metald; do
    echo "=== $service ==="
    make -C $service service-status
done
```

### SPIRE Management

```bash
# SPIRE server commands
make -C spire service-start-server
make -C spire service-stop-server
make -C spire service-logs-server

# SPIRE agent commands
make -C spire service-start-agent
make -C spire service-stop-agent
make -C spire service-logs-agent

# View both SPIRE logs
make -C spire service-logs
```

### Observability Stack Management

```bash
# Start observability stack
make o11y

# Stop observability stack
make o11y-stop

# View logs
make o11y-logs
```

## Uninstallation

### Quick Complete Cleanup

For a complete system cleanup with confirmation:

```bash
# Complete cleanup - removes everything including data
make clean-all
```

For immediate cleanup without confirmation:

```bash
# Force complete cleanup
make clean-all-force
```

### Selective Cleanup Options

For more control over what gets removed:

```bash
# Basic cleanup - preserves data and users
./scripts/cleanup-system.sh

# Remove data but preserve users
./scripts/cleanup-system.sh --remove-data

# Remove users but preserve data
./scripts/cleanup-system.sh --remove-users

# Complete cleanup without confirmation
./scripts/cleanup-system.sh --remove-data --remove-users --force
```

### Manual Step-by-Step Uninstallation

If you prefer manual control:

```bash
# 1. Stop all services
make -C metald service-stop
make -C builderd service-stop
make -C billaged service-stop
make -C assetmanagerd service-stop
make -C spire service-stop-agent
make -C spire service-stop-server
make o11y-stop

# 2. Uninstall services (removes binaries and systemd files)
make -C metald uninstall
make -C builderd uninstall
make -C billaged uninstall
make -C assetmanagerd uninstall
make -C spire uninstall

# 3. Optional: Remove users
sudo userdel metald
sudo userdel billaged
sudo userdel builderd
sudo userdel assetmanagerd
sudo userdel spire-server
sudo userdel spire-agent

# 4. Optional: Remove data
sudo rm -rf /opt/metald /opt/billaged /opt/builderd
sudo rm -rf /opt/assetmanagerd /opt/vm-assets
sudo rm -rf /opt/spire /var/lib/spire /etc/spire /run/spire

# 5. Reload systemd
sudo systemctl daemon-reload
```

## Troubleshooting Guide

### Common Issues and Solutions

#### 1. SPIRE Server Won't Start

**Symptom**: `spire-server.service` fails with timeout or socket errors

**Solutions**:
```bash
# Check if port 8085 is in use (SPIRE server moved from 8081 to avoid billaged conflict)
ss -tlnp | grep :8085

# Check server logs
sudo journalctl -u spire-server -n 50 --no-pager

# Verify socket permissions
ls -la /run/spire/

# Restart with debug logging
sudo systemctl stop spire-server
sudo /opt/spire/bin/spire-server run -config /etc/spire/server/server.conf
```

#### 2. SPIRE Agent Registration Fails

**Symptom**: Agent crashes with "join token was not provided" or "no identity issued"

**Solution**:
```bash
# Use the automated registration script
cd spire
./scripts/register-agent.sh

# The script will:
# - Detect if agent is already running
# - Generate a new join token if needed
# - Configure systemd with the token
# - Start/restart the agent
# - Clean up sensitive data
# - Verify registration success
```

**If registration still fails**:
```bash
# Check agent logs
sudo journalctl -u spire-agent -n 50 --no-pager

# Verify server has the node entry
sudo /opt/spire/bin/spire-server entry show -socketPath /run/spire/server.sock | grep node1

# Clear agent data and retry
sudo systemctl stop spire-agent
sudo rm -rf /var/lib/spire/agent/data/*
./scripts/register-agent.sh
```

#### 3. Service Port Conflicts

**Symptom**: Service fails to start with "address already in use"

**Solution**:
```bash
# Find what's using the port (example for port 8081)
sudo ss -tlnp | grep :8081

# Kill the process or change the service port
# Edit /etc/systemd/system/<service>.service
# Add: Environment="UNKEY_<SERVICE>_PORT=<new-port>"
sudo systemctl daemon-reload
sudo systemctl restart <service>
```

#### 4. mTLS Connection Failures

**Symptom**: Services can't connect (mTLS is required)

**Solutions**:
```bash
# Verify SPIFFE socket exists
ls -la /run/spire/sockets/agent.sock

# Check if services have SPIFFE IDs
sudo /opt/spire/bin/spire-server entry list -socketPath /run/spire/server.sock

# Re-register services if needed
make -C spire register-services

# Check service logs for TLS errors
sudo journalctl -u metald -n 50 | grep -i "tls\|spiffe\|x509"
```

#### 5. Build Failures

**Symptom**: `make install` fails with Go errors

**Solutions**:
```bash
# Check Go version (needs 1.24+)
go version

# Clean module cache
go clean -modcache

# Update dependencies
cd <service-directory>
go mod download
go mod tidy
```

#### 6. Firecracker Jailer Issues

**Symptom**: Metald fails with jailer permission errors

**Common Solutions**:

```bash
# Check jailer binary has correct capabilities
getcap /usr/local/bin/jailer
# Should show: /usr/local/bin/jailer cap_mknod,cap_sys_admin=ep

# Verify cgroups are set up correctly
ls -la /sys/fs/cgroup/cpu/firecracker/  # For cgroup v1
ls -la /sys/fs/cgroup/firecracker/       # For cgroup v2

# Check if metald user exists and matches configuration
id metald
grep JAILER /etc/systemd/system/metald.service.d/jailer.conf

# Re-run setup scripts if needed
sudo ./scripts/configure-jailer-user.sh
sudo ./scripts/setup-firecracker-cgroups.sh
sudo ./scripts/install-sudoers.sh

# Check sudoers configuration
sudo -l -U metald | grep jailer
```

**Specific Error Solutions**:

- **"Failed to chmod api.socket"**: UID/GID mismatch - ensure jailer UID/GID match the metald user
- **"Cannot create cgroups"**: Run `setup-firecracker-cgroups.sh` as root
- **"operation not permitted"**: Jailer missing capabilities - run `install-sudoers.sh`

### Debug Commands Reference

```bash
# Check all service statuses quickly
systemctl status spire-server spire-agent assetmanagerd billaged builderd metald --no-pager

# View recent logs for all services
journalctl -u spire-server -u spire-agent -u assetmanagerd -u billaged -u builderd -u metald --since "10 minutes ago"

# Test service connectivity
for port in 8080 8081 8082 8083 8084 8085; do
    echo -n "Port $port: "
    curl -s -o /dev/null -w "%{http_code}" http://localhost:$port/health || echo "No response"
done

# Check SPIFFE workload API
sudo /opt/spire/bin/spire-agent healthcheck -socketPath /run/spire/sockets/agent.sock
```

## Configuration Reference

### Service Ports

| Service | API Port | Metrics Port | Health Endpoint |
|---------|----------|--------------|-----------------|
| metald | 8080 | 9464 | `/health` |
| billaged | 8081 | 9465 | `/health` |
| builderd | 8082 | 9466 | `/health` |
| assetmanagerd | 8083 | 9467 | `/health` |
| SPIRE Server | 8085 | N/A | `/live` (8085) |
| SPIRE Agent | N/A | N/A | `/live` (8084) |

### Key Environment Variables

**TLS/SPIFFE Configuration**:
- `UNKEY_<SERVICE>_TLS_MODE`: `disabled`, `file`, or `spiffe`
- `UNKEY_<SERVICE>_SPIFFE_SOCKET`: Path to SPIFFE socket (default: `/run/spire/sockets/agent.sock`)

**Service Configuration**:
- `UNKEY_<SERVICE>_PORT`: API listen port
- `UNKEY_<SERVICE>_LOG_LEVEL`: `debug`, `info`, `warn`, `error`
- `UNKEY_<SERVICE>_OTEL_ENABLED`: `true` or `false`
- `UNKEY_<SERVICE>_OTEL_ENDPOINT`: OTLP endpoint (default: `localhost:4317`)

## Next Steps

1. **Enable Monitoring**: Access Grafana at http://localhost:3000 to view service metrics and traces
2. **Configure Production Settings**: Review and adjust service configurations in `/etc/systemd/system/`
3. **Set Up Backups**: Create backup procedures for `/opt/` service directories
4. **Review Security**: Ensure firewall rules and SELinux policies are appropriate
5. **Documentation**: Read service-specific docs in each service directory

For production deployment, refer to:
- `spire/DEPLOYMENT_PLAN.md` - Production SPIRE setup
- `spire/AUTOMATED_REGISTRATION.md` - Automated SPIRE registration details
- `spire/OPT_IN_ROLLOUT.md` - SPIFFE opt-in migration strategy
- `pkg/tls/PERFORMANCE.md` - TLS performance tuning
- `PORTS.md` - Complete port allocation reference
- `metald/docs/jailer-requirements.md` - Complete Firecracker jailer setup
- `metald/docs/jailer-integration.md` - Production jailer configuration
- Individual service CLAUDE.md files for service-specific guidance

### Production Security Requirements

The following security features are **REQUIRED** and automatically enabled:

1. **Firecracker Jailer**: Provides VM isolation through:
   - Minimal chroot environment for each VM
   - Privilege dropping after setup
   - Resource limits through cgroups
   - Network namespace isolation

2. **SPIFFE/mTLS**: Ensures secure inter-service communication through:
   - Automatic certificate rotation
   - Service identity verification
   - Encrypted communication channels
   - Zero-trust networking

To verify both are active:
```bash
# Check jailer processes
ps aux | grep jailer  # Should show jailer processes for each VM

# Check mTLS
sudo journalctl -u metald -u billaged -u assetmanagerd -u builderd | grep -i "tls mode: spiffe"
```