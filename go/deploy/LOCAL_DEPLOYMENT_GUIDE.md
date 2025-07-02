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

<details>
<summary><b>For Fedora</b></summary>

```bash
# Install development tools
sudo dnf group install -y development-tools
sudo dnf install -y git make golang curl wget iptables-legacy

# Install buf for protobuf generation
sudo ./scripts/install-buf.sh
```

#### Install Docker (Official Method)

Follow the official Docker installation for Fedora:

```bash
# Remove old versions
sudo dnf remove docker \
                docker-client \
                docker-client-latest \
                docker-common \
                docker-latest \
                docker-latest-logrotate \
                docker-logrotate \
                docker-selinux \
                docker-engine-selinux \
                docker-engine

# Set up the Docker repository
sudo dnf -y install dnf-plugins-core
sudo dnf config-manager addrepo --from-repofile=https://download.docker.com/linux/fedora/docker-ce.repo

# Install Docker Engine
sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Start and enable Docker
sudo systemctl start docker
sudo systemctl enable docker

# Add your user to the docker group
sudo usermod -aG docker $USER

# Verify installation
sudo docker run hello-world
```

#### Complete Setup

```bash
# Install KVM support
sudo dnf install -y qemu-kvm
sudo usermod -aG kvm $USER

# Install Firecracker with jailer (required for metald)
sudo ./scripts/install-firecracker.sh

# Log out and back in for group changes to take effect
```

</details>

### 3. Firecracker Setup (REQUIRED)

Metald now uses an integrated jailer approach that handles VM isolation automatically:

```bash
# Install Firecracker with jailer (required for metald)
sudo ./scripts/install-firecracker.sh

# The metald service now includes integrated jailer functionality
# No separate jailer binary or manual cgroup setup is required
```

**Notes**:
- Metald v0.2.0+ includes integrated jailer functionality
- No manual jailer user or cgroup configuration needed
- The system automatically handles VM isolation and security

<details>
<summary><b>For Ubuntu</b></summary>

```bash
# Install development tools
sudo apt update
sudo apt install -y build-essential git make golang curl wget

# Install buf for protobuf generation
sudo ./scripts/install-buf.sh
```

#### Install Docker (Official Method)

Follow the official Docker installation for Ubuntu:

```bash
# Remove old versions
for pkg in docker.io docker-doc docker-compose docker-compose-v2 podman-docker containerd runc; do
  sudo apt-get remove $pkg;
done

# Update package index and install prerequisites
sudo apt-get update
sudo apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release

# Add Docker's official GPG key
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# Set up the repository
echo \
  "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install Docker Engine
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Add your user to the docker group
sudo usermod -aG docker $USER

# Verify installation
sudo docker run hello-world
```

#### Complete Setup

```bash
# Install KVM support
sudo apt install -y qemu-kvm
sudo usermod -aG kvm $USER

# Install Firecracker with jailer (required for metald)
sudo ./scripts/install-firecracker.sh

# Log out and back in for group changes to take effect
```

</details>

## Service Installation Order

Services must be installed in this specific order due to dependencies:

1. **Observability Stack** (optional but recommended) - Grafana, Loki, Tempo, Mimir
2. **SPIRE** (REQUIRED) - Service identity and mTLS for secure inter-service communication
3. **assetmanagerd** - VM asset management
4. **billaged** - Usage billing service
5. **builderd** - Container build service
6. **metald** - VM management service (depends on assetmanagerd and billaged)

## Quick Installation

Run from the `/path/to/unkey/go/deploy` directory:

### Step 1: Observability Stack

```bash
# Start observability stack
make o11y
```

Access Grafana at `http://localhost:3000` (admin/admin)

### Step 2: SPIRE Installation and Setup

```bash
# Install and start SPIRE with all services registered
make spire-install
make spire-start
```

### Step 3: Install Services

```bash
# Install all services
make install

# Verify all services are running
make service-status
```

### Step 4: Verify Installation

```bash
# Check all services are healthy
for port in 8080 8081 8082 8083; do
    curl -s http://localhost:$port/health | jq .
done
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
#   "version": "0.x.x",
#   "uptime_seconds": 123.456
# }

# Test SPIRE health endpoints
curl -s http://localhost:8085/live && echo " SPIRE server is healthy"
curl -s http://localhost:8084/live && echo " SPIRE agent is healthy"
```

