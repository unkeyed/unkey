# Backend Support

metald supports multiple VMM backends through a unified interface.

## Supported Backends

### Firecracker (Recommended)
**Use Cases**: Serverless functions, short-lived workloads, high security requirements

**Strengths**:
- ⚡ **Fast startup**: <100ms boot time
- 🔒 **High security**: Minimal attack surface
- 📊 **FIFO metrics**: Real-time streaming metrics
- 💾 **Low overhead**: Minimal memory footprint

**Limitations**:
- Limited device support (no GPU, USB)
- Network requires TAP configuration
- Best for ephemeral workloads

**Configuration**:
```bash
export UNKEY_METALD_BACKEND=firecracker
./build/metald
```

### Cloud Hypervisor
**Use Cases**: Long-running VMs, development environments, full feature support

**Strengths**:
- 🖥️ **Rich features**: Full device support
- 🔄 **VM lifecycle**: Full pause/resume support
- 🌐 **Networking**: Built-in network support
- 📱 **Live migration**: Support for VM migration

**Limitations**:
- Slower startup time (~1-2 seconds)
- Higher memory overhead
- More complex setup

**Configuration**:
```bash
export UNKEY_METALD_BACKEND=cloudhypervisor
./build/metald -socket /tmp/ch.sock
```

## Feature Matrix

| Feature | Firecracker | Cloud Hypervisor |
|---------|-------------|------------------|
| Startup Time | <100ms | ~1-2s |
| Memory Overhead | ~5MB | ~20MB |
| CPU Support | x86_64, aarch64 | x86_64, aarch64 |
| Network | TAP required | Built-in |
| Block Storage | ✅ | ✅ |
| Live Migration | ❌ | ✅ |
| Pause/Resume | ❌ | ✅ |
| GPU Passthrough | ❌ | ✅ |
| FIFO Metrics | ✅ | ❌ |
| Billing Integration | ✅ Full | ✅ Limited |

## Choosing a Backend

**Choose Firecracker if:**
- You need fast startup times
- Running short-lived workloads
- Security is a top priority
- You need real-time billing metrics

**Choose Cloud Hypervisor if:**
- You need full VM features
- Running long-lived applications
- You need GPU or advanced device support
- Live migration is required

## Backend Configuration

### Environment Variables
```bash
# Backend selection
UNKEY_METALD_BACKEND=firecracker|cloudhypervisor

# Backend-specific endpoints
UNKEY_METALD_CH_ENDPOINT=unix:///tmp/ch.sock
UNKEY_METALD_FC_ENDPOINT=unix:///tmp/firecracker.sock
```

### VM Asset Requirements
Both backends require:
- Kernel: `/opt/vm-assets/vmlinux`
- Root filesystem: `/opt/vm-assets/rootfs.ext4`

### Firecracker Setup
```bash
# Download and setup Firecracker binary
curl -L https://github.com/firecracker-microvm/firecracker/releases/latest/download/firecracker-v1.4.1-x86_64.tgz | tar -xz
sudo mv firecracker-v1.4.1-x86_64/firecracker-v1.4.1-x86_64 /usr/local/bin/firecracker
```

### Cloud Hypervisor Setup
```bash
# Start Cloud Hypervisor daemon
cloud-hypervisor --api-socket /tmp/ch.sock
```

## Switching Backends

The unified API means switching backends requires only changing the environment variable:

```bash
# Stop current service
pkill metald

# Switch to different backend
export UNKEY_METALD_BACKEND=firecracker  # or cloudhypervisor

# Restart service
./build/metald
```

All existing API calls work identically across backends.
