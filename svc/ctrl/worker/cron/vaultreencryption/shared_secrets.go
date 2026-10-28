package vaultreencryption

import (
	"context"
	"strconv"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *Handler) sharedSecrets(ctx context.Context, after cursor) pageResult {
	result := pageResult{Cursor: after, Failed: 0}
	rows, err := h.db.ListVaultSharedSecrets(ctx, db.ListVaultSharedSecretsParams{
		AfterPk: after.Pk, Limit: batchSize,
	})
	if err != nil {
		logger.Error("ciphertext scan failed", "table", "shared_secrets")
		result.Failed++
		return result
	}
	if len(rows) == 0 {
		return pageResult{Cursor: cursor{Pk: 0}, Failed: 0}
	}
	items := make(map[string]*vaultv1.DecryptRequest, len(rows))
	for _, row := range rows {
		items[strconv.FormatUint(row.Pk, 10)] = &vaultv1.DecryptRequest{Keyring: row.Keyring, Encrypted: row.Ciphertext}
	}
	res, err := h.vault.ReEncryptBulk(ctx, &vaultv1.ReEncryptBulkRequest{Items: items})
	if err != nil {
		logger.Error("ciphertext batch re-encryption failed", "table", "shared_secrets")
		result.Failed++
		return result
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			result.Failed++
			return result
		}
		result.Cursor.Pk = row.Pk
		item := res.GetItems()[strconv.FormatUint(row.Pk, 10)]
		if item == nil {
			logger.Error("ciphertext re-encryption failed", "table", "shared_secrets", "pk", row.Pk)
			result.Failed++
			continue
		}
		_, err := h.db.UpdateVaultSharedSecret(ctx, db.UpdateVaultSharedSecretParams{
			Pk: row.Pk, Keyring: row.Keyring, CiphertextOld: row.Ciphertext,
			Ciphertext: item.GetEncrypted(), KeyID: item.GetKeyId(),
		})
		if err != nil {
			logger.Error("ciphertext re-encryption failed", "table", "shared_secrets", "pk", row.Pk)
			result.Failed++
		}
	}
	if len(rows) < batchSize {
		result.Cursor = cursor{Pk: 0}
	}
	return result
}
