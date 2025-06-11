# Metald Installation Guide for Fedora 42

This guide covers secure installation and configuration of metald on Fedora 42 systems.

## Prerequisites

### System Requirements

- Fedora 42 with systemd
- Go 1.21+ (for building from source)
- Root or sudo access for installation
- At least 4GB RAM and 2 CPU cores for VM workloads

### Required Packages

```bash
# Update system
sudo dnf update -y

# Install development tools and dependencies
sudo dnf install -y \
    golang \
    git \
    make \
    curl \
    jq \
    systemd-devel \
    cgroup-tools \
    iptables \
    bridge-utils

# Install Firecracker (if using Firecracker backend)
# Download latest release from https://github.com/firecracker-microvm/firecracker/releases
sudo curl -L https://github.com/firecracker-microvm/firecracker/releases/latest/download/firecracker-v1.5.1-x86_64.tgz \
    -o /tmp/firecracker.tgz
sudo tar -xzf /tmp/firecracker.tgz -C /tmp
sudo cp /tmp/release-v1.5.1-x86_64/firecracker-v1.5.1-x86_64 /usr/bin/firecracker
sudo cp /tmp/release-v1.5.1-x86_64/jailer-v1.5.1-x86_64 /usr/bin/jailer
sudo chmod +x /usr/bin/firecracker /usr/bin/jailer

# Verify Firecracker installation
firecracker --version
jailer --version
```

## Security Setup

### 1. Create Dedicated System User

```bash
# Create metald system user with restricted permissions
sudo useradd -r -s /bin/false -d /opt/metald -c "Metald VM Management Service" metald

# Verify user creation
id metald
# Should show: uid=995(metald) gid=993(metald) groups=993(metald)
```

### 2. Set Up Directory Structure

```bash
# Create application directories
sudo mkdir -p /opt/metald
sudo mkdir -p /var/log/metald
sudo mkdir -p /etc/metald

# Create runtime directories
sudo mkdir -p /tmp/github.com/unkeyed/unkey/go/deploy/metald/sockets
sudo mkdir -p /tmp/github.com/unkeyed/unkey/go/deploy/metald/logs

# Create jailer chroot directory (for production)
sudo mkdir -p /srv/jailer

# Set ownership
sudo chown -R metald:metald /opt/metald
sudo chown -R metald:metald /var/log/metald
sudo chown -R metald:metald /etc/metald
sudo chown -R metald:metald /tmp/github.com/unkeyed/unkey/go/deploy/metald
sudo chown -R metald:metald /srv/jailer

# Set permissions
sudo chmod 755 /opt/metald
sudo chmod 750 /var/log/metald
sudo chmod 750 /etc/metald
sudo chmod 755 /srv/jailer
```

### 3. Configure Cgroups (Required for Resource Limits)

```bash
# Ensure cgroups v1 is available (required by Firecracker jailer)
sudo mkdir -p /sys/fs/cgroup/metald

# Add metald user to systemd-journal group for logging
sudo usermod -a -G systemd-journal metald
```

### 4. Configure Firewall

```bash
# Configure firewalld for metald services
sudo firewall-cmd --permanent --new-service=metald
sudo firewall-cmd --permanent --service=metald --set-description="Metald VM Management Service"
sudo firewall-cmd --permanent --service=metald --set-short="Metald"
sudo firewall-cmd --permanent --service=metald --add-port=8080/tcp
sudo firewall-cmd --permanent --service=metald --add-port=9464/tcp

# Enable the service
sudo firewall-cmd --permanent --add-service=metald
sudo firewall-cmd --reload

# Verify firewall configuration
sudo firewall-cmd --list-services | grep metald
sudo firewall-cmd --list-ports
```

## Installation Methods

### Method 1: Using Makefile (Recommended)

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey.git
cd unkey/go/deploy/metald

# Build and install
make install

# Enable and start the service
make service-install
make service-start

# Check status
make service-status
```

### Method 2: Manual Installation

```bash
# Build metald
go build -ldflags "-s -w" -o build/metald ./cmd/api

