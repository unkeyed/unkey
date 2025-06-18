# Metald Operations Guide

## Deployment Procedures

### Prerequisites

Before deploying Metald, ensure:

1. **System Requirements**
   - Linux kernel 4.14+ with KVM support
   - systemd 232+
   - Firecracker binary at `/usr/local/bin/firecracker`
   - 4GB+ RAM, 2+ CPU cores
   - 20GB+ available disk space

2. **Network Requirements**
   - CAP_NET_ADMIN capability or root access
   - IPv6 enabled (optional but recommended)
   - Bridge networking configured

3. **Security Requirements**
   - Dedicated system user (created by installer)
   - Appropriate Linux capabilities
   - SELinux/AppArmor policies (if enabled)

### Production Deployment

#### 1. Install from Source

```bash
# Clone repository
git clone https://github.com/unkeyed/unkey
cd unkey/go/deploy/metald

# Build and install
make build
sudo make install

# This will:
# - Create metald user and group
# - Install binary to /usr/local/bin/metald
# - Install systemd service file
# - Set required capabilities
# - Create necessary directories
```

#### 2. Configure Environment

```bash
# Edit environment file
sudo cp contrib/systemd/metald.env.example /etc/metald/metald.env
sudo chmod 600 /etc/metald/metald.env
sudo vim /etc/metald/metald.env

# Key configurations:
# - UNKEY_METALD_PORT=8080
# - UNKEY_METALD_JAILER_UID=977
# - UNKEY_METALD_JAILER_GID=976
# - UNKEY_METALD_BILLING_ENDPOINT=http://billaged:8081
# - UNKEY_METALD_ASSETMANAGER_ENDPOINT=http://assetmanager:8083
```

#### 3. Prepare Storage

```bash
# Create required directories
sudo mkdir -p /srv/jailer
sudo mkdir -p /opt/metald/assets
sudo mkdir -p /var/lib/metald
sudo mkdir -p /var/log/metald

# Set permissions
sudo chown -R metald:metald /opt/metald
sudo chown -R metald:metald /var/lib/metald
sudo chown -R metald:metald /var/log/metald
sudo chown root:root /srv/jailer
sudo chmod 755 /srv/jailer
```

#### 4. Configure Networking

```bash
# Create bridge for VM networking (if needed)
sudo ip link add br0 type bridge
sudo ip addr add 192.168.100.1/24 dev br0
sudo ip link set br0 up

# Enable IP forwarding
sudo sysctl -w net.ipv4.ip_forward=1
sudo sysctl -w net.ipv6.conf.all.forwarding=1

# Configure firewall (example for iptables)
sudo iptables -t nat -A POSTROUTING -s 192.168.100.0/24 -j MASQUERADE
sudo iptables -A FORWARD -i br0 -j ACCEPT
sudo iptables -A FORWARD -o br0 -j ACCEPT
```

#### 5. Start Service

```bash
# Enable and start metald
sudo systemctl enable metald
sudo systemctl start metald

# Check status
sudo systemctl status metald
sudo journalctl -u metald -f
```

### Docker Deployment (Development)

```bash
# Build container
docker build -t metald:latest .

# Run with privileges (required for VM management)
docker run -d \
  --name metald \
  --privileged \
  --network host \
  -v /dev/kvm:/dev/kvm \
  -v /srv/jailer:/srv/jailer \
  -v /opt/metald:/opt/metald \
  -e UNKEY_METALD_PORT=8080 \
  metald:latest
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: metald
spec:
  selector:
    matchLabels:
      app: metald
  template:
    metadata:
      labels:
        app: metald
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: metald
        image: metald:latest
        securityContext:
          privileged: true
          capabilities:
            add:
            - SYS_ADMIN
            - NET_ADMIN
            - SYS_CHROOT
        volumeMounts:
        - name: dev-kvm
          mountPath: /dev/kvm
        - name: jailer-root
          mountPath: /srv/jailer
      volumes:
      - name: dev-kvm
        hostPath:
          path: /dev/kvm
      - name: jailer-root
        hostPath:
          path: /srv/jailer
```

## Monitoring Setup

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'metald'
    static_configs:
      - targets: ['metald:9464']
    metric_relabel_configs:
      # Drop high cardinality metrics in production
      - source_labels: [__name__]
        regex: '.*_bucket|.*_sum|.*_count'
        action: drop
