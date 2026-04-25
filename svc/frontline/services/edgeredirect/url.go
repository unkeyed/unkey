package edgeredirect

import (
	"net"
	"net/http"
	"strings"
)

// scheme returns the request scheme. http.Request.URL.Scheme is empty on
// the server side, so we infer from req.TLS the way most server-side code
// does (matches frontline's existing pattern in services/proxy/director).
func scheme(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

// splitHost separates the host and port in a Host header. Returns the
// untouched header as the host and an empty port when there is none.
// Mirrors svc/frontline/services/proxy/hostname.ExtractHostname but also
// returns the port so callers that need to preserve it can.
func splitHost(hostHeader string) (host, port string) {
	h, p, err := net.SplitHostPort(hostHeader)
	if err != nil {
		return hostHeader, ""
	}
	return h, p
}

// joinHostPort re-attaches a port to a host, choosing the IPv6-bracketed
// form when needed. When port is empty, IPv6 hosts (which contain a colon)
// still need brackets to be valid in a URL authority.
func joinHostPort(host, port string) string {
	if port == "" {
		if strings.Contains(host, ":") {
			return "[" + host + "]"
		}
		return host
	}
	return net.JoinHostPort(host, port)
}

// hasWWWPrefix reports whether host begins with the (case-insensitive)
// "www." label. Hostnames are ASCII or punycode, so a byte-level
// case-fold via EqualFold on the first four bytes is correct and
// allocation-free.
func hasWWWPrefix(host string) bool {
	if len(host) < 4 {
		return false
	}
	return strings.EqualFold(host[:4], "www.")
}
