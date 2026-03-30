//go:build !linux

package collector

import (
	"net"

	"github.com/unkeyed/unkey/pkg/logger"
)

// podEgress holds aggregated egress bytes for a single pod.
type podEgress struct {
	totalBytes  int64
	publicBytes int64
}

// collectEgress is a no-op on non-Linux platforms.
// Conntrack requires the Linux netfilter subsystem.
func collectEgress(_ map[string]string, _ []*net.IPNet) (map[string]podEgress, error) {
	return make(map[string]podEgress), nil
}

func defaultInternalCIDRs() []*net.IPNet {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"100.64.0.0/10",
	}

	var nets []*net.IPNet
	for _, cidr := range cidrs {
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			logger.Error("failed to parse internal CIDR", "cidr", cidr, "error", err.Error())
			continue
		}
		nets = append(nets, n)
	}
	return nets
}
