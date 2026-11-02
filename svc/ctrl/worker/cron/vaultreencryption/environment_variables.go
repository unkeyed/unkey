package vaultreencryption

import (
	"context"
	"fmt"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/vaultreencryption/internal/db"
)

func (h *Handler) environmentVariables(ctx context.Context, after cursor) (cursor, error) {
	rows, err := h.db.ListEnvironmentVariables(ctx, db.ListEnvironmentVariablesParams{
		AfterPk: after.Pk,
		Limit:   batchSize,
	})
	if err != nil {
		return after, fmt.Errorf("failed to list rows")
	}
	if len(rows) == 0 {
		return cursor{Pk: 0}, nil
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return after, err
		}
		item, err := h.reencrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   row.Keyring,
			Encrypted: row.Ciphertext,
		})
		if err != nil {
			if skipRow(ctx, "app_environment_variables", row.Pk, err) {
				continue
			}
			return after, fmt.Errorf("failed to re-encrypt pk %d", row.Pk)
		}
		updated, err := h.db.UpdateEnvironmentVariable(ctx, db.UpdateEnvironmentVariableParams{
			Pk:            row.Pk,
			Keyring:       row.Keyring,
			CiphertextOld: row.Ciphertext,
			Ciphertext:    item.GetEncrypted(),
		})
		recordReEncryption("app_environment_variables", updated, err)
		if err != nil {
			return after, fmt.Errorf("failed to update pk %d", row.Pk)
		}
	}
	if len(rows) < batchSize {
		return cursor{Pk: 0}, nil
	}
	return cursor{Pk: rows[len(rows)-1].Pk}, nil
}
