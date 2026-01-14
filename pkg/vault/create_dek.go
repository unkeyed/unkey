package vault

import (
	"context"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
)

func (s *Service) CreateDEK(ctx context.Context, keyring string) (string, error) {
	ctx, span := tracing.Start(ctx, "vault.CreateDEK")
	defer span.End()

	key, err := s.keyring.CreateKey(ctx, keyring)
	if err != nil {
		return "", err
	}
	return key.GetId(), nil
}
