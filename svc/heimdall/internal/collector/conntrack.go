//go:build linux

package collector

import (
	"fmt"
	"net"

	ct "github.com/ti-mo/conntrack"
	"github.com/unkeyed/unkey/pkg/logger"
)

// podEgress holds aggregated egress bytes for a single pod.
type podEgress struct {
	totalBytes  int64
	publicBytes int64
}

// collectEgress reads the kernel conntrack table and returns per-pod egress
// bytes, split into total and public (non-RFC1918) traffic.
// podIPs maps pod IP → pod name for attribution.
func collectEgress(podIPs map[string]string, internalCIDRs []*net.IPNet) (map[string]podEgress, error) {
	conn, err := ct.Dial(nil)
	if err != nil {
		return nil, fmt.Errorf("opening conntrack: %w", err)
	}
	defer func() { _ = conn.Close() }()

	flows, err := conn.Dump(&ct.DumpOptions{ZeroCounters: false})
	if err != nil {
		return nil, fmt.Errorf("dumping conntrack: %w", err)
	}

	result := make(map[string]podEgress)

	for _, flow := range flows {
		srcIP := flow.TupleOrig.IP.SourceAddress.String()
		dstIP := flow.TupleOrig.IP.DestinationAddress.String()

		podName, isPod := podIPs[srcIP]
		if !isPod {
			continue
		}

		bytes := int64(flow.CountersOrig.Bytes)
		if bytes <= 0 {
			continue
		}

		eg := result[podName]
		eg.totalBytes += bytes

		if !isInternalIP(net.ParseIP(dstIP), internalCIDRs) {
			eg.publicBytes += bytes
		}

		result[podName] = eg
	}

	return result, nil
}

// isInternalIP checks if an IP falls within any of the internal CIDRs
// (pod CIDR, service CIDR, node CIDR, RFC1918).
func isInternalIP(ip net.IP, cidrs []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// defaultInternalCIDRs returns RFC1918 + link-local CIDRs.
// These cover the vast majority of in-cluster traffic.
func defaultInternalCIDRs() []*net.IPNet {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"100.64.0.0/10", // CGNAT, used by some cloud providers for internal routing
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
