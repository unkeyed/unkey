package handler

import (
	"context"
	"net"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
)

// Handler 308-redirects plain-HTTP requests to their https:// equivalent.
// Kept deliberately allocation-light: no router lookups, no observability
// middleware. Volume is tracked via the redirectsTotal Prometheus counter.
type Handler struct{}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

// 308 Permanent Redirect preserves the request method so non-GET requests
// aren't silently downgraded to GET by clients following the redirect.
// Any inbound port on the Host header is dropped so the target resolves
// to the default 443 instead of echoing back e.g. ":80".
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()

	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		// No port present in Host header — use it verbatim.
		host = req.Host
	}

	target := *req.URL
	target.Scheme = "https"
	target.Host = host
	sess.ResponseWriter().Header().Set("Location", target.String())

	redirectsTotal.Inc()
	return sess.Send(http.StatusPermanentRedirect, nil)
}
