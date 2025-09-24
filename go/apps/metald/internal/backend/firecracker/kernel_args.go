//go:build linux
// +build linux

package firecracker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/unkeyed/unkey/go/apps/metald/internal/network"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/builderd/v1"
)

const (
	// Static kernel parameters that never change for Firecracker VMs
	// AIDEV-BUSINESS_RULE: metald-init is always our init process
	staticKernelArgs = "console=ttyS0 reboot=k panic=1 pci=off init=/usr/bin/metald-init root=/dev/vda rw"

	// Debug parameters - only add when explicitly enabled
	debugKernelArgs = "loglevel=8 earlyprintk=serial,ttyS0,115200 debug ignore_loglevel printk.devkmsg=on"
)

// BuildKernelArgs generates kernel arguments for VM boot
func (c *Client) BuildKernelArgs(ctx context.Context, networkInfo *network.VMNetwork, metadata *builderv1.ImageMetadata) string {
	args := []string{staticKernelArgs}

	// Format: ip=G::T:GM::GI:off
	// G = Guest IP, T = TAP IP, GM = Guest Mask, GI = Guest Interface
	ipArg := fmt.Sprintf("ip=%s::%s:%s:%s:off",
		networkInfo.IPAddress,
		networkInfo.Gateway,
		networkInfo.Netmask,
		"eth0",
	)

	args = append(args, ipArg)

	// Add container metadata if available (metald-init will use these)
	if metadata != nil {
		// Add container environment variables (if needed)
		for key, value := range metadata.GetEnv() {
			// Skip PATH and anything with spaces to avoid kernel cmdline parsing issues
			if key == "PATH" || strings.Contains(key, " ") || strings.Contains(value, " ") {
				continue
			}
			args = append(args, fmt.Sprintf("env.%s=%s", key, value))
		}

		// Add working directory if specified
		if workdir := metadata.GetWorkingDir(); workdir != "" {
			args = append(args, fmt.Sprintf("workdir=%s", workdir))
		}
	}

	finalArgs := strings.Join(args, " ")

	c.logger.LogAttrs(ctx, slog.LevelDebug, "built kernel args",
		slog.String("vm_id", getVMID(networkInfo)),
		slog.Bool("has_network", networkInfo != nil),
		slog.Bool("has_metadata", metadata != nil),
		slog.String("args", finalArgs),
	)

	return finalArgs
}

func getVMID(networkInfo *network.VMNetwork) string {
	if networkInfo != nil {
		return networkInfo.VMID
	}
	return "unknown"
}
