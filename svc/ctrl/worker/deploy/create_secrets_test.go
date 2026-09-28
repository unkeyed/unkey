package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

type bindingVault struct {
	vault.VaultServiceClient
	request *vaultv1.EncryptBulkRequest
	err     error
	omit    bool
}

func (v *bindingVault) EncryptBulk(_ context.Context, request *vaultv1.EncryptBulkRequest) (*vaultv1.EncryptBulkResponse, error) {
	v.request = request
	if v.err != nil {
		return nil, v.err
	}
	items := make(map[string]*vaultv1.EncryptBulkResponseItem, len(request.GetItems()))
	if !v.omit {
		for key := range request.GetItems() {
			items[key] = &vaultv1.EncryptBulkResponseItem{Encrypted: "ciphertext-for-" + key}
		}
	}
	return &vaultv1.EncryptBulkResponse{Items: items}, nil
}

func TestBuildSecretsBlobInjectsEncryptedBindingHosts(t *testing.T) {
	v := &bindingVault{}
	w := &Workflow{vault: v}
	blob, err := w.buildSecretsBlob(t.Context(), "env_123",
		[]db.FindAppEnvVarsByAppAndEnvRow{{Key: "EXISTING", Value: "encrypted-existing"}},
		[]db.ListAppBindingsByAppRow{{Name: "database"}, {Name: "event-store"}},
	)
	require.NoError(t, err)
	require.Equal(t, "env_123", v.request.GetKeyring())
	require.Equal(t, map[string]string{
		"DATABASE_HOST":    "database.unkey.internal",
		"EVENT_STORE_HOST": "event-store.unkey.internal",
	}, v.request.GetItems())

	config := &ctrlv1.SecretsConfig{}
	require.NoError(t, protojson.Unmarshal(blob, config))
	require.Equal(t, "encrypted-existing", config.GetSecrets()["EXISTING"])
	require.Equal(t, "ciphertext-for-DATABASE_HOST", config.GetSecrets()["DATABASE_HOST"])
	require.NotContains(t, string(blob), "database.unkey.internal")
}

func TestBuildSecretsBlobRejectsUnsafeBindings(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		env      []db.FindAppEnvVarsByAppAndEnvRow
		binding  string
	}{
		{name: "reserved prefix", workflow: &Workflow{vault: &bindingVault{}}, binding: "unkey-internal"},
		{name: "leading digit", workflow: &Workflow{vault: &bindingVault{}}, binding: "1db"},
		{name: "collision", workflow: &Workflow{vault: &bindingVault{}}, env: []db.FindAppEnvVarsByAppAndEnvRow{{Key: "API_HOST", Value: "encrypted"}}, binding: "api"},
		{name: "missing vault", workflow: &Workflow{}, binding: "api"},
		{name: "vault failure", workflow: &Workflow{vault: &bindingVault{err: errors.New("unavailable")}}, binding: "api"},
		{name: "vault omission", workflow: &Workflow{vault: &bindingVault{omit: true}}, binding: "api"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.workflow.buildSecretsBlob(t.Context(), "env", test.env, []db.ListAppBindingsByAppRow{{Name: test.binding}})
			require.Error(t, err)
		})
	}
}
