package email

import (
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
)

type noop struct{}

// NewNoop returns a Sender that logs the message instead of delivering it. Use
// it where no provider key is configured (local dev, CI) so callers can send
// unconditionally without nil checks.
func NewNoop() Sender { return noop{} }

func (noop) Send(_ context.Context, email Email) error {
	logger.Info("email not sent (noop sender)",
		"template_id", email.TemplateID,
		"to", email.To,
	)
	return nil
}
