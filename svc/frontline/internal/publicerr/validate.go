package publicerr

import "strings"

// ForbiddenInPublicMessages lists substrings that must never appear in
// any string a customer can see — Title, Detail, TypeURL, or any
// per-request UserFacingMessage set on a fault. They identify internal
// frontline topology and failure mechanisms that the public surface
// deliberately abstracts away.
//
// Exported so both the catalog test and the proxy classification test
// share one source of truth; any new internal-only term should be added
// here once and is then enforced everywhere customer-facing prose is
// produced.
var ForbiddenInPublicMessages = []string{
	"peer_frontline",
	"peer-frontline",
	"upstream_connection",
	"upstream_host",
	"dial_timeout",
	"dns_timeout",
	"dns_not_found",
	"config_load",
	"instance_load",
	"deployment_selection",
	"no_reachable_region",
	"no_running_instances",
	"no_deployment_instances",
	"invalid_configuration",
	"proxy_error",
	"gateway_deadline",
}

// ContainsForbidden reports whether s contains any substring from
// ForbiddenInPublicMessages. Matching is case-insensitive. Returns the
// matched forbidden substring on hit (for assertion messages); empty
// string on clean.
func ContainsForbidden(s string) string {
	lower := strings.ToLower(s)
	for _, bad := range ForbiddenInPublicMessages {
		if strings.Contains(lower, bad) {
			return bad
		}
	}
	return ""
}
