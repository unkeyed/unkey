package routing

import (
	"net"
	"net/http"
)

func ExtractHostname(req *http.Request) string {
	// Strip port from hostname for database lookup (Host header may include port)
	hostname, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		// If SplitHostPort fails, req.Host doesn't contain a port, use it as-is
		hostname = req.Host
	}

	return hostname
}
