package vaultreencryption

import (
	"context"
	"fmt"
	"strconv"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

func (h *Handler) encryptedKeys(ctx context.Context, after cursor) (cursor, error) {
	rows, err := h.db.ListVaultEncryptedKeys(ctx, db.ListVaultEncryptedKeysParams{
		AfterPk: after.Pk,
		Limit:   batchSize,
	})
	if err != nil {
		return after, fmt.Errorf("failed to list rows")
	}
	if len(rows) == 0 {
		return cursor{Pk: 0}, nil
	}
	items := make(map[string]*vaultv1.DecryptRequest, len(rows))
	for _, row := range rows {
		items[strconv.FormatUint(row.Pk, 10)] = &vaultv1.DecryptRequest{
			Keyring:   row.Keyring,
			Encrypted: row.Ciphertext,
		}
	}
	res, err := h.vault.ReEncryptBulk(ctx, &vaultv1.ReEncryptBulkRequest{Items: items})
	if err != nil {
		return after, fmt.Errorf("failed to re-encrypt rows")
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return after, err
		}
		item := res.GetItems()[strconv.FormatUint(row.Pk, 10)].GetSuccess()
		if item == nil {
			logger.Error("skipping row without a re-encrypted value", "table", "encrypted_keys", "pk", row.Pk)
			continue
		}
		updated, err := h.db.UpdateVaultEncryptedKey(ctx, db.UpdateVaultEncryptedKeyParams{
			Pk:            row.Pk,
			Keyring:       row.Keyring,
			CiphertextOld: row.Ciphertext,
			Ciphertext:    item.GetEncrypted(),
			KeyID:         item.GetKeyId(),
		})
		outcome := "updated"
		switch {
		case err != nil:
			outcome = "error"
		case updated == 0:
			outcome = "conflict"
		}
		metrics.VaultReEncryptionsDataTotal.WithLabelValues("encrypted_keys", outcome).Inc()
		if err != nil {
			return after, fmt.Errorf("failed to update pk %d", row.Pk)
		}
	}
	if len(rows) < batchSize {
		return cursor{Pk: 0}, nil
	}
	return cursor{Pk: rows[len(rows)-1].Pk}, nil
}
