# Operations Manual

This document provides comprehensive guidance for deploying, configuring, monitoring, and maintaining metald in production environments.

## Installation

### System Requirements

#### Operating System
- **Linux**: Ubuntu 20.04+, CentOS 8+, or RHEL 8+
- **Kernel**: 4.14+ with KVM support
- **Systemd**: Required for service management
- **Root Access**: Required for network operations and VM management

#### Hardware Requirements

| Component | Minimum | Recommended | Production |
|-----------|---------|-------------|------------|
| CPU | 4 cores | 8 cores | 16+ cores |
| Memory | 8GB | 16GB | 32GB+ |
| Storage | 50GB | 100GB | 500GB+ SSD |
| Network | 1Gbps | 10Gbps | 25Gbps+ |

#### Software Dependencies

**Required**:
- `firecracker` binary (v1.0.0+) - [Installation Guide](../../../scripts/install-firecracker.sh)
- `SQLite3` development libraries
- `iproute2` for network management
- `iptables` for firewall rules

**Optional**:
- Docker (for builderd integration)
- SPIRE agent (for production mTLS)
- Prometheus (for metrics collection)

### Installation Methods

#### From Source

```bash
# Clone repository
git clone https://github.com/unkeyed/unkey
cd go/deploy/metald

# Build binary
make build

# Install systemd service (requires root)
sudo make install

# Verify installation
sudo systemctl status metald
```

#### Using Package Manager

```bash
# Ubuntu/Debian (when available)
sudo apt install metald

# RHEL/CentOS (when available)
sudo yum install metald
```

### Initial Setup

#### Directory Structure

```bash
# Create required directories
sudo mkdir -p /opt/metald/{data,assets,logs}
sudo mkdir -p /srv/jailer
sudo mkdir -p /etc/metald

# Set ownership
sudo chown -R metald:metald /opt/metald
sudo chown -R firecracker:firecracker /srv/jailer
```

#### Firecracker Installation

```bash
# Download and install Firecracker
curl -fsSL https://github.com/firecracker-microvm/firecracker/releases/download/v1.4.1/firecracker-v1.4.1-x86_64.tgz | tar -xz
sudo mv firecracker-v1.4.1-x86_64 /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker

# Verify installation
firecracker --version
```

## Configuration

### Environment Variables

#### Core Service Configuration

**Server Settings**:
```bash
export UNKEY_METALD_PORT=8080                    # API server port
export UNKEY_METALD_ADDRESS=0.0.0.0              # Bind address (0.0.0.0 for all interfaces)
export UNKEY_METALD_BACKEND=firecracker          # Hypervisor backend
```

**Data Storage**:
```bash
export UNKEY_METALD_DATA_DIR=/opt/metald/data     # SQLite database location
```

#### Security Configuration

**mTLS/SPIFFE Settings** (Production):
```bash
export UNKEY_METALD_TLS_MODE=spiffe               # Enable SPIFFE authentication
export UNKEY_METALD_SPIFFE_SOCKET=/var/lib/spire/agent/agent.sock
export UNKEY_METALD_TLS_ENABLE_CERT_CACHING=true
export UNKEY_METALD_TLS_CERT_CACHE_TTL=5s
```

**Development TLS Settings**:
```bash
export UNKEY_METALD_TLS_MODE=disabled             # Disable TLS for development
# OR for file-based TLS:
export UNKEY_METALD_TLS_MODE=file
export UNKEY_METALD_TLS_CERT_FILE=/etc/ssl/certs/metald.crt
export UNKEY_METALD_TLS_KEY_FILE=/etc/ssl/private/metald.key
export UNKEY_METALD_TLS_CA_FILE=/etc/ssl/certs/ca.crt
```

#### VM Isolation Configuration

**Jailer Settings** (Critical for Security):
```bash
export UNKEY_METALD_JAILER_UID=1000               # User ID for VM isolation
export UNKEY_METALD_JAILER_GID=1000               # Group ID for VM isolation
export UNKEY_METALD_JAILER_CHROOT_DIR=/srv/jailer # Chroot base directory
```

