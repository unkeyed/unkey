package chproxyVerifications

import (
	"context"
	"crypto/subtle"
	"net/http"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"

	chproxyMetrics "github.com/unkeyed/unkey/svc/api/routes/chproxy_metrics"
)

// Handler handles key verification events for ClickHouse proxy
type Handler struct {
	ClickHouse clickhouse.ClickHouse
	Token      string
	Metrics    chproxyMetrics.Metrics
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

	events, err := zen.BindBody[[]schema.KeyVerification](s)
	if err != nil {
		return err
	}

	// Record metrics
	h.Metrics.RecordChproxyRequest("verifications")
	h.Metrics.RecordChproxyRows("verifications", float64(len(events)))

	// Buffer all events to ClickHouse
	for _, event := range events {
		h.ClickHouse.BufferKeyVerification(event)
	}

	return s.Send(http.StatusOK, nil)
}
