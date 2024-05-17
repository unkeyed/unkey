package service

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"google.golang.org/protobuf/proto"
)


func (s *Service) RollDeks(ctx context.Context) (error){


	if len(s.decryptionKeys) < 2{
		return fmt.Errorf("not enough master keys to roll, you need at least 2")
	}


	objectKeys, err := s.storage.ListObjects(ctx, "")

	if err != nil{
		return fmt.Errorf("failed to list objects: %w", err)
	}


	for _, objectKey := range objectKeys{
		s.logger.Info().Str("object", objectKey).Msg("rolling dek")
		dekB, err := s.storage.GetObject(ctx, objectKey)
		if err != nil{
			return fmt.Errorf("failed to get object: %w", err)
		}

		encryptedDek := &vaultv1.EncryptedDEK{}
		err = proto.Unmarshal(dekB, encryptedDek)
		if err != nil{
			return fmt.Errorf("failed to unmarshal dek: %w", err)
		}
		if encryptedDek.Encrypted.GetEncryptionKeyId() != s.encryptionKey.Id{
			s.logger.Info().Str("object", objectKey).Msg("dek already encrypted by latest key, skipping")
			continue
		}



		dek, err :=s.decryptDEK(ctx, encryptedDek)
		if err != nil{
			return fmt.Errorf("failed to decrypt dek: %w", err)
		}


		reEncryptedDek, err := s.encryptDEK(ctx, dek)
		if err != nil{
			return fmt.Errorf("failed to re-encrypt dek: %w", err)
		}

		b, err := proto.Marshal(reEncryptedDek)
		if err != nil{
			return fmt.Errorf("failed to marshal dek: %w", err)
		}
		err = s.storage.PutObject(ctx, objectKey, b)
		if err != nil{
			return fmt.Errorf("failed to put object: %w", err)
		}


	}
	return nil


}