#### Service Integration

**AssetManager Integration**:
```bash
export UNKEY_METALD_ASSETMANAGER_ENABLED=true     # Enable asset management
export UNKEY_METALD_ASSETMANAGER_ENDPOINT=https://assetmanagerd:8083
export UNKEY_METALD_ASSETMANAGER_CACHE_DIR=/opt/metald/assets
```

**Billing Integration**:
```bash
export UNKEY_METALD_BILLING_ENABLED=true          # Enable billing collection
export UNKEY_METALD_BILLING_ENDPOINT=https://billaged:8081
export UNKEY_METALD_BILLING_MOCK_MODE=false       # Use real billing client
```

#### Network Configuration

**IPv4 Network Settings**:
```bash
export UNKEY_METALD_NETWORK_ENABLED=true
export UNKEY_METALD_NETWORK_IPV4_ENABLED=true
export UNKEY_METALD_NETWORK_BRIDGE_IPV4=172.31.0.1/19      # Bridge IP
export UNKEY_METALD_NETWORK_VM_SUBNET_IPV4=172.31.0.0/19   # VM subnet
export UNKEY_METALD_NETWORK_DNS_IPV4=8.8.8.8,8.8.4.4      # DNS servers
export UNKEY_METALD_NETWORK_BRIDGE=br-vms                  # Bridge name
```

**IPv6 Network Settings** (Optional):
```bash
export UNKEY_METALD_NETWORK_IPV6_ENABLED=true
export UNKEY_METALD_NETWORK_BRIDGE_IPV6=fd00::1/64
export UNKEY_METALD_NETWORK_VM_SUBNET_IPV6=fd00::/64
export UNKEY_METALD_NETWORK_DNS_IPV6=2606:4700:4700::1111,2606:4700:4700::1001
export UNKEY_METALD_NETWORK_IPV6_MODE=dual-stack           # dual-stack, ipv6-only, ipv4-only
```

**Network Security**:
```bash
export UNKEY_METALD_NETWORK_HOST_PROTECTION=true          # Protect host routes
export UNKEY_METALD_NETWORK_PRIMARY_INTERFACE=eth0        # Primary interface to protect
export UNKEY_METALD_NETWORK_RATE_LIMIT=true               # Enable rate limiting
export UNKEY_METALD_NETWORK_RATE_LIMIT_MBPS=1000          # Rate limit in Mbps
```

#### Observability Configuration

**OpenTelemetry Settings**:
```bash
export UNKEY_METALD_OTEL_ENABLED=true                     # Enable OpenTelemetry
export UNKEY_METALD_OTEL_SERVICE_NAME=metald
export UNKEY_METALD_OTEL_SERVICE_VERSION=1.0.0
export UNKEY_METALD_OTEL_SAMPLING_RATE=1.0                # 100% sampling for development
export UNKEY_METALD_OTEL_ENDPOINT=localhost:4318          # OTLP endpoint
```

**Prometheus Settings**:
```bash
export UNKEY_METALD_OTEL_PROMETHEUS_ENABLED=true          # Enable Prometheus metrics
export UNKEY_METALD_OTEL_PROMETHEUS_PORT=9464             # Metrics port
export UNKEY_METALD_OTEL_PROMETHEUS_INTERFACE=127.0.0.1   # Bind to localhost only
export UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED=false   # Limit cardinality in production
```

### Configuration Files

#### Systemd Service Configuration

**File**: `/etc/systemd/system/metald.service`
```ini
[Unit]
Description=Metald VM Control Plane
Documentation=https://github.com/unkeyed/unkey/tree/main/go/deploy/metald
After=network.target spire-agent.service
Requires=network.target
Wants=spire-agent.service

[Service]
Type=exec
User=root
Group=metald
ExecStart=/usr/local/bin/metald
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=5s
TimeoutStartSec=30s
TimeoutStopSec=30s

# Security settings
NoNewPrivileges=false
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/metald /srv/jailer /var/log/metald

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

# Environment file
EnvironmentFile=-/etc/metald/environment

[Install]
WantedBy=multi-user.target
```

