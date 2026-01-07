package vault

import (
	"context"
)

func (s *Service) CreateDEK(ctx context.Context, keyring string) (string, error) {
	key, err := s.keyring.CreateKey(ctx, keyring)
	if err != nil {
		return "", err
	}
	return key.GetId(), nil
}
