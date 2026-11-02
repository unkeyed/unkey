package vaultreencryption

import (
	"context"
	"maps"
	"slices"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (h *Handler) reencrypt(ctx context.Context, req *vaultv1.DecryptRequest) (*vaultv1.EncryptResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	decrypted, err := h.vault.Decrypt(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return h.vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: req.GetKeyring(),
		Data:    decrypted.GetPlaintext(),
	})
}

func (h *Handler) reencryptDeployment(ctx context.Context, keyring string, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return ciphertext, nil
	}
	var config ctrlv1.SecretsConfig
	if err := protojson.Unmarshal(ciphertext, &config); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	names := slices.Sorted(maps.Keys(config.Secrets))
	values := make([]*string, 0, len(names))
	for _, name := range names {
		value := config.Secrets[name]
		values = append(values, &value)
	}
	changed, err := h.reencryptValues(ctx, keyring, values)
	if err != nil {
		return nil, err
	}
	if !changed {
		return ciphertext, nil
	}
	for i, name := range names {
		config.Secrets[name] = *values[i]
	}
	return protojson.Marshal(&config)
}

func (h *Handler) reencryptLogdrain(ctx context.Context, keyring string, ciphertext []byte) ([]byte, error) {
	var config logdrainv1.Config
	if err := proto.Unmarshal(ciphertext, &config); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	var values []*string
	if http := config.GetHttp(); http != nil {
		for _, header := range http.GetHeaders() {
			values = append(values, &header.EncryptedValue)
		}
	}
	if axiom := config.GetAxiom(); axiom != nil {
		values = append(values, &axiom.EncryptedToken)
	}
	changed, err := h.reencryptValues(ctx, keyring, values)
	if err != nil {
		return nil, err
	}
	if !changed {
		return ciphertext, nil
	}
	return proto.Marshal(&config)
}

func (h *Handler) reencryptValues(ctx context.Context, keyring string, values []*string) (bool, error) {
	changed := false
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		res, err := h.reencrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: *value,
		})
		if err != nil {
			return false, err
		}
		changed = changed || *value != res.GetEncrypted()
		*value = res.GetEncrypted()
	}
	return changed, nil
}

func skipRow(ctx context.Context, table string, pk uint64, err error) bool {
	if ctx.Err() != nil {
		return false
	}
	code := connect.CodeOf(err)
	if code == connect.CodeInternal || code == connect.CodeInvalidArgument {
		logger.Error("skipping row that could not be re-encrypted", "table", table, "pk", pk)
		return true
	}
	return false
}

func recordReEncryption(table string, updated int64, err error) {
	outcome := "updated"
	switch {
	case err != nil:
		outcome = "error"
	case updated == 0:
		outcome = "conflict"
	}
	metrics.VaultReEncryptionsDataTotal.WithLabelValues(table, outcome).Inc()
}