# Install binary
sudo cp build/metald /usr/local/bin/metald
sudo chmod +x /usr/local/bin/metald

# Install systemd service
sudo cp metald.service /etc/systemd/system/metald.service
sudo systemctl daemon-reload
sudo systemctl enable metald
sudo systemctl start metald
```

## Configuration

### Environment Variables

Create a configuration file for environment variables:

```bash
# Create environment file
sudo tee /etc/metald/metald.env > /dev/null <<EOF
# Metald Configuration
UNKEY_METALD_BACKEND=firecracker
UNKEY_METALD_PORT=8080
UNKEY_METALD_ADDRESS=0.0.0.0

# OpenTelemetry Configuration
UNKEY_METALD_OTEL_ENABLED=true
UNKEY_METALD_OTEL_SERVICE_NAME=metald
UNKEY_METALD_OTEL_ENDPOINT=localhost:4318
UNKEY_METALD_OTEL_PROMETHEUS_PORT=9464

# Jailer Configuration (Production)
UNKEY_METALD_JAILER_ENABLED=false
UNKEY_METALD_JAILER_BINARY=/usr/bin/jailer
UNKEY_METALD_FIRECRACKER_BINARY=/usr/bin/firecracker
UNKEY_METALD_JAILER_UID=1000
UNKEY_METALD_JAILER_GID=1000
UNKEY_METALD_JAILER_CHROOT_DIR=/srv/jailer
UNKEY_METALD_JAILER_NETNS=true
UNKEY_METALD_JAILER_PIDNS=true
UNKEY_METALD_JAILER_MEMORY_LIMIT=134217728
UNKEY_METALD_JAILER_CPU_QUOTA=100
UNKEY_METALD_JAILER_FD_LIMIT=1024
EOF

# Set secure permissions
sudo chown root:metald /etc/metald/metald.env
sudo chmod 640 /etc/metald/metald.env
```

### Update Systemd Service to Use Environment File

```bash
# Update the service file to use environment file
sudo tee -a /etc/systemd/system/metald.service > /dev/null <<EOF

# Load environment from file
EnvironmentFile=-/etc/metald/metald.env
EOF

# Reload systemd
sudo systemctl daemon-reload
sudo systemctl restart metald
```

## Security Hardening

### 1. SELinux Configuration (Fedora Default)

```bash
# Check SELinux status
sestatus

# If SELinux is enforcing, create a custom policy (basic example)
# This is a starting point - production deployments should have proper SELinux policies
sudo setsebool -P httpd_can_network_connect 1
sudo setsebool -P domain_can_mmap_files 1

# For development, you might temporarily set permissive mode for metald
# WARNING: Only for development - never in production
# sudo semanage permissive -a metald_t
```

### 2. Systemd Security Features

The provided `metald.service` includes comprehensive security hardening:

- **NoNewPrivileges=true**: Prevents privilege escalation
- **PrivateTmp=true**: Isolated /tmp directory
- **ProtectSystem=strict**: Read-only filesystem protection
- **ProtectHome=true**: Home directory protection
- **RestrictNamespaces=true**: Namespace restrictions
- **MemoryDenyWriteExecute=false**: Allows JIT (needed for Go runtime)
- **SystemCallFilter**: Restricts dangerous system calls

### 3. File Permissions Audit

```bash
# Verify secure permissions
ls -la /usr/local/bin/metald
# Should show: -rwxr-xr-x 1 root root

ls -la /etc/systemd/system/metald.service
# Should show: -rw-r--r-- 1 root root

ls -ld /opt/metald
# Should show: drwxr-xr-x 2 metald metald

ls -ld /var/log/metald
# Should show: drwxr-x--- 2 metald metald
```

## Production Jailer Setup

For production deployments with enhanced security:

### 1. Enable Jailer

```bash
# Update environment configuration
sudo sed -i 's/UNKEY_METALD_JAILER_ENABLED=false/UNKEY_METALD_JAILER_ENABLED=true/' /etc/metald/metald.env

