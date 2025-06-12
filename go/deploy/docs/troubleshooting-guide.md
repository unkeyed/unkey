# Troubleshooting Guide

Comprehensive troubleshooting guide for common issues with metald and billaged services.

## Quick Diagnostics

```bash
# Check service health
curl -f http://localhost:8080/health  # metald
curl -f http://localhost:8081/health  # billaged

# Check metrics endpoints
curl -s http://localhost:9464/metrics | grep -E "(up|metald_)" | head -20
curl -s http://localhost:9465/metrics | grep -E "(up|billaged_)" | head -20

# Check recent logs
journalctl -u metald -n 100 --no-pager | grep -E "(ERROR|WARN)"
journalctl -u billaged -n 100 --no-pager | grep -E "(ERROR|WARN)"

# Check system resources
free -h
df -h
ps aux | grep -E "(metald|billaged)" | grep -v grep
```

## Common Issues

### 1. Service Won't Start

#### Symptom: "bind: address already in use"
```
Error: listen tcp :8080: bind: address already in use
```

**Solution:**
```bash
# Find process using the port
sudo lsof -i :8080
# or
sudo netstat -tlnp | grep 8080

# Kill the process or change port
export UNKEY_METALD_PORT=8090  # metald
export BILLAGED_PORT=8091       # billaged
```

#### Symptom: "permission denied"
```
Error: open /var/lib/metald/state.db: permission denied
```

**Solution:**
```bash
# Fix permissions
sudo chown -R $(whoami):$(whoami) /var/lib/metald
sudo chown -R $(whoami):$(whoami) /var/lib/billaged

# Or run with correct user
sudo -u metald ./metald
```

### 2. VM Creation Failures

#### Symptom: "failed to create VM: firecracker not found"
```
Error: failed to create firecracker process: exec: "firecracker": executable file not found in $PATH
```

**Solution:**
```bash
# Install Firecracker
curl -fsSL https://github.com/firecracker-microvm/firecracker/releases/download/v1.5.0/firecracker-v1.5.0-x86_64.tgz | tar -xz
sudo mv release-v1.5.0-x86_64/firecracker-v1.5.0-x86_64 /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker

# Verify installation
firecracker --version
```

#### Symptom: "failed to create VM: assets not found"
```
Error: kernel file not found: /opt/vm-assets/vmlinux
```

**Solution:**
```bash
# Download VM assets
cd metald
sudo ./scripts/setup-vm-assets.sh

# Verify assets
ls -la /opt/vm-assets/
# Should see: vmlinux, rootfs.ext4

# Fix permissions if needed
sudo chmod 644 /opt/vm-assets/*
```

#### Symptom: "failed to create TAP device"
```
Error: failed to create TAP device: operation not permitted
```

**Solution:**
```bash
# Option 1: Run with sudo (development only)
sudo ./metald

# Option 2: Set CAP_NET_ADMIN capability
sudo setcap cap_net_admin+ep /path/to/metald

# Option 3: Use pre-created TAP devices
sudo ip tuntap add tap0 mode tap
sudo ip link set tap0 up
```

### 3. Billing Integration Issues

#### Symptom: "failed to send metrics batch: connection refused"
```
Error: failed to send metrics batch: Post "http://localhost:8081/billing.v1.BillingService/SendMetricsBatch": dial tcp 127.0.0.1:8081: connect: connection refused
```

**Solution:**
```bash
# 1. Verify billaged is running
ps aux | grep billaged
systemctl status billaged

# 2. Check billaged is listening
netstat -tlnp | grep 8081

# 3. Test connectivity
curl -f http://localhost:8081/health

# 4. Check configuration
echo $UNKEY_METALD_BILLING_ENDPOINT
# Should be: http://localhost:8081 or http://billaged:8081

# 5. If using Docker, ensure network connectivity
docker network ls
docker inspect metald | jq '.[0].NetworkSettings.Networks'
```

#### Symptom: "no metrics received in billaged"
```
Log: Waiting for metrics batch...
```

**Solution:**
```bash
# 1. Verify VM is running
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" -d '{}' | jq '.vms[] | select(.state == "VM_STATE_RUNNING")'

# 2. Check metrics collection is enabled
env | grep BILLING
# Should see: UNKEY_METALD_BILLING_ENABLED=true

# 3. Check metald logs for collection
journalctl -u metald -n 1000 | grep -E "(metrics collection|batch)"

# 4. Wait full 60 seconds (batch interval)
# Metrics are sent in 600-sample batches every 60 seconds

# 5. Check for errors in metald
journalctl -u metald -n 1000 | grep -E "(ERROR.*billing|failed.*batch)"
```

### 4. Performance Issues

#### Symptom: High Memory Usage
```
metald using 4GB+ memory with only 10 VMs
```

**Solution:**
```bash
# 1. Check for memory leaks
go tool pprof http://localhost:8080/debug/pprof/heap

# 2. Check goroutine count
curl -s http://localhost:8080/debug/pprof/goroutine?debug=1 | head -20

# 3. Possible causes:
# - Metrics buffer not being flushed
# - Failed billing batches accumulating
# - VM cleanup not happening

# 4. Restart with debug logging
LOG_LEVEL=debug ./metald 2>&1 | grep -E "(buffer|batch|cleanup)"

# 5. Check VM cleanup
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" -d '{}' | jq '.vms | length'
```

#### Symptom: Slow API Response
```
VM creation taking >10 seconds
```

**Solution:**
```bash
# 1. Enable trace logging
export LOG_LEVEL=trace
export UNKEY_METALD_OTEL_ENABLED=true

# 2. Check disk I/O
iostat -x 1

# 3. Check CPU usage
top -p $(pgrep metald)

# 4. Profile the service
curl -s http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# 5. Common bottlenecks:
# - Slow disk for VM images
# - Network latency to billaged
# - Lock contention in VM manager
```