### Step 7: Verify mTLS

```bash
# Check SPIFFE SVIDs are being issued
sudo /opt/spire/bin/spire-server entry list -socketPath /var/lib/spire/server/server.sock

# Test agent can fetch SVIDs
sudo /opt/spire/bin/spire-agent api fetch x509 \
  -socketPath /var/lib/spire/agent/agent.sock \
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
make -C <service-name> service-logs     # Follow logs (or service-logs-tail for some services)

# Examples
make -C metald service-restart
make -C billaged service-logs-tail
make -C assetmanagerd service-status
make -C builderd service-logs
```

### Managing All Services

```bash
# Use top-level commands for all services
make service-start     # Start all services
make service-stop      # Stop all services
make service-status    # Check all service status
```

### SPIRE Management

```bash
# Essential SPIRE commands
make spire-start       # Start SPIRE and register services
make spire-stop        # Stop SPIRE services
make spire-status      # Check SPIRE status
```

### Observability Stack Management

```bash
make o11y              # Start observability stack
make o11y-stop         # Stop observability stack
```

## Uninstallation

```bash
# Complete cleanup - removes everything including data
make clean-all
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
ls -la /var/lib/spire/

# Restart with debug logging
sudo systemctl stop spire-server
sudo /opt/spire/bin/spire-server run -config /etc/spire/server/server.conf
```

#### 2. SPIRE Agent Registration Fails

**Symptom**: Agent crashes with "join token was not provided" or "no identity issued"

**Solution**:
```bash
# Use the automated registration via make
make -C spire register-agent

# The command will:
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
sudo /opt/spire/bin/spire-server entry show -socketPath /var/lib/spire/server/server.sock | grep node1

# Clear agent data and retry
sudo systemctl stop spire-agent
sudo rm -rf /var/lib/spire/agent/data/*
make -C spire register-agent
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
ls -la /var/lib/spire/agent/agent.sock

# Check if services have SPIFFE IDs
sudo /opt/spire/bin/spire-server entry list -socketPath /var/lib/spire/server/server.sock

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
# Check jailer binary exists
which jailer

# Check if metald user exists
id metald

# Check metald service configuration
sudo systemctl status metald

# Re-install metald if jailer issues persist
make -C metald uninstall
make -C metald install
```

**Specific Error Solutions**:

- **"Failed to chmod api.socket"**: UID/GID mismatch - ensure jailer UID/GID match the metald user
- **"Cannot create cgroups"**: Ensure metald is running as root (check systemd service)
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
sudo /opt/spire/bin/spire-agent healthcheck -socketPath /var/lib/spire/agent/agent.sock
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
- `UNKEY_<SERVICE>_TLS_MODE`: `file` or `spiffe` (default: `spiffe`)
- `UNKEY_<SERVICE>_SPIFFE_SOCKET`: Path to SPIFFE socket (default: `/var/lib/spire/agent/agent.sock`)

**SPIRE Configuration**:
- `UNKEY_SPIRE_TRUST_DOMAIN`: Trust domain (e.g., `development.unkey.app`)
- `UNKEY_SPIRE_SERVER_URL`: Server URL for agents (default: `https://localhost:8085`)
- `UNKEY_SPIRE_JOIN_TOKEN`: 1-year join token for development auto-join

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
- `pkg/tls/PERFORMANCE.md` - TLS performance tuning
- `PORTS.md` - Complete port allocation reference
- `spire/docs/README.md` - SPIRE architecture and deployment guide
- Individual service README.md files for service-specific guidance

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
   - Encrypted HTTPS communication
   - Zero-trust networking

3. **Enhanced SPIRE Security**: 
   - Agents communicate with servers over HTTPS (not Unix sockets)
   - TLS mode defaults to `spiffe` (disabled mode deprecated)
   - Node attestation support for production environments
   - Environment-specific trust domains

To verify security features are active:
```bash
# Check jailer processes
ps aux | grep jailer  # Should show jailer processes for each VM

# Check mTLS is enabled
sudo journalctl -u metald -u billaged -u assetmanagerd -u builderd | grep -i "tls mode: spiffe"

# Verify agent HTTPS communication
sudo journalctl -u spire-agent | grep -i "server_address.*https"
```
