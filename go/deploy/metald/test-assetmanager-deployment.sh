#!/bin/bash
set -e

echo "=== Testing AssetManagerd Integration ==="
echo

# Check services are running
echo "1. Checking service status..."
systemctl status assetmanagerd --no-pager | grep "Active:" || echo "AssetManagerd not running"
systemctl status metald --no-pager | grep "Active:" || echo "Metald not running"
echo

# List registered assets
echo "2. Listing registered assets..."
curl -s -X POST http://localhost:8083/asset.v1.AssetManagerService/ListAssets \
  -H "Content-Type: application/json" \
  -d '{}' | jq -r '.assets[] | "\(.type): \(.name) (ID: \(.id))"'
echo

# Get kernel and rootfs IDs
KERNEL_ID=$(curl -s -X POST http://localhost:8083/asset.v1.AssetManagerService/ListAssets \
  -H "Content-Type: application/json" \
  -d '{"type": "ASSET_TYPE_KERNEL"}' | jq -r '.assets[0].id')

ROOTFS_ID=$(curl -s -X POST http://localhost:8083/asset.v1.AssetManagerService/ListAssets \
  -H "Content-Type: application/json" \
  -d '{"type": "ASSET_TYPE_ROOTFS", "labelSelector": {"os": "linux"}}' | jq -r '.assets[0].id')

echo "Using Kernel ID: $KERNEL_ID"
echo "Using Rootfs ID: $ROOTFS_ID"
echo

# Create VM with assetmanagerd integration
echo "3. Creating VM with assetmanagerd integration..."

# The paths should be relative to the jailer chroot, not the host filesystem
VM_RESPONSE=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_test123" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 134217728},
      "boot": {
        "kernel_path": "opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [{
        "path": "opt/vm-assets/rootfs.ext4",
        "readonly": false
      }]
    }
  }')

echo "VM Creation Response:"
echo "$VM_RESPONSE" | jq .

# Extract VM ID if successful
VM_ID=$(echo "$VM_RESPONSE" | jq -r '.vmId // empty')

if [ -n "$VM_ID" ]; then
    echo
    echo "4. VM created successfully with ID: $VM_ID"
    echo
    echo "5. Checking VM status..."
    curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer dev_customer_test123" \
      -d '{}' | jq -r '.vms[] | select(.vmId == "'$VM_ID'") | "VM: \(.vmId), State: \(.state)"'
    
    echo
    echo "6. Check assetmanagerd logs for asset preparation:"
    sudo journalctl -u assetmanagerd -n 20 --no-pager | grep -E "(PrepareAssets|$VM_ID)" | tail -5
else
    echo
    echo "Failed to create VM. Checking logs..."
    echo
    echo "Metald logs:"
    sudo journalctl -u metald -n 30 --no-pager | grep -E "(ERROR|failed)" | tail -10
    echo
    echo "AssetManagerd logs:"
    sudo journalctl -u assetmanagerd -n 20 --no-pager | tail -10
fi