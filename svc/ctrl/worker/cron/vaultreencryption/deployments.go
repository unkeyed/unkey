package vaultreencryption

import (
	"bytes"
	"context"
	"fmt"

	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/vaultreencryption/internal/db"
)

func (h *Handler) deployments(ctx context.Context, after cursor) (cursor, error) {
	rows, err := h.db.ListVaultDeployments(ctx, db.ListVaultDeploymentsParams{
		AfterPk: after.Pk,
		Limit:   batchSize,
	})
	if err != nil {
		return after, fmt.Errorf("failed to list rows")
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return after, err
		}
		ciphertext, err := h.reencryptDeployment(ctx, row.Keyring, row.Ciphertext)
		if err != nil {
			return after, fmt.Errorf("failed to re-encrypt pk %d", row.Pk)
		}
		if bytes.Equal(ciphertext, row.Ciphertext) {
			continue
		}
		_, err = h.db.UpdateVaultDeployment(ctx, db.UpdateVaultDeploymentParams{
			Pk:            row.Pk,
			Keyring:       row.Keyring,
			CiphertextOld: row.Ciphertext,
			Ciphertext:    ciphertext,
		})
		if err != nil {
			return after, fmt.Errorf("failed to update pk %d", row.Pk)
		}
	}
	if len(rows) < batchSize {
		return cursor{Pk: 0}, nil
	}
	return cursor{Pk: rows[len(rows)-1].Pk}, nil
}
