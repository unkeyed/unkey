package vault

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
)

func (s *Service) RunKeyringCleanup(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		if err := s.keyring.SweepDeleted(ctx); err != nil && ctx.Err() == nil {
			logger.Error("unable to clean up deleted keyrings", "error", err.Error())
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
