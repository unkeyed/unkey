package vaultreencryption

import (
	"bytes"
	"context"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *Handler) deployments(ctx context.Context, after cursor) pageResult {
	result := pageResult{Cursor: after, Failed: 0}
	rows, err := h.db.ListVaultDeployments(ctx, db.ListVaultDeploymentsParams{
		AfterPk: after.Pk, Limit: batchSize,
	})
	if err != nil {
		logger.Error("ciphertext scan failed", "table", "deployments")
		result.Failed++
		return result
	}
	for _, row := range rows {
		if ctx.Err() != nil {
			result.Failed++
			return result
		}
		result.Cursor.Pk = row.Pk
		ciphertext, err := h.reencryptDeployment(ctx, row.Keyring, row.Ciphertext)
		if err != nil {
			logger.Error("ciphertext re-encryption failed", "table", "deployments", "pk", row.Pk)
			result.Failed++
			continue
		}
		if bytes.Equal(ciphertext, row.Ciphertext) {
			continue
		}
		_, err = h.db.UpdateVaultDeployment(ctx, db.UpdateVaultDeploymentParams{
			Pk: row.Pk, Keyring: row.Keyring, CiphertextOld: string(row.Ciphertext), Ciphertext: ciphertext,
		})
		if err != nil {
			logger.Error("ciphertext re-encryption failed", "table", "deployments", "pk", row.Pk)
			result.Failed++
		}
	}
	if len(rows) < batchSize {
		result.Cursor = cursor{Pk: 0}
	}
	return result
}
