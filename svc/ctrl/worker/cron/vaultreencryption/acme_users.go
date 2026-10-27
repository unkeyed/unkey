package vaultreencryption

import (
	"context"
	"fmt"
	"strconv"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *Handler) acmeUsers(ctx context.Context, batch *batch) {
	rows, err := h.db.ListVaultAcmeUsers(ctx, db.ListVaultAcmeUsersParams{
		AfterPk: batch.result.Cursor.Pk, Limit: batchSize,
	})
	if err != nil {
		batch.scanFailed()
		return
	}
	if len(rows) == 0 {
		batch.finish(0)
		return
	}
	items := make(map[string]*vaultv1.DecryptRequest, len(rows))
	for _, row := range rows {
		items[strconv.FormatUint(row.Pk, 10)] = &vaultv1.DecryptRequest{Keyring: row.Keyring, Encrypted: row.Ciphertext}
	}
	res, err := h.vault.ReEncryptBulk(ctx, &vaultv1.ReEncryptBulkRequest{Items: items})
	if err != nil {
		logger.Error("ciphertext batch re-encryption failed", "table", batch.table)
		batch.result.Failed++
		return
	}
	for _, row := range rows {
		if !batch.process(ctx, row.Pk, func() (int64, error) {
			item := res.GetItems()[strconv.FormatUint(row.Pk, 10)].GetSuccess()
			if item == nil {
				return 0, fmt.Errorf("Vault did not re-encrypt the item")
			}
			return h.db.UpdateVaultAcmeUser(ctx, db.UpdateVaultAcmeUserParams{
				Pk: row.Pk, Keyring: row.Keyring, CiphertextOld: row.Ciphertext, Ciphertext: item.GetEncrypted(),
			})
		}) {
			return
		}
	}
	batch.finish(len(rows))
}
