package firecracker

import (
	"context"
	"net"
	"strings"
	"testing"

	"log/slog"

	"github.com/unkeyed/unkey/go/deploy/metald/internal/network"
)

// TestBuildNetworkKernelArgs tests the network kernel arguments builder
func TestBuildNetworkKernelArgs(t *testing.T) {
	// Create a mock logger
	logger := slog.Default()

	// Create a test SDKClientV4 with network config enabled
	client := &SDKClientV4{
		logger:                    logger,
		enableKernelNetworkConfig: true,
	}

	ctx := context.Background()

	// Test case 1: Basic network configuration
	t.Run("BasicNetworkConfig", func(t *testing.T) {
		networkInfo := &network.VMNetwork{
			VMID:       "test-vm-1",
			IPAddress:  net.ParseIP("192.168.1.100"),
			Gateway:    net.ParseIP("192.168.1.1"),
			Netmask:    net.IPv4Mask(255, 255, 255, 0),
			DNSServers: []string{"8.8.8.8", "1.1.1.1"},
		}

		args := client.buildNetworkKernelArgs(ctx, networkInfo)

		// Verify we get arguments
		if len(args) == 0 {
			t.Fatal("Expected network kernel arguments, got none")
		}

		// Join args to check content
		argsStr := strings.Join(args, " ")
		t.Logf("Generated kernel args: %s", argsStr)

		// Check for correct Firecracker IP format: ip=G::T:GM::GI:off
		// G=192.168.1.100, T=192.168.1.99, GM=255.255.255.0, GI=eth0
		expectedIPArg := "ip=192.168.1.100::192.168.1.99:255.255.255.0:eth0:off"
		if !strings.Contains(argsStr, expectedIPArg) {
			t.Errorf("Expected IP argument %s, got %s", expectedIPArg, argsStr)
		}

		// Check for DNS servers
		if !strings.Contains(argsStr, "nameserver=8.8.8.8") {
			t.Error("Expected primary DNS server in kernel args")
		}

		if !strings.Contains(argsStr, "nameserver1=1.1.1.1") {
			t.Error("Expected secondary DNS server in kernel args")
		}
	})

	// Test case 2: Network config with routes
	t.Run("NetworkConfigWithRoutes", func(t *testing.T) {
		_, destNet, _ := net.ParseCIDR("10.0.0.0/8")
		routes := []network.Route{
			{
				Destination: destNet,
				Gateway:     net.ParseIP("192.168.1.254"),
				Metric:      100,
			},
		}

		networkInfo := &network.VMNetwork{
			VMID:      "test-vm-2",
			IPAddress: net.ParseIP("192.168.1.200"),
			Gateway:   net.ParseIP("192.168.1.1"),
			Netmask:   net.IPv4Mask(255, 255, 255, 0),
			Routes:    routes,
		}

		args := client.buildNetworkKernelArgs(ctx, networkInfo)
		argsStr := strings.Join(args, " ")
		t.Logf("Generated kernel args with routes: %s", argsStr)

		// Check for route configuration
		if !strings.Contains(argsStr, "route=") {
			t.Error("Expected route configuration in kernel args")
		}
	})

	// Test case 3: IPv6 configuration
	t.Run("IPv6Config", func(t *testing.T) {
		networkInfo := &network.VMNetwork{
			VMID:        "test-vm-3",
			IPAddress:   net.ParseIP("192.168.1.300"),
			Gateway:     net.ParseIP("192.168.1.1"),
			Netmask:     net.IPv4Mask(255, 255, 255, 0),
			IPv6Address: net.ParseIP("2001:db8::1"),
		}

		args := client.buildNetworkKernelArgs(ctx, networkInfo)
		argsStr := strings.Join(args, " ")
		t.Logf("Generated kernel args with IPv6: %s", argsStr)

		// Check for IPv6 configuration
		if !strings.Contains(argsStr, "ipv6=2001:db8::1") {
			t.Error("Expected IPv6 configuration in kernel args")
		}
	})

	// Test case 4: VLAN configuration
	t.Run("VLANConfig", func(t *testing.T) {
		networkInfo := &network.VMNetwork{
			VMID:      "test-vm-4",
			IPAddress: net.ParseIP("192.168.1.400"),
			Gateway:   net.ParseIP("192.168.1.1"),
			Netmask:   net.IPv4Mask(255, 255, 255, 0),
			VLANID:    100,
		}

		args := client.buildNetworkKernelArgs(ctx, networkInfo)
		argsStr := strings.Join(args, " ")
		t.Logf("Generated kernel args with VLAN: %s", argsStr)

		// Check for VLAN configuration
		if !strings.Contains(argsStr, "vlan=100") {
			t.Error("Expected VLAN configuration in kernel args")
		}
	})

	// Test case 5: Realistic /29 subnet (like user's example)
	t.Run("Slash29Subnet", func(t *testing.T) {
		// Simulate user's actual network: 172.16.2.175/29 with TAP at 172.16.2.174
		networkInfo := &network.VMNetwork{
			VMID:      "test-vm-29",
			IPAddress: net.ParseIP("172.16.2.175"),
			Gateway:   net.ParseIP("172.16.2.1"),
			Netmask:   net.CIDRMask(29, 32), // /29 = 255.255.255.248
		}

		args := client.buildNetworkKernelArgs(ctx, networkInfo)
		argsStr := strings.Join(args, " ")
		t.Logf("Generated /29 subnet args: %s", argsStr)

		// Should generate: ip=172.16.2.175::172.16.2.174:255.255.255.248:eth0:off
		expectedIPArg := "ip=172.16.2.175::172.16.2.174:255.255.255.248:eth0:off"
		if !strings.Contains(argsStr, expectedIPArg) {
			t.Errorf("Expected IP argument %s, got %s", expectedIPArg, argsStr)
		}
	})

	// Test case 6: Nil network info
	t.Run("NilNetworkInfo", func(t *testing.T) {
		args := client.buildNetworkKernelArgs(ctx, nil)
		if len(args) != 0 {
			t.Error("Expected no arguments for nil network info")
		}
	})
}