```

### Key Metrics to Monitor

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `unkey_metald_active_vms` | Current VM count | > 900 (capacity) |
| `unkey_metald_vm_boot_duration_seconds` | VM startup time | > 1s (p99) |
| `unkey_metald_backend_errors_total` | Backend failures | > 10/min |
| `unkey_metald_api_request_duration_seconds` | API latency | > 100ms (p99) |
| `process_resident_memory_bytes` | Memory usage | > 4GB |
| `go_goroutines` | Goroutine count | > 10000 |

### Grafana Dashboard

Import the dashboard from `contrib/grafana-dashboards/metald.json` or create custom panels:

```json
{
  "dashboard": {
    "title": "Metald Operations",
    "panels": [
      {
        "title": "Active VMs by Customer",
        "targets": [{
          "expr": "sum(unkey_metald_active_vms) by (customer_id)"
        }]
      },
      {
        "title": "VM Boot Performance",
        "targets": [{
          "expr": "histogram_quantile(0.99, unkey_metald_vm_boot_duration_seconds_bucket)"
        }]
      }
    ]
  }
}
```

### Alerting Rules

```yaml
# alerts.yml
groups:
  - name: metald
    rules:
      - alert: MetaldDown
        expr: up{job="metald"} == 0
        for: 5m
        annotations:
          summary: "Metald service is down"
          
      - alert: HighVMBootLatency
        expr: histogram_quantile(0.99, unkey_metald_vm_boot_duration_seconds_bucket) > 1
        for: 10m
        annotations:
          summary: "VM boot time exceeds 1 second"
          
      - alert: BackendErrors
        expr: rate(unkey_metald_backend_errors_total[5m]) > 0.1
        annotations:
          summary: "Firecracker backend errors detected"
```

## Troubleshooting Guide

### Common Issues

#### 1. Service Won't Start

**Symptoms**: `systemctl status metald` shows failed state

**Diagnostics**:
```bash
# Check logs
sudo journalctl -u metald -n 100 --no-pager

# Verify capabilities
getcap /usr/local/bin/metald

# Check file permissions
ls -la /srv/jailer /opt/metald /var/lib/metald

# Test firecracker binary
/usr/local/bin/firecracker --version
```

**Solutions**:
- Ensure all directories exist with correct permissions
- Verify firecracker binary is accessible
- Check for port conflicts on 8080
- Ensure KVM module is loaded: `lsmod | grep kvm`

#### 2. VM Creation Fails

**Symptoms**: CreateVm returns errors

**Diagnostics**:
```bash
# Check system resources
free -h
df -h /srv/jailer

# Verify network setup
ip link show
ip netns list

# Check for capability issues
sudo -u metald /usr/local/bin/metald --help

# Look for jailer errors
sudo journalctl -u metald | grep -i jailer
```

**Solutions**:
- Ensure sufficient disk space in /srv/jailer
- Verify CAP_NET_ADMIN is set
- Check network namespace limits: `sysctl user.max_net_namespaces`
- Increase file descriptor limits if needed

#### 3. Network Connectivity Issues

**Symptoms**: VMs can't reach network

**Diagnostics**:
```bash
# Check TAP devices
ip link show | grep tap

# Verify namespace networking
ip netns exec ns_vm_xxxxx ip addr

# Check iptables rules
sudo iptables -L -n -v
sudo iptables -t nat -L -n -v

# Test bridge connectivity
ping -c1 192.168.100.1
```

**Solutions**:
- Ensure IP forwarding is enabled
- Add missing iptables rules
- Check bridge configuration
- Verify no conflicting network policies

#### 4. High Memory Usage

**Symptoms**: Metald process using excessive memory

**Diagnostics**:
```bash
# Check process memory
ps aux | grep metald
pmap -x $(pidof metald)

# Count active VMs
curl -s http://localhost:8080/health | jq .

# Check for goroutine leaks
curl -s http://localhost:8080/debug/pprof/goroutine
```

**Solutions**:
- Restart service to clear any leaks
- Check for VMs not being cleaned up
- Reduce VM count if approaching limits
- Enable memory profiling for analysis

### Debug Mode

Enable verbose logging for troubleshooting:

```bash
# Temporary debug mode
sudo systemctl stop metald
sudo -u metald UNKEY_METALD_LOG_LEVEL=debug /usr/local/bin/metald

# Or update systemd environment
sudo systemctl edit metald
# Add: Environment="UNKEY_METALD_LOG_LEVEL=debug"
sudo systemctl restart metald
```

### Performance Tuning

#### System Tuning

```bash
# Increase file descriptor limits
echo "metald soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "metald hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# Increase network namespace limit
echo "user.max_net_namespaces = 2000" | sudo tee -a /etc/sysctl.conf

# Increase inotify limits for monitoring
echo "fs.inotify.max_user_instances = 512" | sudo tee -a /etc/sysctl.conf
echo "fs.inotify.max_user_watches = 524288" | sudo tee -a /etc/sysctl.conf

sudo sysctl -p
```

#### Metald Tuning

```bash
# Environment variables for performance
UNKEY_METALD_MAX_PROCESSES=2000         # Max concurrent VMs
UNKEY_METALD_OTEL_SAMPLING_RATE=0.1    # Reduce tracing overhead
UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED=false  # Reduce metrics
```

## Disaster Recovery

### Backup Procedures

#### 1. Database Backup

```bash
# Stop writes (optional for consistency)
curl -X POST http://localhost:8080/admin/maintenance/enable

# Backup SQLite database
sudo -u metald sqlite3 /var/lib/metald/metald.db ".backup /backup/metald-$(date +%Y%m%d).db"