# Create jailer user for isolation
sudo useradd -r -s /bin/false -d /srv/jailer -c "Firecracker Jailer User" jailer

# Set up jailer directories with proper permissions
sudo mkdir -p /srv/jailer
sudo chown jailer:jailer /srv/jailer
sudo chmod 755 /srv/jailer

# Verify jailer binary permissions
sudo chown root:root /usr/bin/jailer /usr/bin/firecracker
sudo chmod 755 /usr/bin/jailer /usr/bin/firecracker

# Restart service with jailer enabled
sudo systemctl restart metald
```

### 2. Network Namespace Setup (Optional)

```bash
# Install additional networking tools for namespace isolation
sudo dnf install -y iproute bridge-utils

# Create network namespaces for VM isolation
# This will be handled automatically by jailer when NETNS=true
```

## Monitoring and Logging

### 1. Log Configuration

```bash
# Configure log rotation
sudo tee /etc/logrotate.d/metald > /dev/null <<EOF
/var/log/metald/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 metald metald
    postrotate
        systemctl reload metald
    endscript
}
EOF
```

### 2. Monitoring Setup

```bash
# View real-time logs
sudo journalctl -u metald -f

# Check service status
sudo systemctl status metald

# Monitor metrics (if OpenTelemetry is enabled)
curl -s http://localhost:9464/metrics | grep vm_

# Check health endpoint
curl -s http://localhost:8080/_/health | jq .
```

## Troubleshooting

### Common Issues

1. **Permission Denied Errors**
   ```bash
   # Check file permissions
   sudo ls -la /usr/local/bin/metald
   sudo ls -la /etc/systemd/system/metald.service
   
   # Fix permissions if needed
   sudo chown root:root /usr/local/bin/metald
   sudo chmod 755 /usr/local/bin/metald
   ```

2. **Service Won't Start**
   ```bash
   # Check detailed service status
   sudo systemctl status metald -l
   
   # Check journal logs
   sudo journalctl -u metald --since "5 minutes ago"
   
   # Test binary directly
   sudo -u metald /usr/local/bin/metald
   ```

3. **Firewall Issues**
   ```bash
   # Check if ports are open
   sudo ss -tlnp | grep -E ":8080|:9464"
   
   # Verify firewall rules
   sudo firewall-cmd --list-all
   ```

4. **Jailer Issues**
   ```bash
   # Check jailer permissions
   ls -la /usr/bin/jailer /usr/bin/firecracker
   
   # Verify chroot directory
   ls -la /srv/jailer
   
   # Test jailer directly
   sudo jailer --help
   ```

### Log Analysis

```bash
# Check for common error patterns
sudo journalctl -u metald | grep -i error
sudo journalctl -u metald | grep -i failed
sudo journalctl -u metald | grep -i denied

# Monitor resource usage
sudo systemctl show metald --property=MemoryCurrent
sudo systemctl show metald --property=CPUUsageNSec
```

## Maintenance

### Updates

```bash
# Update metald (development)
make dev-install

# Update metald (production with jailer)
make prod-install

# Check service status after update
make service-status
```

### Backup Configuration

```bash
# Backup configuration files
sudo tar -czf /root/metald-backup-$(date +%Y%m%d).tar.gz \
    /etc/systemd/system/metald.service \
    /etc/metald/ \
    /usr/local/bin/metald

# Verify backup
sudo tar -tzf /root/metald-backup-$(date +%Y%m%d).tar.gz
```

### Security Auditing

```bash
# Check for security updates
sudo dnf check-update | grep -E "(firecracker|systemd|kernel)"

# Audit systemd security settings
sudo systemd-analyze security metald

# Check file integrity
sudo rpm -V firecracker || echo "Firecracker not from RPM"
sudo sha256sum /usr/local/bin/metald
```

This comprehensive guide ensures metald is installed securely on Fedora 42 with proper user isolation, systemd hardening, and production-ready configuration options.