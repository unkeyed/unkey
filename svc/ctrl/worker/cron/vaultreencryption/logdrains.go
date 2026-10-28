package vaultreencryption

import (
	"bytes"
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *Handler) logdrains(ctx context.Context, after cursor) pageResult {
	result := pageResult{Cursor: after, Failed: 0}
	rows, err := h.db.ListVaultLogdrains(ctx, db.ListVaultLogdrainsParams{
		AfterPk: after.Pk, Limit: batchSize,
	})
	if err != nil {
		logger.Error("ciphertext scan failed", "table", "logdrains")
		result.Failed++
		return result
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			result.Failed++
			return result
		}
		result.Cursor.Pk = row.Pk
		ciphertext, err := h.reencryptLogdrain(ctx, row.Keyring, row.Ciphertext)
		if err != nil {
			logger.Error("ciphertext re-encryption failed", "table", "logdrains", "pk", row.Pk)
			result.Failed++
			continue
		}
		if bytes.Equal(ciphertext, row.Ciphertext) {
			continue
		}
		_, err = h.db.UpdateVaultLogdrain(ctx, db.UpdateVaultLogdrainParams{
			Pk: row.Pk, Keyring: row.Keyring, CiphertextOld: string(row.Ciphertext), Ciphertext: ciphertext,
		})
		if err != nil {
			logger.Error("ciphertext re-encryption failed", "table", "logdrains", "pk", row.Pk)
			result.Failed++
		}
	}
	if len(rows) < batchSize {
		result.Cursor = cursor{Pk: 0}
	}
	return result
}
