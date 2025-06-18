#!/bin/bash
set -euo pipefail

echo "=== Testing Live AssetManager Integration ==="
echo

# Check if services are running
echo "1. Checking service status..."
if systemctl is-active --quiet metald; then
    echo "✓ metald is running"
else
    echo "✗ metald is not running"
    exit 1
fi

if systemctl is-active --quiet assetmanagerd; then
    echo "✓ assetmanagerd is running"
else
    echo "✗ assetmanagerd is not running"
    exit 1
fi

echo
echo "2. Checking metald logs for assetmanager initialization..."
if sudo journalctl -u metald -n 100 --no-pager | grep -q "initialized asset manager client.*enabled.*true"; then
    echo "✓ AssetManager client initialized successfully"
    sudo journalctl -u metald -n 100 --no-pager | grep "asset" | tail -5
else
    echo "✗ AssetManager client initialization not found in logs"
fi

echo
echo "3. Testing assetmanagerd API directly..."
if curl -s http://localhost:8082/health | jq -r '.status' | grep -q "ok"; then
    echo "✓ assetmanagerd health check passed"
else
    echo "✗ assetmanagerd health check failed"
fi

echo
echo "4. Testing metald API..."
if curl -s http://localhost:8080/health | jq -r '.status' | grep -q "ok"; then
    echo "✓ metald health check passed"
else
    echo "✗ metald health check failed"
fi

echo
echo "5. Checking for asset-related errors in metald logs..."
ERROR_COUNT=$(sudo journalctl -u metald -n 200 --no-pager | grep -i "asset" | grep -i "error" | wc -l)
if [ "$ERROR_COUNT" -eq 0 ]; then
    echo "✓ No asset-related errors found"
else
    echo "⚠ Found $ERROR_COUNT asset-related errors:"
    sudo journalctl -u metald -n 200 --no-pager | grep -i "asset" | grep -i "error" | tail -5
fi

echo
echo "6. Testing VM creation to trigger asset management..."
echo "Creating a test VM request..."

# Create a simple VM to test if assets are prepared
VM_RESPONSE=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 134217728},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [
        {"path": "/opt/vm-assets/rootfs.ext4", "readonly": false}
      ]
    }
  }' 2>&1 || echo "Request failed")

echo "VM creation response: $VM_RESPONSE"

# Give it a moment for logs to be written
sleep 2

echo
echo "7. Checking metald logs for asset preparation..."
echo "Recent asset-related logs:"
sudo journalctl -u metald -n 50 --no-pager | grep -E "(asset|Asset)" | tail -10 || echo "No recent asset logs found"

echo
echo "=== Summary ==="
echo "The AssetManager integration is:"
if sudo journalctl -u metald -n 100 --no-pager | grep -q "initialized asset manager client.*enabled.*true"; then
    echo "✓ CONFIGURED and INITIALIZED"
    
    if sudo journalctl -u metald -n 200 --no-pager | grep -q "prepared assets from assetmanagerd"; then
        echo "✓ ACTIVELY WORKING (assets prepared via assetmanagerd)"
    elif sudo journalctl -u metald -n 200 --no-pager | grep -q "no assets found in assetmanagerd"; then
        echo "⚠ WORKING but no assets registered in assetmanagerd yet"
    elif sudo journalctl -u metald -n 200 --no-pager | grep -q "failed to.*assets.*assetmanagerd"; then
        echo "⚠ CONFIGURED but falling back to hardcoded assets due to errors"
    else
        echo "⚠ CONFIGURED but not yet tested with VM creation"
    fi
else
    echo "✗ NOT PROPERLY CONFIGURED"
fi