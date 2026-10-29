package vaultreencryption

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *Handler) encryptedKeys(ctx context.Context, after cursor) (cursor, error) {
	rows, err := h.db.ListVaultEncryptedKeys(ctx, db.ListVaultEncryptedKeysParams{
		AfterPk: after.Pk, Limit: batchSize,
	})
	if err != nil {
		return after, fmt.Errorf("failed to list rows")
	}
	if len(rows) == 0 {
		return cursor{Pk: 0}, nil
	}
	items := make(map[string]*vaultv1.DecryptRequest, len(rows))
	for _, row := range rows {
		items[strconv.FormatUint(row.Pk, 10)] = &vaultv1.DecryptRequest{Keyring: row.Keyring, Encrypted: row.Ciphertext}
	}
	res, err := h.vault.ReEncryptBulk(ctx, &vaultv1.ReEncryptBulkRequest{Items: items})
	if err != nil {
		return after, fmt.Errorf("failed to re-encrypt rows")
	}
	type update struct {
		Keyring    string `json:"keyring"`
		Ciphertext string `json:"ciphertext"`
		KeyID      string `json:"key_id"`
	}
	updates := make(map[uint64]update, len(rows))
	pks := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return after, err
		}
		item := res.GetItems()[strconv.FormatUint(row.Pk, 10)]
		if item == nil {
			return after, fmt.Errorf("missing re-encrypted value for pk %d", row.Pk)
		}
		updates[row.Pk] = update{
			Keyring:    row.Keyring,
			Ciphertext: item.GetEncrypted(), KeyID: item.GetKeyId(),
		}
		pks = append(pks, row.Pk)
	}
	payload, err := json.Marshal(updates)
	if err != nil {
		return after, fmt.Errorf("failed to encode updates")
	}
	_, err = h.db.UpdateVaultEncryptedKeys(ctx, db.UpdateVaultEncryptedKeysParams{
		Updates: payload, Updates_2: payload, Pks: pks, Updates_3: payload,
	})
	if err != nil {
		return after, fmt.Errorf("failed to update rows")
	}
	if len(rows) < batchSize {
		return cursor{Pk: 0}, nil
	}
	return cursor{Pk: rows[len(rows)-1].Pk}, nil
}
