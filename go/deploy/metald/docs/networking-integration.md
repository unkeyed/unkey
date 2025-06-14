# Networking Integration Guide for Metald

## Overview

This guide explains how to integrate the networking subsystem into metald to enable network connectivity for Firecracker microVMs.

## Architecture

The networking system consists of:
- **Network Manager**: Manages VM networks, IP allocation, and namespace creation
- **IP Allocator**: Handles IP address allocation from configured subnets
- **TAP/Veth Setup**: Creates network interfaces and bridges
- **Network Namespaces**: Provides isolation between VMs
- **iptables NAT**: Enables internet connectivity

## Integration Steps

### 1. Initialize Network Manager

Add to `internal/config/config.go`:

```go
// NetworkConfig holds network-related configuration
type NetworkConfig struct {
    Enabled         bool     `envconfig:"UNKEY_METALD_NETWORK_ENABLED" default:"true"`
    BridgeName      string   `envconfig:"UNKEY_METALD_NETWORK_BRIDGE" default:"br-vms"`
    BridgeIP        string   `envconfig:"UNKEY_METALD_NETWORK_BRIDGE_IP" default:"10.100.0.1/16"`
    VMSubnet        string   `envconfig:"UNKEY_METALD_NETWORK_VM_SUBNET" default:"10.100.0.0/16"`
    DNSServers      []string `envconfig:"UNKEY_METALD_NETWORK_DNS" default:"8.8.8.8,8.8.4.4"`
    EnableRateLimit bool     `envconfig:"UNKEY_METALD_NETWORK_RATE_LIMIT" default:"true"`
    RateLimitMbps   int      `envconfig:"UNKEY_METALD_NETWORK_RATE_LIMIT_MBPS" default:"100"`
}
```

### 2. Update Process Manager

Modify `internal/process/manager.go`:

```go
import (
    "github.com/unkeyed/unkey/go/deploy/metald/internal/network"
)

type ProcessManager struct {
    // ... existing fields ...
    networkMgr *network.Manager
}

// In NewProcessManager:
func NewProcessManager(config *Config, logger *slog.Logger) (*ProcessManager, error) {
    // ... existing code ...
    
    // Initialize network manager if enabled
    var networkMgr *network.Manager
    if config.Network.Enabled {
        netConfig := &network.Config{
            BridgeName:      config.Network.BridgeName,
            BridgeIP:        config.Network.BridgeIP,
            VMSubnet:        config.Network.VMSubnet,
            DNSServers:      config.Network.DNSServers,
            EnableRateLimit: config.Network.EnableRateLimit,
            RateLimitMbps:   config.Network.RateLimitMbps,
        }
        
        var err error
        networkMgr, err = network.NewManager(logger, netConfig)
        if err != nil {
            return nil, fmt.Errorf("failed to initialize network manager: %w", err)
        }
    }
    
    return &ProcessManager{
        // ... existing fields ...
        networkMgr: networkMgr,
    }, nil
}
```

### 3. Update VM Creation Flow

In `createDedicatedProcess`:

```go
func (m *ProcessManager) createDedicatedProcess(ctx context.Context, vmID string) (*FirecrackerProcess, error) {
    // ... existing code ...
    
    // Setup networking if enabled
    var vmNet *network.VMNetwork
    if m.networkMgr != nil {
        var err error
        vmNet, err = m.networkMgr.CreateVMNetwork(ctx, vmID)
        if err != nil {
            return nil, fmt.Errorf("failed to create VM network: %w", err)
        }
        
        // Update jailer command to use network namespace
        cmd.Args = append(cmd.Args, "--netns", fmt.Sprintf("/var/run/netns/%s", vmNet.Namespace))
    }
    
    // ... continue with existing code ...
    
    process := &FirecrackerProcess{
        ID:          processID,
        VMID:        vmID,
        Cmd:         cmd,
        SocketPath:  socketPath,
        ChrootPath:  chrootPath,
        JailerID:    jailerID,
        NetworkInfo: vmNet, // Add this field to FirecrackerProcess
        startTime:   time.Now(),
    }
    
    return process, nil
}
```

### 4. Update Firecracker Configuration

In `internal/backend/firecracker/client.go`:

```go
func (c *Client) CreateVMWithID(ctx context.Context, config *vmprovisionerv1.VmConfig, vmID string) (string, error) {
    // ... existing machine config ...
    
    // Configure network interfaces if available
    if c.process != nil && c.process.NetworkInfo != nil {
        netConfig := c.vmConfigToFirecrackerNetwork(config, c.process.NetworkInfo)
        for _, iface := range netConfig {
            if err := c.configureNetworkInterface(ctx, iface); err != nil {
                return "", fmt.Errorf("failed to configure network interface %s: %w", iface.IfaceID, err)
            }
        }
    }
    
    // ... continue with existing code ...
}

func (c *Client) vmConfigToFirecrackerNetwork(config *vmprovisionerv1.VmConfig, vmNet *network.VMNetwork) []firecrackerNetworkInterface {
    var interfaces []firecrackerNetworkInterface
    
    // If no network config provided, create default
    if len(config.Network) == 0 {
        interfaces = append(interfaces, firecrackerNetworkInterface{
            IfaceID:     "eth0",
            HostDevName: vmNet.TapDevice,
            GuestMAC:    vmNet.MacAddress,
        })
    } else {
        // Use provided network config
        for _, netCfg := range config.Network {
            iface := firecrackerNetworkInterface{
                IfaceID:     netCfg.Id,
                HostDevName: vmNet.TapDevice,
                GuestMAC:    vmNet.MacAddress,
            }
            
            // Add rate limiting if specified
            if netCfg.RxRateLimit != nil {
                iface.RxRateLimiter = &rateLimiter{
                    Bandwidth: &tokenBucket{
                        Size:       netCfg.RxRateLimit.Bandwidth,
                        RefillTime: netCfg.RxRateLimit.RefillTime,
                    },
                }
            }
            
            interfaces = append(interfaces, iface)
        }
    }
    
    return interfaces
}
```

