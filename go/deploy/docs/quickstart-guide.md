# Quick Start Guide

Get metald and billaged running in under 10 minutes for local development.

## Prerequisites

- Go 1.21+ installed
- Linux or macOS (with Docker for Firecracker)
- 4GB RAM minimum
- Basic familiarity with terminal commands

## 1. Clone and Build (2 minutes)

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey.git
cd unkey/go/deploy

# Build both services
make -C metald build
make -C billaged build

# Verify builds
ls -la metald/build/metald billaged/build/billaged
```

## 2. Download VM Assets (3 minutes)

```bash
# Create assets directory
sudo mkdir -p /opt/vm-assets

# Download kernel and rootfs
cd metald
./scripts/setup-vm-assets.sh

# Verify assets
ls -la /opt/vm-assets/
# Should see: vmlinux, rootfs.ext4
```

## 3. Start Services (2 minutes)

### Terminal 1: Start billaged
```bash
cd billaged
./build/billaged
```

You should see:
```
INFO Starting billaged service port=8081
INFO OpenTelemetry disabled
INFO Server listening address=0.0.0.0:8081
```

### Terminal 2: Start metald
```bash
cd metald

# For Firecracker backend (recommended)
UNKEY_METALD_BACKEND=firecracker \
UNKEY_METALD_BILLING_ENABLED=true \
UNKEY_METALD_BILLING_ENDPOINT=http://localhost:8081 \
./build/metald
```

You should see:
```
INFO Starting metald with Firecracker backend
INFO Billing integration enabled endpoint=http://localhost:8081
INFO Server listening address=0.0.0.0:8080
```

## 4. Create Your First VM (3 minutes)

### Terminal 3: VM Operations

```bash
# Create a VM
VM_ID=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 134217728},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [{
        "path": "/opt/vm-assets/rootfs.ext4",
        "readonly": false
      }]
    }
  }' | jq -r '.vmId')

echo "Created VM: $VM_ID"

# Boot the VM
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/BootVm \
  -H "Content-Type: application/json" \
  -d "{\"vm_id\":\"$VM_ID\"}" | jq

# List VMs
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" \
  -d '{}' | jq

# Wait for metrics (60 seconds)
echo "Waiting for first metrics batch..."
sleep 65

# Check billing metrics
curl -s http://localhost:9465/metrics | grep billaged_metrics_processed_total

# Shutdown VM
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ShutdownVm \
  -H "Content-Type: application/json" \
  -d "{\"vm_id\":\"$VM_ID\", \"force\": true}" | jq

# Delete VM
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/DeleteVm \
  -H "Content-Type: application/json" \
  -d "{\"vm_id\":\"$VM_ID\"}" | jq
```

## 5. Verify Integration

Check that metald and billaged are communicating:

```bash
# Check metald logs (Terminal 2)
# Should see: "Started metrics collection" and "Sent metrics batch"

# Check billaged logs (Terminal 1)  
# Should see: "received metrics batch" with metrics_count=600

# Check metrics endpoints
curl -s http://localhost:9464/metrics | grep metald_
curl -s http://localhost:9465/metrics | grep billaged_
```

## Common Development Tasks

### Enable Debug Logging

```bash
# metald with debug logs
LOG_LEVEL=debug ./build/metald

# billaged with debug logs
LOG_LEVEL=debug ./build/billaged
```

### Run with Docker

```bash
# Build images
docker build -t metald:dev -f metald/Dockerfile metald/
docker build -t billaged:dev -f billaged/Dockerfile billaged/

# Run billaged
docker run -d --name billaged \
  -p 8081:8081 -p 9465:9465 \
  billaged:dev

# Run metald (requires privileged for Firecracker)
docker run -d --name metald \
  --privileged \
  -p 8080:8080 -p 9464:9464 \
  -v /opt/vm-assets:/opt/vm-assets:ro \
  -e UNKEY_METALD_BACKEND=firecracker \
  -e UNKEY_METALD_BILLING_ENABLED=true \
  -e UNKEY_METALD_BILLING_ENDPOINT=http://billaged:8081 \
  --link billaged \
  metald:dev
```

### Run Tests

```bash
# Unit tests
cd metald && go test ./...
cd billaged && go test ./...

# Integration tests
go test ./... -tags=integration

# Stress test
cd metald
./build/stress-test -max-vms 10 -interval-duration 30s
```

### Development Configuration

Create `.env` files for easier development:

**metald/.env**:
```bash
UNKEY_METALD_BACKEND=firecracker
UNKEY_METALD_PORT=8080
UNKEY_METALD_OTEL_ENABLED=true
UNKEY_METALD_BILLING_ENABLED=true
UNKEY_METALD_BILLING_ENDPOINT=http://localhost:8081
LOG_LEVEL=debug
```

**billaged/.env**:
```bash
BILLAGED_PORT=8081
BILLAGED_OTEL_ENABLED=true
BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=true
LOG_LEVEL=debug
```

Load with:
```bash
source .env && ./build/metald
source .env && ./build/billaged
```

## Troubleshooting

### "VM assets not found"
```bash
# Ensure assets are downloaded
./scripts/setup-vm-assets.sh

# Check permissions
sudo chown -R $USER:$USER /opt/vm-assets
```

### "Failed to create Firecracker process"
```bash
# Install Firecracker
curl -fsSL https://github.com/firecracker-microvm/firecracker/releases/download/v1.5.0/firecracker-v1.5.0-x86_64.tgz | tar -xz
sudo mv release-v1.5.0-x86_64/firecracker-v1.5.0-x86_64 /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker
```

### "Connection refused to billaged"
```bash
# Check billaged is running
ps aux | grep billaged

# Check port is open
netstat -tlnp | grep 8081

# Test connectivity
curl -f http://localhost:8081/health
```

### "No metrics received"
```bash
# Check VM is running
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" -d '{}' | jq

# Wait full 60 seconds for first batch
# Check metald logs for "Sent metrics batch"
```

## Next Steps

1. **Explore the APIs**:
   - [metald API Reference](../metald/docs/api-reference.md)
   - [billaged API Reference](../billaged/docs/api-reference.md)

2. **Configure for Production**:
   - [metald Configuration](../metald/docs/vm-configuration.md)
   - [billaged Configuration](../billaged/docs/configuration-guide.md)

3. **Set Up Monitoring**:
   - [Observability Guide](../metald/docs/observability.md)
   - [Monitoring Setup](monitoring-alerting-guide.md)

4. **Learn the Architecture**:
   - [Integration Guide](metald-billaged-integration.md)
   - [Billing Architecture](../metald/docs/billing-metrics-architecture.md)

## Quick Reference

### Service URLs
- metald API: http://localhost:8080
- metald Metrics: http://localhost:9464/metrics
- billaged API: http://localhost:8081  
- billaged Metrics: http://localhost:9465/metrics

### Key Commands
```bash
# Create VM
curl -X POST localhost:8080/vmprovisioner.v1.VmService/CreateVm -d '{...}'

# List VMs
curl -X POST localhost:8080/vmprovisioner.v1.VmService/ListVms -d '{}'

# Check Health
curl localhost:8080/health
curl localhost:8081/health
```

Congratulations! You now have a working metald + billaged development environment. 🎉