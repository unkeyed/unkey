package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"connectrpc.com/connect"
	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
)

func main() {
	// Create ConnectRPC client
	client := vmprovisionerv1connect.NewVmServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)

	ctx := context.Background()

	// Example 1: Dual-Stack VM (both IPv4 and IPv6)
	dualStackVM := createDualStackVM(ctx, client)
	fmt.Printf("Created dual-stack VM: %s\n", dualStackVM)

	// Example 2: IPv6-only VM
	ipv6OnlyVM := createIPv6OnlyVM(ctx, client)
	fmt.Printf("Created IPv6-only VM: %s\n", ipv6OnlyVM)

	// Example 3: IPv4-only VM (legacy compatibility)
	ipv4OnlyVM := createIPv4OnlyVM(ctx, client)
	fmt.Printf("Created IPv4-only VM: %s\n", ipv4OnlyVM)
}

// createDualStackVM creates a VM with both IPv4 and IPv6 networking
func createDualStackVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient) string {
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 2,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 2147483648, // 2GB
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/opt/vm-assets/vmlinux",
			KernelArgs: "console=ttyS0 root=/dev/vda rw",
		},
		Storage: []*metaldv1.StorageDevice{
			{
				Id:           "rootfs",
				Path:         "/opt/vm-assets/rootfs.ext4",
				IsRootDevice: true,
				ReadOnly:     false,
			},
		},
		Network: []*metaldv1.NetworkInterface{
			{
				Id:            "eth0",
				InterfaceType: "virtio-net",
				Mode:          metaldv1.NetworkMode_NETWORK_MODE_DUAL_STACK,
				Ipv4Config: &metaldv1.IPv4Config{
					Dhcp: false, // Static configuration
					// Address will be auto-allocated by metald
				},
				Ipv6Config: &metaldv1.IPv6Config{
					Slaac:              false, // Static configuration
					PrivacyExtensions:  false, // Disabled for servers
					// Address will be auto-allocated by metald
				},
				// Optional rate limiting
				RxRateLimit: &metaldv1.RateLimit{
					Bandwidth: 100 * 1024 * 1024, // 100 Mbps
				},
				TxRateLimit: &metaldv1.RateLimit{
					Bandwidth: 100 * 1024 * 1024, // 100 Mbps
				},
			},
		},
		Metadata: map[string]string{
			"name":        "dual-stack-vm",
			"environment": "production",
			"network":     "dual-stack",
		},
	}

	req := connect.NewRequest(&metaldv1.CreateVmRequest{
		Config: config,
	})

	// Add authentication
	req.Header().Set("Authorization", "Bearer dev_customer_example")

	resp, err := client.CreateVm(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create dual-stack VM: %v", err)
	}

	return resp.Msg.VmId
}

// createIPv6OnlyVM creates a VM with only IPv6 networking
func createIPv6OnlyVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient) string {
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 1,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 1073741824, // 1GB
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/opt/vm-assets/vmlinux",
			KernelArgs: "console=ttyS0 root=/dev/vda rw ipv6.disable=0",
		},
		Storage: []*metaldv1.StorageDevice{
			{
				Id:           "rootfs",
				Path:         "/opt/vm-assets/rootfs.ext4",
				IsRootDevice: true,
				ReadOnly:     false,
			},
		},
		Network: []*metaldv1.NetworkInterface{
			{
				Id:            "eth0",
				InterfaceType: "virtio-net",
				Mode:          metaldv1.NetworkMode_NETWORK_MODE_IPV6_ONLY,
				// Only IPv6 configuration
				Ipv6Config: &metaldv1.IPv6Config{
					Slaac:             false,
					PrivacyExtensions: false,
					// Could specify custom DNS servers
					DnsServers: []string{
						"2606:4700:4700::1111", // Cloudflare
						"2606:4700:4700::1001",
					},
				},
			},
		},
		Metadata: map[string]string{
			"name":        "ipv6-only-vm",
			"environment": "production",
			"network":     "ipv6-only",
		},
	}

	req := connect.NewRequest(&metaldv1.CreateVmRequest{
		Config: config,
	})

	req.Header().Set("Authorization", "Bearer dev_customer_example")

	resp, err := client.CreateVm(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create IPv6-only VM: %v", err)
	}

	return resp.Msg.VmId
}

// createIPv4OnlyVM creates a VM with only IPv4 networking (legacy)
func createIPv4OnlyVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient) string {
	config := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 1,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 1073741824, // 1GB
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/opt/vm-assets/vmlinux",
			KernelArgs: "console=ttyS0 root=/dev/vda rw",
		},
		Storage: []*metaldv1.StorageDevice{
			{
				Id:           "rootfs",
				Path:         "/opt/vm-assets/rootfs.ext4",
				IsRootDevice: true,
				ReadOnly:     false,
			},
		},
		Network: []*metaldv1.NetworkInterface{
			{
				Id:            "eth0",
				InterfaceType: "virtio-net",
				Mode:          metaldv1.NetworkMode_NETWORK_MODE_IPV4_ONLY,
				// Only IPv4 configuration
				Ipv4Config: &metaldv1.IPv4Config{
					Dhcp: false,
					// Custom DNS servers
					DnsServers: []string{
						"8.8.8.8",
						"8.8.4.4",
					},
				},
			},
		},
		Metadata: map[string]string{
			"name":        "ipv4-only-vm",
			"environment": "legacy",
			"network":     "ipv4-only",
		},
	}

	req := connect.NewRequest(&metaldv1.CreateVmRequest{
		Config: config,
	})

	req.Header().Set("Authorization", "Bearer dev_customer_example")

	resp, err := client.CreateVm(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create IPv4-only VM: %v", err)
	}

	return resp.Msg.VmId
}

// Example cloud-init configuration for dual-stack
func generateCloudInitForDualStack() string {
	return `#cloud-config
network:
  version: 2
  ethernets:
    eth0:
      match:
        macaddress: "02:00:00:00:00:01"
      addresses:
        - 10.100.1.2/24
        - fd00::1:2/64
      gateway4: 10.100.0.1
      gateway6: fd00::1
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4
          - 2606:4700:4700::1111
          - 2606:4700:4700::1001
      ipv6-privacy: false

# Ensure both IPv4 and IPv6 are enabled
bootcmd:
  - echo 0 > /proc/sys/net/ipv6/conf/all/disable_ipv6
  - echo 0 > /proc/sys/net/ipv6/conf/default/disable_ipv6
  - echo 0 > /proc/sys/net/ipv6/conf/eth0/disable_ipv6
`
}

// Example systemd-networkd configuration for IPv6-only
func generateSystemdNetworkdIPv6Only() string {
	return `[Match]
MACAddress=02:00:00:00:00:02

[Network]
Address=fd00::1:2/64
Gateway=fd00::1
DNS=2606:4700:4700::1111
DNS=2606:4700:4700::1001

# Disable IPv4
LinkLocalAddressing=ipv6
IPv6AcceptRA=no
IPv6PrivacyExtensions=no

[IPv6AcceptRA]
UseDNS=no
UseAutonomousPrefix=no
UseOnLinkPrefix=no
`
}