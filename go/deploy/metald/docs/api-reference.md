# API Reference

ConnectRPC-based API for VM management. Supports both gRPC and HTTP protocols.

## Base URL
- HTTP: `http://localhost:8080`
- gRPC: `localhost:8080`

## VM Management APIs

### CreateVm
Creates a new VM instance.

**Endpoint**: `POST /vmprovisioner.v1.VmService/CreateVm`

**Request**:
```json
{
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
}
```

**Response**:
```json
{
  "vmId": "ud-1234567890abcdef",
  "state": "VM_STATE_CREATED"
}
```

### BootVm
Boots a created VM.

**Endpoint**: `POST /vmprovisioner.v1.VmService/BootVm`

**Request**:
```json
{
  "vm_id": "ud-1234567890abcdef"
}
```

### ListVms
Lists all VMs.

**Endpoint**: `POST /vmprovisioner.v1.VmService/ListVms`

**Request**: `{}`

**Response**:
```json
{
  "vms": [
    {
      "vmId": "ud-1234567890abcdef",
      "state": "VM_STATE_RUNNING",
      "vcpuCount": 1,
      "memorySizeBytes": "134217728",
      "createdTimestamp": "1749646093",
      "modifiedTimestamp": "1749646093"
    }
  ],
  "totalCount": 1
}
```

### ShutdownVm
Shuts down a running VM.

**Endpoint**: `POST /vmprovisioner.v1.VmService/ShutdownVm`

**Request**:
```json
{
  "vm_id": "ud-1234567890abcdef",
  "force": true,
  "timeout_seconds": 10
}
```

### DeleteVm
Deletes a VM.

**Endpoint**: `POST /vmprovisioner.v1.VmService/DeleteVm`

**Request**:
```json
{
  "vm_id": "ud-1234567890abcdef"
}
```

## VM States

- `VM_STATE_CREATED` - VM created but not running
- `VM_STATE_RUNNING` - VM is running
- `VM_STATE_SHUTDOWN` - VM is shut down
- `VM_STATE_PAUSED` - VM is paused

## Error Handling

API returns standard HTTP status codes:
- `200` - Success
- `400` - Invalid request
- `500` - Internal server error

Error responses include details:
```json
{
  "code": "invalid_argument",
  "message": "vm_id is required"
}
```

## Examples

### Create and Boot VM
```bash
# 1. Create VM
VM_ID=$(curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{"config":{"cpu":{"vcpu_count":1},"memory":{"size_bytes":134217728},"boot":{"kernel_path":"/opt/vm-assets/vmlinux","kernel_args":"console=ttyS0 reboot=k panic=1 pci=off"},"storage":[{"path":"/opt/vm-assets/rootfs.ext4","readonly":false}]}}' \
  | jq -r '.vmId')

# 2. Boot VM
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/BootVm \
  -H "Content-Type: application/json" \
  -d "{\"vm_id\":\"$VM_ID\"}"

# 3. Check status
curl -s -X POST http://localhost:8080/vmprovisioner.v1.VmService/ListVms \
  -H "Content-Type: application/json" \
  -d '{}'
```