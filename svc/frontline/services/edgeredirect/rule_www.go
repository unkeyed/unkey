package edgeredirect

import (
	"net/http"
	"strings"
)

// applyStripWWW matches hosts beginning with the (case-insensitive)
// "www." label and redirects to the same URL with that label removed.
// Only the first label is stripped — "www.www.example.com" becomes
// "www.example.com", not "example.com".
func applyStripWWW(req *http.Request) (string, bool) {
	host, port := splitHost(req.Host)
	if !hasWWWPrefix(host) {
		return "", false
	}

	target := joinHostPort(host[4:], port)
	return buildLocation(scheme(req), target, req.URL.RequestURI()), true
}

// applyAddWWW matches hosts that do NOT begin with "www." and redirects
// with that label prepended. Skips single-label hosts (e.g. "localhost")
// where prepending "www." is almost certainly wrong.
func applyAddWWW(req *http.Request) (string, bool) {
	host, port := splitHost(req.Host)
	if hasWWWPrefix(host) {
		return "", false
	}
	if !strings.Contains(host, ".") {
		return "", false
	}

	target := joinHostPort("www."+host, port)
	return buildLocation(scheme(req), target, req.URL.RequestURI()), true
}

func buildLocation(scheme, host, requestURI string) string {
	var b strings.Builder
	b.Grow(len(scheme) + len("://") + len(host) + len(requestURI))
	b.WriteString(scheme)
	b.WriteString("://")
	b.WriteString(host)
	b.WriteString(requestURI)
	return b.String()
}
