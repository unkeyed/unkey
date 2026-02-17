package cluster

import (
	"fmt"
	"net"

	"github.com/unkeyed/unkey/pkg/logger"
)

// ResolveDNSSeeds resolves a list of hostnames to "host:port" addresses.
// Hostnames that resolve to multiple A records (e.g. k8s headless services)
// produce one entry per IP. Literal IPs pass through unchanged.
func ResolveDNSSeeds(hosts []string, port int) []string {
	var addrs []string

	for _, host := range hosts {
		ips, err := net.LookupHost(host)
		if err != nil {
			logger.Warn("Failed to resolve seed host", "host", host, "error", err)
			// Use the raw host as fallback (might be an IP already)
			addrs = append(addrs, fmt.Sprintf("%s:%d", host, port))

			continue
		}

		for _, ip := range ips {
			addrs = append(addrs, fmt.Sprintf("%s:%d", ip, port))
		}
	}

	return addrs
}
