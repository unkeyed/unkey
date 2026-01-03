package usagelimiter

import (
	"context"
)

// Invalidate doesn't do anything as we don't need to invalidate anything
func (s *service) Invalidate(ctx context.Context, keyID string) error {
	return nil
}
