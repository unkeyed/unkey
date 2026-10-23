package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/vault/internal/keyring"
)

func (s *Service) RollDeks(ctx context.Context) error {
	ctx, span := tracing.Start(ctx, "vault.RollDeks")
	defer span.End()
	lookupKeys, err := s.storage.ListObjectKeys(ctx, "keyring/")
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	for _, objectKey := range lookupKeys {
		if err := s.keyring.RewrapKey(ctx, objectKey); err != nil && !errors.Is(err, keyring.ErrDeleted) {
			return err
		}
	}

	return nil
}