#### Environment File

**File**: `/etc/metald/environment`
```bash
# Core configuration
UNKEY_METALD_PORT=8080
UNKEY_METALD_ADDRESS=0.0.0.0
UNKEY_METALD_BACKEND=firecracker
UNKEY_METALD_DATA_DIR=/opt/metald/data

# Security
UNKEY_METALD_TLS_MODE=spiffe
UNKEY_METALD_JAILER_UID=1000
UNKEY_METALD_JAILER_GID=1000
UNKEY_METALD_JAILER_CHROOT_DIR=/srv/jailer

# Service integration
UNKEY_METALD_ASSETMANAGER_ENABLED=true
UNKEY_METALD_ASSETMANAGER_ENDPOINT=https://assetmanagerd:8083
UNKEY_METALD_BILLING_ENABLED=true
UNKEY_METALD_BILLING_ENDPOINT=https://billaged:8081

# Observability
UNKEY_METALD_OTEL_ENABLED=true
UNKEY_METALD_OTEL_PROMETHEUS_ENABLED=true
UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED=false
```

### Network Setup

#### Bridge Configuration

**Manual Bridge Setup**:
```bash
# Create bridge for VMs
sudo ip link add name br-vms type bridge
sudo ip addr add 172.31.0.1/19 dev br-vms
sudo ip link set br-vms up

# Enable IP forwarding
echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Configure NAT (replace eth0 with your primary interface)
sudo iptables -t nat -A POSTROUTING -s 172.31.0.0/19 -o eth0 -j MASQUERADE
sudo iptables -A FORWARD -i br-vms -o eth0 -j ACCEPT
sudo iptables -A FORWARD -i eth0 -o br-vms -m state --state RELATED,ESTABLISHED -j ACCEPT
```

**Persistent Network Configuration** (Ubuntu):

**File**: `/etc/netplan/60-metald-bridge.yaml`
```yaml
network:
  version: 2
  bridges:
    br-vms:
      addresses:
        - 172.31.0.1/19
      parameters:
        stp: false
        forward-delay: 0
```

#### Firewall Rules

**File**: `/etc/metald/firewall-rules.sh`
```bash
#!/bin/bash
# Metald firewall configuration

# VM bridge subnet
VM_SUBNET="172.31.0.0/19"
BRIDGE_NAME="br-vms"
PRIMARY_INTERFACE="eth0"  # Adjust for your system

# Enable forwarding
echo 1 > /proc/sys/net/ipv4/ip_forward

# NAT for VM traffic
iptables -t nat -A POSTROUTING -s $VM_SUBNET -o $PRIMARY_INTERFACE -j MASQUERADE

# Allow VM bridge traffic
iptables -A FORWARD -i $BRIDGE_NAME -o $PRIMARY_INTERFACE -j ACCEPT
iptables -A FORWARD -i $PRIMARY_INTERFACE -o $BRIDGE_NAME -m state --state RELATED,ESTABLISHED -j ACCEPT

# Allow bridge internal communication
iptables -A FORWARD -i $BRIDGE_NAME -o $BRIDGE_NAME -j ACCEPT

# Protect host from VM access (security)
iptables -A INPUT -i $BRIDGE_NAME -d 172.31.0.1 -p tcp --dport 22 -j DROP
iptables -A INPUT -i $BRIDGE_NAME -d 172.31.0.1 -p tcp --dport 80 -j DROP
iptables -A INPUT -i $BRIDGE_NAME -d 172.31.0.1 -p tcp --dport 443 -j DROP
```

## Service Management

### Systemd Operations

```bash
# Start service
sudo systemctl start metald

# Enable auto-start
sudo systemctl enable metald

# Check status
sudo systemctl status metald

# View logs
sudo journalctl -u metald -f

# Restart service
sudo systemctl restart metald

# Stop service
sudo systemctl stop metald

# Reload configuration
sudo systemctl reload metald
```

