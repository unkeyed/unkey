#!/bin/bash
set -euo pipefail

echo "=== Final AssetManager Integration Test ==="
echo

# 1. Verify services are running
echo "1. Service Status:"
echo -n "   metald: "
systemctl is-active metald || echo "not running"
echo -n "   assetmanagerd: "
systemctl is-active assetmanagerd || echo "not running"

echo
echo "2. Endpoint Configuration:"
echo -n "   AssetManager endpoint in metald: "
sudo journalctl -u metald --since "5 minutes ago" --no-pager | grep -o '"endpoint":"[^"]*"' | tail -1 | cut -d'"' -f4

echo
echo "3. Health Checks:"
echo -n "   assetmanagerd (port 8083): "
if curl -s http://localhost:8083/health | grep -q "OK"; then
    echo "✓ OK"
else
    echo "✗ Failed"
fi

echo -n "   metald (port 8080): "
if curl -s http://localhost:8080/health | jq -r '.status' | grep -q "ok"; then
    echo "✓ OK"
else
    echo "✗ Failed"
fi

echo
echo "4. Integration Status:"
if sudo journalctl -u metald --since "5 minutes ago" --no-pager | grep -q "initialized asset manager client.*enabled.*true.*8083"; then
    echo "   ✓ AssetManager client initialized with correct endpoint"
else
    echo "   ✗ AssetManager client not properly initialized"
fi

echo
echo "5. Testing asset query (simulating VM creation):"
# First, let's register a test asset in assetmanagerd
echo "   Registering a test kernel asset..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8083/asset.v1.AssetManagerService/RegisterAsset \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-vmlinux",
    "type": "ASSET_TYPE_KERNEL",
    "backend": "STORAGE_BACKEND_LOCAL",
    "location": "/opt/vm-assets/vmlinux",
    "size_bytes": 1024,
    "checksum": "test123",
    "labels": {"test": "true"},
    "created_by": "integration-test"
  }' 2>&1 || echo "Failed to register")

if echo "$REGISTER_RESPONSE" | grep -q "asset"; then
    echo "   ✓ Test asset registered successfully"
else
    echo "   Registration response: $REGISTER_RESPONSE"
fi

# Now create a VM to trigger asset management
echo
echo "   Creating a test VM to trigger asset management..."
VM_RESPONSE=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 134217728},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0"
      }
    }
  }' 2>&1 || echo "VM creation failed")

# Give logs time to be written
sleep 2

echo
echo "6. Asset Management Activity:"
echo "   Recent asset-related logs from metald:"
if sudo journalctl -u metald --since "30 seconds ago" --no-pager | grep -i asset | grep -v "initialized asset manager"; then
    echo "   ✓ Asset management activity detected"
else
    echo "   ⚠ No recent asset management activity"
fi

echo
echo "=== Summary ==="
INTEGRATION_WORKING=true

# Check if client is initialized
if ! sudo journalctl -u metald --since "5 minutes ago" --no-pager | grep -q "initialized asset manager client.*enabled.*true.*8083"; then
    INTEGRATION_WORKING=false
    echo "✗ AssetManager client not initialized properly"
fi

# Check if services can communicate
if ! curl -s http://localhost:8083/health | grep -q "OK"; then
    INTEGRATION_WORKING=false
    echo "✗ AssetManagerd not accessible"
fi

if [ "$INTEGRATION_WORKING" = true ]; then
    echo "✓ Integration is CONFIGURED and ACCESSIBLE"
    
    # Check if it's actively being used
    if sudo journalctl -u metald --since "1 minute ago" --no-pager | grep -q "assetmanager rpc"; then
        echo "✓ Integration is ACTIVELY WORKING"
    elif sudo journalctl -u metald --since "1 minute ago" --no-pager | grep -q "no assets found in assetmanagerd"; then
        echo "⚠ Integration is WORKING but no assets are registered yet"
    elif sudo journalctl -u metald --since "1 minute ago" --no-pager | grep -q "hardcoded assets"; then
        echo "⚠ Integration is CONFIGURED but falling back to hardcoded assets"
    else
        echo "⚠ Integration is CONFIGURED but not yet tested with VM operations"
    fi
else
    echo "✗ Integration is NOT WORKING properly"
fi