# Resume writes
curl -X POST http://localhost:8080/admin/maintenance/disable
```

#### 2. Configuration Backup

```bash
# Backup all configuration
sudo tar -czf /backup/metald-config-$(date +%Y%m%d).tar.gz \
  /etc/metald/ \
  /etc/systemd/system/metald.service \
  /etc/systemd/system/metald.service.d/
```

#### 3. VM State Export

```bash
# Export running VM list
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/v1/vms > /backup/vms-$(date +%Y%m%d).json
```

### Restore Procedures

#### 1. Service Restoration

```bash
# Stop service
sudo systemctl stop metald

# Restore database
sudo -u metald cp /backup/metald-20240118.db /var/lib/metald/metald.db

# Restore configuration
sudo tar -xzf /backup/metald-config-20240118.tar.gz -C /

# Start service
sudo systemctl start metald
```

#### 2. VM Recovery

VMs are ephemeral and not restored directly. Recreate from configuration:

```bash
# Parse backed up VM list and recreate
jq -r '.vms[] | @json' /backup/vms-20240118.json | while read vm; do
  echo "$vm" | jq -r '.config' | \
    curl -X POST http://localhost:8080/v1/vms \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json" \
      -d @-
done
```

### Emergency Procedures

#### Service Hung

```bash
# Generate goroutine dump
kill -QUIT $(pidof metald)

# Force restart
sudo systemctl kill -s KILL metald
sudo systemctl start metald
```

#### Disk Full

```bash
# Clean up old jailer directories
sudo find /srv/jailer -type d -name "firecracker" -mtime +1 -exec rm -rf {} +

# Clean up socket files
sudo find /opt/metald/sockets -type s -mtime +1 -delete

# Rotate logs
sudo journalctl --vacuum-time=1d
```

#### Network Exhaustion

```bash
# Clean up orphaned namespaces
for ns in $(ip netns list | grep '^ns_vm_' | awk '{print $1}'); do
  if ! pgrep -f "firecracker.*$ns" > /dev/null; then
    sudo ip netns delete "$ns"
  fi
done
```

## Maintenance Windows

### Rolling Updates

```bash
# For single node (causes downtime)
sudo systemctl stop metald
sudo cp /new/metald /usr/local/bin/metald
sudo systemctl start metald
```

### Configuration Updates

```bash
# Update environment file
sudo vim /etc/metald/metald.env

# Reload and restart
sudo systemctl daemon-reload
sudo systemctl restart metald
```

### Certificate Rotation (SPIFFE)

```bash
# SPIFFE handles automatic rotation
# Force renewal if needed
sudo systemctl restart spire-agent
sudo systemctl restart metald
```

## Health Checks

### Automated Health Monitoring

```bash
#!/bin/bash
# health-check.sh

# Check service status
if ! systemctl is-active --quiet metald; then
  echo "CRITICAL: Metald service not running"
  exit 2
fi

# Check API health
if ! curl -sf http://localhost:8080/health > /dev/null; then
  echo "CRITICAL: Metald API not responding"
  exit 2
fi

# Check backend health
health=$(curl -s http://localhost:8080/health | jq -r .backend.status)
if [ "$health" != "healthy" ]; then
  echo "WARNING: Backend unhealthy"
  exit 1
fi

echo "OK: Metald healthy"
exit 0
```

### Manual Health Verification

```bash
# Check overall health
curl -s http://localhost:8080/health | jq .

# Test VM creation
curl -X POST http://localhost:8080/v1/vms \
  -H "Authorization: Bearer dev_customer_test" \
  -H "Content-Type: application/json" \
  -d '{"config": {"cpu": {"vcpu_count": 1}, "memory": {"size_bytes": 134217728}}}'

# Verify metrics
curl -s http://localhost:8080/metrics | grep ^unkey_metald
```

## Security Operations

### Audit Logging

```bash
# Enable audit logging
auditctl -w /usr/local/bin/metald -p x -k metald_exec
auditctl -w /srv/jailer -p wa -k jailer_access

# Review audit logs
ausearch -k metald_exec
ausearch -k jailer_access
```

### Security Scanning

```bash
# Check for vulnerable dependencies
cd /home/dev/metald
go list -m all | nancy sleuth

# Scan container image
trivy image metald:latest

# Check file permissions
find /opt/metald -type f -perm /o+w -ls
find /srv/jailer -type d -perm /o+w -ls
```

### Incident Response

1. **Isolate**: Stop accepting new VMs
   ```bash
   iptables -I INPUT -p tcp --dport 8080 -j REJECT
   ```

2. **Investigate**: Collect evidence
   ```bash
   sudo tar -czf /tmp/incident-$(date +%Y%m%d).tar.gz \
     /var/log/metald/ \
     /var/lib/metald/ \
     /srv/jailer/
   ```

3. **Remediate**: Apply fixes and restart
   ```bash
   sudo systemctl stop metald
   # Apply security patches
   sudo systemctl start metald
   ```

4. **Report**: Document incident and response