### Process Management

```bash
# Check metald process
ps aux | grep metald

# Check VM processes
ps aux | grep firecracker

# View network bridges
ip link show type bridge

# Check TAP devices
ip link show type tun

# View active VMs
curl -H "Authorization: Bearer <token>" http://localhost:8080/vmprovisioner.v1.VmService/ListVms
```

## Monitoring

### Health Checks

#### Service Health Endpoint

```bash
# Basic health check
curl http://localhost:8080/health

# Expected response:
{
  "status": "healthy",
  "service": "metald",
  "version": "1.0.0",
  "uptime": "2h15m30s",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

#### System Health Checks

**File**: `/etc/metald/health-check.sh`
```bash
#!/bin/bash
# Comprehensive metald health check

set -e

# Check service status
systemctl is-active --quiet metald || exit 1

# Check API responsiveness
curl -f -s http://localhost:8080/health > /dev/null || exit 1

# Check database accessibility
test -r /opt/metald/data/metald.db || exit 1

# Check bridge interface
ip link show br-vms > /dev/null || exit 1

# Check firecracker binary
test -x /usr/local/bin/firecracker || exit 1

echo "All health checks passed"
```

### Metrics Collection

#### Prometheus Configuration

**File**: `/etc/prometheus/metald.yml`
```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'metald'
    static_configs:
      - targets: ['localhost:9464']
    scrape_interval: 10s
    metrics_path: /metrics
    scheme: http