### 5. Cleanup on VM Deletion

Update the VM deletion flow:

```go
func (m *ProcessManager) ReleaseProcess(ctx context.Context, vmID string) error {
    // ... existing code ...
    
    // Clean up networking
    if m.networkMgr != nil {
        if err := m.networkMgr.DeleteVMNetwork(ctx, vmID); err != nil {
            m.logger.LogAttrs(ctx, slog.LevelWarn, "failed to delete VM network",
                slog.String("vm_id", vmID),
                slog.String("error", err.Error()),
            )
        }
    }
    
    // ... continue with existing cleanup ...
}
```

### 6. Guest OS Configuration

The guest OS needs to be configured to use the network. Options:

#### Option A: DHCP (requires DHCP server setup)
```bash
# In guest init script
dhclient eth0
```

#### Option B: Static IP via kernel cmdline
```go
// In VM boot configuration
bootConfig.KernelArgs += fmt.Sprintf(" ip=%s::%s:255.255.255.0::eth0:off",
    vmNet.IPAddress.String(),
    vmNet.Gateway.String(),
)
```

#### Option C: Cloud-init
```yaml
# /etc/cloud/cloud.cfg.d/network.cfg
network:
  version: 2
  ethernets:
    eth0:
      addresses:
        - 10.100.1.2/24
      gateway4: 10.100.0.1
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4
```

## Testing

### 1. Manual Testing

```bash
# Check bridge is created
ip link show br-vms

# Check iptables rules
iptables -t nat -L POSTROUTING -n -v

# List network namespaces
ip netns list

# Test connectivity from namespace
ip netns exec vm-<vmid> ping 10.100.0.1
ip netns exec vm-<vmid> ping 8.8.8.8
```

### 2. Integration Test

```go
func TestVMNetworking(t *testing.T) {
    // Create VM with networking
    config := &vmprovisionerv1.VmConfig{
        Cpu:    &vmprovisionerv1.CpuConfig{VcpuCount: 1},
        Memory: &vmprovisionerv1.MemoryConfig{SizeBytes: 134217728},
        Boot: &vmprovisionerv1.BootConfig{
            KernelPath: "/opt/vm-assets/vmlinux",
            KernelArgs: "console=ttyS0",
        },
        Network: []*vmprovisionerv1.NetworkInterface{
            {
                Id:            "eth0",
                InterfaceType: "virtio-net",
            },
        },
    }
    
    vmID, err := client.CreateVM(ctx, config)
    require.NoError(t, err)
    
    // Boot VM
    err = client.BootVM(ctx, vmID)
    require.NoError(t, err)
    
    // Wait for VM to get IP
    time.Sleep(5 * time.Second)
    
    // Get network stats
    stats, err := networkMgr.GetNetworkStats(vmID)
    require.NoError(t, err)
    assert.Greater(t, stats.RxPackets, uint64(0))
}
```

## Network Policies

For multi-tenant isolation, implement network policies:

```go
// Example: Allow only HTTP/HTTPS traffic
policy := &network.NetworkPolicy{
    VMID:       vmID,
    CustomerID: customerID,
    Rules: []network.FirewallRule{
        {
            Name:      "allow-http",
            Direction: "ingress",
            Protocol:  "tcp",
            Port:      80,
            Source:    "0.0.0.0/0",
            Action:    "allow",
        },
        {
            Name:      "allow-https",
            Direction: "ingress",
            Protocol:  "tcp",
            Port:      443,
            Source:    "0.0.0.0/0",
            Action:    "allow",
        },
        {
            Name:      "deny-all",
            Direction: "ingress",
            Source:    "0.0.0.0/0",
            Action:    "deny",
            Priority:  1000, // Lower priority
        },
    },
}

networkMgr.ApplyNetworkPolicy(vmNet, policy)
```

## Performance Tuning

### 1. Enable SR-IOV for high-performance networking
### 2. Use DPDK for packet processing
### 3. Configure CPU affinity for network interrupts
### 4. Enable jumbo frames if needed

## Monitoring

Add network metrics to the billing pipeline:

```go
// Collect network stats every 100ms
stats, err := networkMgr.GetNetworkStats(vmID)
if err == nil {
    metrics.NetworkRxBytes = stats.RxBytes
    metrics.NetworkTxBytes = stats.TxBytes
    metrics.NetworkRxPackets = stats.RxPackets
    metrics.NetworkTxPackets = stats.TxPackets
}
```

## Security Considerations

1. **Namespace Isolation**: Each VM runs in its own network namespace
2. **MAC Address Spoofing**: Prevented by bridge configuration
3. **IP Spoofing**: Prevented by iptables rules
4. **Rate Limiting**: Prevents DoS attacks
5. **VLAN Isolation**: Can be added for additional separation

## Troubleshooting

Common issues and solutions:

1. **No connectivity**: Check IP forwarding is enabled
2. **DNS not working**: Verify DNS servers in configuration
3. **Poor performance**: Check for rate limiting or CPU throttling
4. **Packet loss**: Monitor interface statistics for errors

This completes the networking implementation for microVMs, providing isolated, secure, and performant network connectivity.