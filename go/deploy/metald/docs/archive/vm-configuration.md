# VM Configuration

Guide for configuring VMs through metald's unified API.

## Basic VM Configuration

```json
{
  "config": {
    "cpu": {
      "vcpu_count": 1
    },
    "memory": {
      "size_bytes": 134217728
    },
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

## Configuration Options

### CPU Configuration
```json
{
  "cpu": {
    "vcpu_count": 2,           // Number of virtual CPUs
    "max_vcpu_count": 4        // Maximum CPUs (for hotplug)
  }
}
```

### Memory Configuration  
```json
{
  "memory": {
    "size_bytes": 268435456,   // Memory size in bytes (256MB)
    "hotplug_size_bytes": 0    // Additional hotplug memory
  }
}
```

### Storage Configuration
```json
{
  "storage": [
    {
      "path": "/opt/vm-assets/rootfs.ext4",
      "readonly": false,
      "is_root_device": true
    },
    {
      "path": "/data/additional.ext4", 
      "readonly": true,
      "is_root_device": false
    }
  ]
}
```

### Boot Configuration
```json
{
  "boot": {
    "kernel_path": "/opt/vm-assets/vmlinux",
    "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off init=/sbin/init",
    "initrd_path": "/opt/vm-assets/initrd.img"  // Optional
  }
}
```

## Size Reference

| Memory Size | Bytes | Use Case |
|-------------|-------|----------|
| 128MB | 134,217,728 | Minimal microservices |
| 256MB | 268,435,456 | Light applications |  
| 512MB | 536,870,912 | Standard applications |
| 1GB | 1,073,741,824 | Memory-intensive apps |

## Asset Requirements

Both Firecracker and Cloud Hypervisor backends require:

### Kernel Image
- **Path**: `/opt/vm-assets/vmlinux`
- **Format**: Uncompressed Linux kernel
- **Requirements**: Built with required drivers (virtio, console)

### Root Filesystem
- **Path**: `/opt/vm-assets/rootfs.ext4` 
- **Format**: ext4 filesystem image
- **Contents**: Complete Linux root filesystem with init system

### Creating Assets
```bash
# Download pre-built assets (example)
wget https://s3.amazonaws.com/spec.ccfc.min/img/hello/kernel/hello-vmlinux.bin -O /opt/vm-assets/vmlinux
wget https://s3.amazonaws.com/spec.ccfc.min/img/hello/fsfiles/hello-rootfs.ext4 -O /opt/vm-assets/rootfs.ext4

# Or build custom kernel/rootfs as needed
```

## Backend-Specific Notes

### Firecracker
- Fast startup optimized for short-lived workloads
- Limited device support (no USB, GPU)
- Automatic FIFO metrics streaming
- Process isolation (one process per VM)

### Cloud Hypervisor  
- Full feature set including device passthrough
- Built-in networking support
- Live migration capabilities
- Shared daemon process

## Examples

### Minimal VM
```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -d '{"config":{"cpu":{"vcpu_count":1},"memory":{"size_bytes":134217728},"boot":{"kernel_path":"/opt/vm-assets/vmlinux","kernel_args":"console=ttyS0 reboot=k panic=1 pci=off"},"storage":[{"path":"/opt/vm-assets/rootfs.ext4","readonly":false}]}}'
```

### Multi-CPU VM with Additional Storage
```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -d '{"config":{"cpu":{"vcpu_count":2},"memory":{"size_bytes":536870912},"boot":{"kernel_path":"/opt/vm-assets/vmlinux","kernel_args":"console=ttyS0"},"storage":[{"path":"/opt/vm-assets/rootfs.ext4","readonly":false,"is_root_device":true},{"path":"/data/app.ext4","readonly":true}]}}'
```