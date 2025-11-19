package proxy

import "net"

// ExtractHostname extracts the hostname from a host string, stripping any port number.
// Examples:
//   - "example.com:443" -> "example.com"
//   - "example.com" -> "example.com"
//   - "[::1]:8080" -> "::1"
//   - "192.168.1.1:80" -> "192.168.1.1"
func ExtractHostname(host string) string {
	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		// No port present, return as-is
		return host
	}

	return hostname
}
