package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
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
// The inbound port is stripped so the target uses the default 443.
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	req := sess.Request()

	target := *req.URL
	target.Scheme = "https"
	target.Host = proxy.ExtractHostname(req.Host)
	sess.ResponseWriter().Header().Set("Location", target.String())

	redirectsTotal.Inc()
	return sess.Send(http.StatusPermanentRedirect, nil)
}