```

#### Key Metrics to Monitor

**VM Operations**:
- `metald_vm_operations_total{method, result, customer_id}` - Operation counts
- `metald_vm_operation_duration_seconds{method}` - Operation latency
- `metald_active_vms{state, customer_id}` - VM count by state

**System Health**:
- `metald_api_requests_total{method, code}` - API request counts
- `metald_backend_errors_total{backend, error_type}` - Backend error rates
- `metald_database_operations_total{operation, result}` - Database operation counts

**Resource Usage**:
- `metald_billing_metrics_collected_total{customer_id}` - Billing data collection
- `metald_network_allocations_total` - Network resource usage
- `metald_asset_operations_total{operation, result}` - Asset management operations

**Source**: [observability/metrics.go](../../internal/observability/metrics.go)

#### Alerting Rules

**File**: `/etc/prometheus/metald-alerts.yml`
```yaml
groups:
  - name: metald
    rules:
      # Service availability
      - alert: MetaldDown
        expr: up{job="metald"} == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Metald service is down"
          description: "Metald has been down for more than 30 seconds"

      # High error rate
      - alert: MetaldHighErrorRate
        expr: rate(metald_vm_operations_total{result="error"}[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High VM operation error rate"
          description: "VM operation error rate is {{ $value }} errors/sec"

      # Database issues
      - alert: MetaldDatabaseErrors
        expr: rate(metald_database_operations_total{result="error"}[5m]) > 0.05
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database operation errors"
          description: "Database error rate is {{ $value }} errors/sec"

      # Resource exhaustion
      - alert: MetaldHighVMCount
        expr: metald_active_vms > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High VM count"
          description: "{{ $value }} VMs are currently active"
```

### Logging

#### Log Configuration

**Structured JSON Logging**: [main.go:97](../../cmd/metald/main.go#L97)
- All logs output in JSON format
- Includes request IDs and customer context
- Multi-level logging (debug, info, warn, error)

#### Log Aggregation

**Fluent Bit Configuration**:

**File**: `/etc/fluent-bit/metald.conf`
```ini
[INPUT]
    Name systemd
    Tag metald
    Systemd_Filter _SYSTEMD_UNIT=metald.service
    Read_From_Tail On

[FILTER]
    Name parser
    Match metald
    Key_Name MESSAGE
    Parser json
    Reserve_Data On

[OUTPUT]
    Name es
    Match metald
    Host elasticsearch.local
    Port 9200
    Index metald-logs
    Type _doc
```

#### Important Log Events

**VM Lifecycle Events**:
```json
{"level":"info","msg":"vm created successfully","vm_id":"vm-123","customer_id":"cust-456","duration":"1.2s"}
{"level":"info","msg":"vm booted successfully","vm_id":"vm-123","duration":"3.4s"}
{"level":"warn","msg":"vm cleanup failed after all retries","vm_id":"vm-123","final_error":"context timeout"}
```

**Security Events**:
```json
{"level":"warn","msg":"SECURITY: customer_id mismatch in request","authenticated_customer":"cust-123","request_customer":"cust-456"}
{"level":"warn","msg":"token validation failed","procedure":"CreateVm","error":"invalid customer token"}
```

**Service Integration Events**:
```json
{"level":"error","msg":"assetmanager connection error","code":"UNAVAILABLE","vm_id":"vm-123","operation":"QueryAssets"}
{"level":"error","msg":"billaged connection error","code":"UNAVAILABLE","vm_id":"vm-123","operation":"SendMetricsBatch"}
```

## Performance Tuning

### System Optimization

#### Kernel Parameters

**File**: `/etc/sysctl.d/99-metald.conf`
```bash
# Network optimization
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 16384 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728

# VM networking
net.bridge.bridge-nf-call-iptables = 1
net.bridge.bridge-nf-call-ip6tables = 1

# File descriptor limits
fs.file-max = 1000000

# Virtual memory
vm.max_map_count = 262144
```

#### Resource Limits

**File**: `/etc/security/limits.d/metald.conf`
```bash
# Metald service limits
metald soft nofile 65536
metald hard nofile 65536
metald soft nproc 32768
metald hard nproc 32768

# Root limits (for network operations)
root soft nofile 65536
root hard nofile 65536
```

### Application Tuning

#### Database Optimization

**SQLite Tuning**: [database/database.go](../../internal/database/database.go)
```bash
# Check database performance
sqlite3 /opt/metald/data/metald.db "PRAGMA compile_options;"

# Analyze query performance
sqlite3 /opt/metald/data/metald.db "EXPLAIN QUERY PLAN SELECT * FROM vms WHERE customer_id = ?;"

# Database maintenance
sqlite3 /opt/metald/data/metald.db "VACUUM; ANALYZE;"
```

#### Memory Management

**Go Runtime Tuning**:
```bash
export GOGC=100              # GC target percentage
export GOMEMLIMIT=4GiB       # Memory limit
export GOMAXPROCS=8          # CPU limit
```

#### Concurrent Operations

**Configuration Tuning**:
```bash
# Increase concurrent VM operations (if hardware permits)
export UNKEY_METALD_MAX_CONCURRENT_VMS=50

# Network allocation pool size
export UNKEY_METALD_NETWORK_POOL_SIZE=1000

# Asset cache size
export UNKEY_METALD_ASSET_CACHE_SIZE=100
```

## Troubleshooting

### Common Issues

#### Service Won't Start

**Symptoms**: Service fails to start, exits immediately
**Diagnosis**:
```bash
# Check service status
sudo systemctl status metald

# View detailed logs
sudo journalctl -u metald --no-pager

# Check configuration
sudo -u metald /usr/local/bin/metald --help

# Validate environment
sudo -u metald env | grep UNKEY_METALD
```

**Common Causes**:
- Missing firecracker binary: `sudo which firecracker`
- Permission issues: `sudo chown -R metald:metald /opt/metald`
- Invalid configuration: Check environment variables
- Network conflicts: `ip addr show | grep 172.31`

#### VM Creation Failures

**Symptoms**: CreateVm API calls fail
**Diagnosis**:
```bash
# Check backend connectivity
curl -H "Authorization: Bearer <token>" http://localhost:8080/health

# Check asset manager connectivity
curl https://assetmanagerd:8083/health

# Check network configuration
ip link show br-vms
ip route show table all | grep 172.31

# Check jailer permissions
ls -la /srv/jailer
sudo -u firecracker touch /srv/jailer/test
```

**Common Causes**:
- Missing assets: Check assetmanagerd integration
- Network misconfiguration: Verify bridge setup
- Jailer permission issues: Check UID/GID settings
- Firecracker binary issues: Test manual execution

#### Network Connectivity Issues

**Symptoms**: VMs can't access network or internet
**Diagnosis**:
```bash
# Check bridge status
ip addr show br-vms

# Check NAT rules
sudo iptables -t nat -L -n -v

# Check forwarding
cat /proc/sys/net/ipv4/ip_forward

# Test VM connectivity (if VM is running)
ping 172.31.0.10  # Example VM IP
```

**Common Fixes**:
```bash
# Enable IP forwarding
sudo sysctl -w net.ipv4.ip_forward=1

# Add NAT rule
sudo iptables -t nat -A POSTROUTING -s 172.31.0.0/19 -o eth0 -j MASQUERADE

# Restart networking
sudo systemctl restart networking
```

#### Database Issues

**Symptoms**: VM state inconsistencies, database errors
**Diagnosis**:
```bash
# Check database file
ls -la /opt/metald/data/metald.db

# Check database integrity
sqlite3 /opt/metald/data/metald.db "PRAGMA integrity_check;"

# Check VM records
sqlite3 /opt/metald/data/metald.db "SELECT id, customer_id, state FROM vms;"

# Check reconciler logs
sudo journalctl -u metald | grep reconciler
```

#### High Memory Usage

**Symptoms**: Service consuming excessive memory
**Diagnosis**:
```bash
# Check memory usage
ps aux | grep metald
cat /proc/$(pidof metald)/status | grep -E 'VmRSS|VmSize'

# Check Go runtime stats
curl http://localhost:9464/metrics | grep go_memstats

# Check VM count
curl -H "Authorization: Bearer <token>" http://localhost:8080/vmprovisioner.v1.VmService/ListVms | jq '.vms | length'
```

### Debugging Tools

#### API Testing

```bash
# Test API endpoints
curl -H "Authorization: Bearer test-token" \
     -H "Content-Type: application/json" \
     -d '{}' \
     http://localhost:8080/vmprovisioner.v1.VmService/ListVms

# Test with metald-cli
cd client/cmd/metald-cli
go run main.go --server http://localhost:8080 --token test-token list-vms
```

#### Network Debugging

```bash
# Monitor network traffic
sudo tcpdump -i br-vms

# Check bridge traffic
sudo iftop -i br-vms

# Monitor TAP devices
ip -s link show type tun
```

#### Process Debugging

```bash
# Trace system calls
sudo strace -p $(pidof metald) -e trace=network

# Monitor file access
sudo inotifywait -m -r /opt/metald/

# Check open files
sudo lsof -p $(pidof metald)
```

### Recovery Procedures

#### Service Recovery

```bash
# Graceful restart
sudo systemctl reload metald

# Force restart (if hung)
sudo systemctl kill -s KILL metald
sudo systemctl start metald

# Emergency recovery (clear state)
sudo systemctl stop metald
sudo mv /opt/metald/data/metald.db /opt/metald/data/metald.db.backup
sudo systemctl start metald
```

#### Network Recovery

```bash
# Reset bridge
sudo ip link delete br-vms
sudo systemctl restart metald

# Clear iptables rules
sudo iptables -F
sudo iptables -t nat -F
sudo systemctl restart metald
```

#### Database Recovery

```bash
# Backup current database
sudo cp /opt/metald/data/metald.db /opt/metald/data/metald.db.$(date +%s)

# Repair database
sqlite3 /opt/metald/data/metald.db "PRAGMA integrity_check;"
sqlite3 /opt/metald/data/metald.db "VACUUM;"

# If corruption is severe, restore from backup
sudo systemctl stop metald
sudo cp /opt/metald/data/metald.db.backup /opt/metald/data/metald.db
sudo systemctl start metald
```

## Security Operations

### Access Control

#### Service Accounts

```bash
# Create metald service user
sudo useradd -r -s /bin/false -d /opt/metald metald
sudo usermod -a -G kvm metald

# Create firecracker isolation user
sudo useradd -r -s /bin/false -u 1000 firecracker
```

#### File Permissions

```bash
# Secure configuration files
sudo chmod 600 /etc/metald/environment
sudo chown root:metald /etc/metald/environment

# Secure database
sudo chmod 640 /opt/metald/data/metald.db
sudo chown metald:metald /opt/metald/data/metald.db

# Secure jailer directory
sudo chmod 755 /srv/jailer
sudo chown firecracker:firecracker /srv/jailer
```

### Certificate Management

#### SPIFFE Certificate Rotation

```bash
# Check certificate status
spire-agent api fetch -socketPath /var/lib/spire/agent/agent.sock

# Monitor certificate expiration
spire-agent api watch -socketPath /var/lib/spire/agent/agent.sock

# Manual certificate refresh (if needed)
sudo systemctl restart spire-agent
sudo systemctl restart metald
```

### Audit Logging

#### Security Event Monitoring

```bash
# Monitor authentication failures
sudo journalctl -u metald | grep "token validation failed"

# Monitor privilege escalation attempts
sudo journalctl -u metald | grep "SECURITY:"

# Monitor unusual VM operations
sudo journalctl -u metald | grep "force.*delete\|force.*shutdown"
```

## Backup and Recovery

### Database Backup

```bash
# Daily backup script
#!/bin/bash
BACKUP_DIR="/opt/metald/backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR
sqlite3 /opt/metald/data/metald.db ".backup $BACKUP_DIR/metald_$DATE.db"

# Keep only last 7 days
find $BACKUP_DIR -name "metald_*.db" -mtime +7 -delete
```

### Configuration Backup

```bash
# Backup configuration
sudo tar -czf /opt/metald/backups/config_$(date +%Y%m%d).tar.gz \
    /etc/metald/ \
    /etc/systemd/system/metald.service \
    /etc/metald/firewall-rules.sh
```

### Disaster Recovery

#### Full System Recovery

1. **Reinstall System**:
   ```bash
   # Install metald
   sudo make install
   
   # Restore configuration
   sudo tar -xzf config_backup.tar.gz -C /
   ```

2. **Restore Database**:
   ```bash
   sudo cp metald_backup.db /opt/metald/data/metald.db
   sudo chown metald:metald /opt/metald/data/metald.db
   ```

3. **Restart Services**:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable metald
   sudo systemctl start metald
   ```

## Maintenance

### Regular Maintenance Tasks

#### Weekly Tasks

```bash
# Database maintenance
sqlite3 /opt/metald/data/metald.db "VACUUM; ANALYZE;"

# Log rotation
sudo logrotate /etc/logrotate.d/metald

# Update firecracker binary
curl -fsSL https://github.com/firecracker-microvm/firecracker/releases/latest/download/firecracker-latest-x86_64.tgz | tar -xz
sudo mv firecracker-latest-x86_64 /usr/local/bin/firecracker
```

#### Monthly Tasks

```bash
# Security updates
sudo apt update && sudo apt upgrade

# Certificate renewal check
spire-agent api fetch -socketPath /var/lib/spire/agent/agent.sock

# Backup verification
sqlite3 /opt/metald/backups/metald_latest.db "PRAGMA integrity_check;"
```

### Capacity Planning

#### Resource Monitoring

```bash
# VM capacity tracking
curl -H "Authorization: Bearer <token>" http://localhost:8080/vmprovisioner.v1.VmService/ListVms | jq '.vms | length'

# Memory usage per VM
ps aux | grep firecracker | awk '{sum += $6} END {print sum/1024 " MB total"}'

# Network allocation usage
ip addr show | grep "172.31" | wc -l
```

#### Scaling Indicators

Monitor these metrics for scaling decisions:
- VM creation latency > 10 seconds
- Memory usage > 80% of available
- Network bridge utilization > 1000 VMs
- Database query time > 100ms
- Error rate > 1% of operations