### 5. Firecracker-specific Issues

#### Symptom: "failed to open /dev/kvm"
```
Error: failed to open /dev/kvm: No such file or directory
```

**Solution:**
```bash
# 1. Check KVM support
ls -la /dev/kvm
lsmod | grep kvm

# 2. Enable KVM (Intel)
sudo modprobe kvm_intel

# 3. Enable KVM (AMD)
sudo modprobe kvm_amd

# 4. Add user to kvm group
sudo usermod -aG kvm $USER
newgrp kvm

# 5. For containers/cloud environments, may need to use TCG
# Set in VM config: "machine_type": "microvm,accel=tcg"
```

#### Symptom: "Firecracker process died unexpectedly"
```
Error: firecracker process exited with status 1
```

**Solution:**
```bash
# 1. Check Firecracker logs
cat /var/log/firecracker/firecracker-*.log

# 2. Common causes:
# - Invalid VM configuration
# - Resource conflicts (TAP devices, VSOCK)
# - Kernel/rootfs compatibility

# 3. Test Firecracker directly
firecracker --api-sock /tmp/firecracker.socket --config-file test-config.json

# 4. Validate configuration
cat vm-config.json | jq . # Check for JSON errors

# 5. Check resource limits
ulimit -a
# Increase if needed:
ulimit -n 65536  # file descriptors
ulimit -u 32768  # processes
```

### 6. Data Integrity Issues

#### Symptom: "duplicate metrics detected"
```
WARN: Duplicate metrics batch detected for vm_id=ud-123, timestamp=...
```

**Solution:**
```bash
# 1. This is often normal during retries
# The billing service handles duplicates

# 2. If persistent, check time sync
timedatectl status
# Ensure NTP is synchronized

# 3. Check for multiple metald instances
ps aux | grep metald | grep -v grep

# 4. Verify VM ID uniqueness
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" -d '{}' | jq '.vms[].vmId' | sort | uniq -d
```

#### Symptom: "metrics gap detected"
```
WARN: Possible data gap for vm_id=ud-123, gap_duration=180000ms
```

**Solution:**
```bash
# 1. Check metald health during gap period
journalctl -u metald --since "10 minutes ago" | grep -E "(ERROR|restarted)"

# 2. Check network issues
ping -c 10 billaged-host
mtr billaged-host

# 3. Review billing retry logic
grep -A 10 "retryWithBackoff" metald.log

# 4. Manual reconciliation may be needed
# Contact operations team with VM ID and time range
```

## Debug Commands

### Enable Verbose Logging

```bash
# metald
LOG_LEVEL=debug ./metald 2>&1 | tee metald-debug.log

# billaged  
LOG_LEVEL=debug ./billaged 2>&1 | tee billaged-debug.log

# Filter for specific components
tail -f metald-debug.log | grep -E "(billing|firecracker|metrics)"
```

### Network Debugging

```bash
# Trace API calls
tcpdump -i lo -w api-trace.pcap port 8080 or port 8081

# Monitor metrics endpoints
watch -n 5 'curl -s http://localhost:9464/metrics | grep metald_billing'

# Test connectivity between services
nc -zv localhost 8081
curl -X POST http://localhost:8081/billing.v1.BillingService/SendHeartbeat \
  -H "Content-Type: application/json" \
  -d '{"instance_id": "test", "active_vms": []}'
```

### Resource Debugging

```bash
# Monitor file descriptors
watch -n 1 'lsof -p $(pgrep metald) | wc -l'

# Check goroutines
curl -s http://localhost:8080/debug/pprof/goroutine?debug=1 | grep "goroutine" | wc -l

# Memory profile
go tool pprof -http=:6060 http://localhost:8080/debug/pprof/heap

# CPU profile
curl -s http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof -http=:6060 cpu.prof
```

## Recovery Procedures

### Emergency VM Cleanup

```bash
# List all VMs
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" -d '{}' | jq -r '.vms[].vmId' > vm-list.txt

# Shutdown all VMs
while read vm_id; do
  curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/ShutdownVm \
    -H "Content-Type: application/json" \
    -d "{\"vm_id\":\"$vm_id\", \"force\": true}"
  sleep 1
done < vm-list.txt

# Clean up Firecracker processes
pkill -f firecracker

# Clean up TAP devices
for tap in $(ip link show | grep tap | cut -d: -f2); do
  sudo ip link delete $tap
done
```

### Service Recovery

```bash
# Full restart sequence
sudo systemctl stop metald billaged
sleep 5
sudo systemctl start billaged
sleep 5
sudo systemctl start metald

# Verify recovery
curl -f http://localhost:8080/health && echo "metald OK"
curl -f http://localhost:8081/health && echo "billaged OK"

# Check metrics flow
sleep 65  # Wait for first batch
curl -s http://localhost:9465/metrics | grep billaged_metrics_processed_total
```

## Getting Help

If issues persist after trying these solutions:

1. **Collect Diagnostics**:
   ```bash
   ./scripts/collect-diagnostics.sh
   ```

2. **Check Documentation**:
   - [API Reference](../metald/docs/api-reference.md)
   - [Integration Guide](metald-billaged-integration.md)
   - [Configuration Guide](../billaged/docs/configuration-guide.md)

3. **File an Issue**:
   - Include diagnostic bundle
   - Describe steps to reproduce
   - Include relevant log snippets

4. **Community Support**:
   - GitHub Discussions
   - Slack: #metald-help

Remember: Most issues are configuration-related. Double-check environment variables and network connectivity first!