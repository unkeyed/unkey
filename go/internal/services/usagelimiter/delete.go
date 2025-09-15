package usagelimiter

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
)

// Invalidate doesn't do anything as we don't need to invalidate anything
func (s *service) Invalidate(ctx context.Context, keyID string) error {
	ctx, span := tracing.Start(ctx, "usagelimiter.Invalidate")
	defer span.End()
	return nil
}
