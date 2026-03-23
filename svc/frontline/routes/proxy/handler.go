package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
)

type Handler struct {
	RouterService router.Service
	ProxyService  proxy.Service
	Clock         clock.Clock
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	ctx = proxy.WithRequestStartTime(ctx, h.Clock.Now())
	hostname := proxy.ExtractHostname(sess.Request().Host)

	decision, err := h.RouterService.Route(ctx, hostname)
	if err != nil {
		return err
	}

	return h.ProxyService.Forward(ctx, sess, decision)
}
