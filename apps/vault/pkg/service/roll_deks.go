package service

import (
	"context"
	"fmt"
)

func (s *Service) RollDeks(ctx context.Context) error {

	lookupKeys, err := s.storage.ListObjectKeys(ctx, "keyring/")
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	for _, objectKey := range lookupKeys {
		b, err := s.storage.GetObject(ctx, objectKey)
		if err != nil {
			return fmt.Errorf("failed to get object: %w", err)
		}
		dek, kekID, err := s.keyring.DecodeAndDecryptKey(ctx, b)
		if err != nil {
			return fmt.Errorf("failed to decode and decrypt key: %w", err)
		}
		if kekID == s.encryptionKey.Id {
			s.logger.Info().Str("keyId", dek.Id).Msg("key already encrypted with latest kek")
			continue
		}
		reencrypted, err := s.keyring.EncryptAndEncodeKey(ctx, dek)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt key: %w", err)
		}
		err = s.storage.PutObject(ctx, objectKey, reencrypted)
		if err != nil {
			return fmt.Errorf("failed to put re-encrypted key: %w", err)
		}
	}

	return nil
}
