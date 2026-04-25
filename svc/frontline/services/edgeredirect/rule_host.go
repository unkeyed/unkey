package edgeredirect

import (
	"net/http"
	"strings"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
)

// applyHostRewrite matches an exact request host (case-insensitive) and
// returns the same URL with the host replaced. Path, query, and the
// inbound port are preserved. No suffix or wildcard matching: each
// rewrite is one-to-one between two specific hostnames.
func applyHostRewrite(req *http.Request, rule *edgeredirectv1.HostRewrite) (string, bool) {
	if rule == nil || rule.GetFrom() == "" || rule.GetTo() == "" {
		return "", false
	}

	host, port := splitHost(req.Host)
	if !strings.EqualFold(host, rule.GetFrom()) {
		return "", false
	}

	target := joinHostPort(rule.GetTo(), port)
	return buildLocation(scheme(req), target, req.URL.RequestURI()), true
}
