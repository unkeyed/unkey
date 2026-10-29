package vaultreencryption

import (
	"context"
	"fmt"
	"strconv"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *Handler) certificates(ctx context.Context, after cursor) (cursor, error) {
	rows, err := h.db.ListVaultCertificates(ctx, db.ListVaultCertificatesParams{
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
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return after, err
		}
		item := res.GetItems()[strconv.FormatUint(row.Pk, 10)]
		if item == nil {
			return after, fmt.Errorf("missing re-encrypted value for pk %d", row.Pk)
		}
		_, err := h.db.UpdateVaultCertificate(ctx, db.UpdateVaultCertificateParams{
			Pk: row.Pk, Keyring: row.Keyring, CiphertextOld: row.Ciphertext, Ciphertext: item.GetEncrypted(),
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