// TestBuildKernelArgsWithNetworkAndMetadata tests the comprehensive kernel args builder
func TestBuildKernelArgsWithNetworkAndMetadata(t *testing.T) {
	logger := slog.Default()

	// Test with network config enabled
	client := &SDKClientV4{
		logger:                    logger,
		enableKernelNetworkConfig: true,
	}

	ctx := context.Background()

	t.Run("NetworkConfigEnabled", func(t *testing.T) {
		networkInfo := &network.VMNetwork{
			VMID:       "test-vm-enabled",
			IPAddress:  net.ParseIP("192.168.1.100"),
			Gateway:    net.ParseIP("192.168.1.1"),
			Netmask:    net.IPv4Mask(255, 255, 255, 0),
			DNSServers: []string{"8.8.8.8"},
		}

		baseArgs := "console=ttyS0 reboot=k panic=1"
		result := client.buildKernelArgsWithNetworkAndMetadata(ctx, baseArgs, networkInfo, nil)

		t.Logf("Final kernel args: %s", result)

		// Should contain base args
		if !strings.Contains(result, "console=ttyS0") {
			t.Error("Expected base args to be preserved")
		}

		// Should contain network args
		if !strings.Contains(result, "ip=192.168.1.100") {
			t.Error("Expected network configuration in final args")
		}
	})

	// Test with network config disabled
	clientDisabled := &SDKClientV4{
		logger:                    logger,
		enableKernelNetworkConfig: false,
	}

	t.Run("NetworkConfigDisabled", func(t *testing.T) {
		networkInfo := &network.VMNetwork{
			VMID:      "test-vm-disabled",
			IPAddress: net.ParseIP("192.168.1.100"),
			Gateway:   net.ParseIP("192.168.1.1"),
			Netmask:   net.IPv4Mask(255, 255, 255, 0),
		}

		baseArgs := "console=ttyS0 reboot=k panic=1"
		result := clientDisabled.buildKernelArgsWithNetworkAndMetadata(ctx, baseArgs, networkInfo, nil)

		t.Logf("Final kernel args (disabled): %s", result)

		// Should contain base args
		if !strings.Contains(result, "console=ttyS0") {
			t.Error("Expected base args to be preserved")
		}

		// Should NOT contain network args
		if strings.Contains(result, "ip=192.168.1.100") {
			t.Error("Expected no network configuration when disabled")
		}
	})
}
