package bus

import (
	"fmt"
	"net"

	"github.com/unkeyed/unkey/pkg/logger"
)

// ResolveDNSSeeds resolves a list of hostnames to "host:port" addresses.
// Hostnames that resolve to multiple A records (e.g. k8s headless services
// or per-region NLBs) produce one entry per IP. Literal IPs and unresolved
// hostnames pass through with the supplied port appended so the caller does
// not need to know which is which.
//
// Failures fall back to the raw hostname so a temporarily-unhealthy NLB does
// not block startup; Serf's internal join retries will keep trying.
func ResolveDNSSeeds(hosts []string, port int) []string {
	var addrs []string

	for _, host := range hosts {
		if host == "" {
			continue
		}
		ips, err := net.LookupHost(host)
		if err != nil {
			logger.Warn("Failed to resolve seed host", "host", host, "error", err)
			addrs = append(addrs, fmt.Sprintf("%s:%d", host, port))
			continue
		}

		for _, ip := range ips {
			addrs = append(addrs, fmt.Sprintf("%s:%d", ip, port))
		}
	}

	return addrs
}
