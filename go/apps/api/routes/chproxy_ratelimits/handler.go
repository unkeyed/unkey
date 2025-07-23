package chproxyRatelimits

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Handler handles ratelimit events for ClickHouse proxy
type Handler struct {
	ClickHouse clickhouse.ClickHouse
	Logger     logging.Logger
	Token      string
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/_internal/chproxy/ratelimits"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// Authenticate using Bearer token
	token, err := zen.Bearer(s)
	if err != nil {
		return err
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(h.Token)) != 1 {
		return fault.New("invalid chproxy token",
			fault.Code(codes.Auth.Authentication.KeyNotFound.URN()),
			fault.Internal("chproxy token does not match"),
			fault.Public("The provided token is invalid."))
	}

	events, err := zen.BindBody[[]schema.RatelimitRequestV1](s)
	if err != nil {
		return err
	}

	// Record metrics
	metrics.ChproxyRequestsTotal.WithLabelValues("ratelimits").Inc()
	metrics.ChproxyRowsTotal.WithLabelValues("ratelimits").Add(float64(len(events)))

	// Buffer all events to ClickHouse
	for _, event := range events {
		h.ClickHouse.BufferRatelimit(event)
	}

	return s.JSON(http.StatusOK, map[string]string{"status": "OK"})
}
