package vaultreencryption

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func (h *Handler) reencryptDeployment(ctx context.Context, keyring string, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return ciphertext, nil
	}
	var config ctrlv1.SecretsConfig
	if err := protojson.Unmarshal(ciphertext, &config); err != nil {
		return nil, err
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
		return nil, err
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
	for chunk := range slices.Chunk(values, batchSize) {
		items := make(map[string]*vaultv1.DecryptRequest, len(chunk))
		for i, value := range chunk {
			items[strconv.Itoa(i)] = &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: *value}
		}
		res, err := h.vault.ReEncryptBulk(ctx, &vaultv1.ReEncryptBulkRequest{Items: items})
		if err != nil {
			return false, err
		}
		for i, value := range chunk {
			item := res.GetItems()[strconv.Itoa(i)]
			if item == nil {
				return false, fmt.Errorf("Vault did not re-encrypt the item")
			}
			changed = changed || *value != item.GetEncrypted()
			*value = item.GetEncrypted()
		}
	}
	return changed, nil
}
