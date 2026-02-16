package chproxyVerifications

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Handler handles key verification events for ClickHouse proxy
type Handler struct {
	ClickHouse clickhouse.ClickHouse
	Token      string
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/_internal/chproxy/verifications"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	s.DisableClickHouseLogging()

	if err := zen.BearerTokenAuth(s, h.Token); err != nil {
		return err
	}

	events, err := zen.BindBody[[]schema.KeyVerification](s)
	if err != nil {
		return err
	}

	// Record metrics
	metrics.ChproxyRequestsTotal.WithLabelValues("verifications").Inc()
	metrics.ChproxyRowsTotal.WithLabelValues("verifications").Add(float64(len(events)))

	// Buffer all events to ClickHouse
	for _, event := range events {
		h.ClickHouse.BufferKeyVerification(event)
	}

	return s.Send(http.StatusOK, nil)
}
