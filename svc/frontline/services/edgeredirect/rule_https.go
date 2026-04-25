package edgeredirect

import (
	"net/http"
	"strings"
)

// applyRequireHTTPS matches plain-HTTP requests and returns the https://
// equivalent. The port from the inbound Host header is dropped — a request
// served on :80 redirects to the implicit :443, not :80 over TLS.
func applyRequireHTTPS(req *http.Request) (string, bool) {
	if req.TLS != nil {
		return "", false
	}

	host, _ := splitHost(req.Host)
	target := joinHostPort(host, "")

	var b strings.Builder
	b.Grow(len("https://") + len(target) + len(req.URL.RequestURI()))
	b.WriteString("https://")
	b.WriteString(target)
	b.WriteString(req.URL.RequestURI())
	return b.String(), true
}
