package vaultreencryption

import (
	"bytes"
	"context"

	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *Handler) logdrains(ctx context.Context, batch *batch) {
	rows, err := h.db.ListVaultLogdrains(ctx, db.ListVaultLogdrainsParams{
		AfterPk: batch.result.Cursor.Pk, Limit: batchSize,
	})
	if err != nil {
		batch.scanFailed()
		return
	}
	for _, row := range rows {
		if !batch.process(ctx, row.Pk, func() (int64, error) {
			ciphertext, err := h.reencryptLogdrain(ctx, row.Keyring, row.Ciphertext)
			if err != nil {
				return 0, err
			}
			if bytes.Equal(ciphertext, row.Ciphertext) {
				return 0, nil
			}
			return h.db.UpdateVaultLogdrain(ctx, db.UpdateVaultLogdrainParams{
				Pk: row.Pk, Keyring: row.Keyring, CiphertextOld: string(row.Ciphertext), Ciphertext: ciphertext,
			})
		}) {
			return
		}
	}
	batch.finish(len(rows))
}
