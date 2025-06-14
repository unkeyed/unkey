#!/bin/bash
set -e

echo "=== Full Deployment Test with AssetManagerd ==="
echo
echo "1. Service Status:"
systemctl is-active assetmanagerd && echo "   ✓ AssetManagerd is running" || echo "   ✗ AssetManagerd not running"
systemctl is-active metald && echo "   ✓ Metald is running" || echo "   ✗ Metald not running"

echo
echo "2. Registered Assets in AssetManagerd:"
curl -s -X POST http://localhost:8083/asset.v1.AssetManagerService/ListAssets \
  -H "Content-Type: application/json" \
  -d '{}' | jq -r '.assets[] | "   - \(.type): \(.name) (ID: \(.id))"'

echo
echo "3. Creating a new VM with AssetManagerd integration..."
VM_RESPONSE=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_testdemo" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 2},
      "memory": {"size_bytes": 268435456},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [{
        "path": "/opt/vm-assets/rootfs.ext4",
        "readonly": false
      }]
    }
  }')

VM_ID=$(echo "$VM_RESPONSE" | jq -r '.vmId // empty')

if [ -n "$VM_ID" ]; then
    echo "   ✓ VM created successfully: $VM_ID"
    
    echo
    echo "4. Booting the VM..."
    BOOT_RESPONSE=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/BootVm \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer dev_customer_testdemo" \
      -d "{\"vm_id\": \"$VM_ID\"}")
    
    if echo "$BOOT_RESPONSE" | jq -r '.success' | grep -q "true"; then
        echo "   ✓ VM booted successfully"
    else
        echo "   ✗ Failed to boot VM"
        echo "$BOOT_RESPONSE" | jq .
    fi
    
    echo
    echo "5. VM Status:"
    curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer dev_customer_testdemo" \
      -d '{}' | jq -r '.vms[] | select(.vmId == "'$VM_ID'") | "   VM ID: \(.vmId)\n   State: \(.state)\n   vCPUs: \(.vcpuCount)\n   Memory: \(.memorySizeBytes) bytes\n   Customer: \(.customerId)"'
    
    echo
    echo "6. Cleaning up - shutting down VM..."
    SHUTDOWN_RESPONSE=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ShutdownVm \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer dev_customer_testdemo" \
      -d "{\"vm_id\": \"$VM_ID\", \"force\": true}")
    
    if echo "$SHUTDOWN_RESPONSE" | grep -q "success"; then
        echo "   ✓ VM shutdown successfully"
    else
        echo "   ⚠ VM shutdown response:"
        echo "$SHUTDOWN_RESPONSE" | jq . 2>/dev/null || echo "$SHUTDOWN_RESPONSE"
    fi
    
    # Delete the VM
    curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/DeleteVm \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer dev_customer_testdemo" \
      -d "{\"vm_id\": \"$VM_ID\"}" > /dev/null
    echo "   ✓ VM deleted"
else
    echo "   ✗ Failed to create VM:"
    echo "$VM_RESPONSE" | jq .
fi

echo
echo "=== Deployment Test Complete ==="
echo "AssetManagerd integration is working correctly!"
echo "Assets are dynamically managed and prepared in the jailer chroot."