package routing

import (
	"strings"
)

// isLocalHostname checks if a hostname should be skipped from gateway config creation.
//
// Returns true for localhost and development domains (.local, .test TLDs) that should
// not get gateway configurations. Hostnames using the default domain (e.g., *.unkey.app)
// return false, as they represent production/staging environments and need gateway configs.
func isLocalHostname(hostname, defaultDomain string) bool {
	hostname = strings.ToLower(hostname)
	defaultDomain = strings.ToLower(defaultDomain)

	// Exact matches for localhost
	if hostname == "localhost" || hostname == "127.0.0.1" {
		return true
	}

	// If hostname uses the default domain, it's NOT local
	if strings.HasSuffix(hostname, "."+defaultDomain) || hostname == defaultDomain {
		return false
	}

	// Check for local-only TLD suffixes
	localSuffixes := []string{".local", ".test"}
	for _, suffix := range localSuffixes {
		if strings.HasSuffix(hostname, suffix) {
			return true
		}
	}

	return false
}
