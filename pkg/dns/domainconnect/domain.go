package domainconnect

import (
	"strings"

	"golang.org/x/net/publicsuffix"
)

// normalizeDomain lowercases, trims whitespace, and strips a trailing dot (FQDN form).
func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimSuffix(domain, ".")
	return domain
}

// IsApexDomain returns true if the domain is a zone apex (eTLD+1), e.g.
// "example.com" is apex, "api.example.com" is not.
// Handles FQDN trailing dots and multi-part TLDs (e.g. "example.co.uk").
func IsApexDomain(domain string) bool {
	domain = normalizeDomain(domain)
	root, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return false
	}
	return domain == root
}
