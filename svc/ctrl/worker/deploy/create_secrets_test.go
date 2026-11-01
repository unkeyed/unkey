package deploy

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

type connectionVault struct {
	vault.VaultServiceClient
	request *vaultv1.EncryptBulkRequest
	err     error
	omit    bool
}

func (v *connectionVault) EncryptBulk(_ context.Context, request *vaultv1.EncryptBulkRequest) (*vaultv1.EncryptBulkResponse, error) {
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

func TestBuildSecretsBlobConnectionHostsOverrideUserValues(t *testing.T) {
	v := &connectionVault{}
	w := &Workflow{vault: v}
	target := connectionTarget()
	deploymentID := uid.New(uid.DeploymentPrefix)

	blob, err := w.buildSecretsBlob(t.Context(), deploymentID, target,
		[]db.FindAppEnvVarsByAppAndEnvRow{
			{Key: "EXISTING", Value: "encrypted-existing"},
			{Key: "DATABASE_HOST", Value: "encrypted-stale-host"},
		},
		[]db.ListAppConnectionsByAppRow{{Name: "database"}, {Name: "event-store"}},
	)
	require.NoError(t, err)

	require.Equal(t, target.EnvironmentID, v.request.GetKeyring())
	databaseHost := privatenetwork.ConnectionHost(deploymentID, "database", target.K8sNamespace)
	require.Equal(t, map[string]string{
		"DATABASE_HOST":    databaseHost,
		"EVENT_STORE_HOST": privatenetwork.ConnectionHost(deploymentID, "event-store", target.K8sNamespace),
	}, v.request.GetItems())

	config := &ctrlv1.SecretsConfig{}
	require.NoError(t, protojson.Unmarshal(blob, config))
	require.Equal(t, map[string]string{
		"EXISTING":         "encrypted-existing",
		"DATABASE_HOST":    "ciphertext-for-DATABASE_HOST",
		"EVENT_STORE_HOST": "ciphertext-for-EVENT_STORE_HOST",
	}, config.GetSecrets())
	require.NotContains(t, string(blob), databaseHost)
}

func TestBuildSecretsBlobRejectsUnsafeConnections(t *testing.T) {
	tests := []struct {
		name       string
		workflow   *Workflow
		connection string
		namespace  *string
	}{
		{name: "reserved prefix", workflow: &Workflow{vault: &connectionVault{}}, connection: "unkey-internal"},
		{name: "leading digit", workflow: &Workflow{vault: &connectionVault{}}, connection: "1db"},
		{name: "missing namespace", workflow: &Workflow{vault: &connectionVault{}}, connection: "api", namespace: new("")},
		{name: "namespace not a label", workflow: &Workflow{vault: &connectionVault{}}, connection: "api", namespace: new("a.b")},
		{name: "missing vault", workflow: &Workflow{}, connection: "api"},
		{name: "vault failure", workflow: &Workflow{vault: &connectionVault{err: errors.New("unavailable")}}, connection: "api"},
		{name: "vault omission", workflow: &Workflow{vault: &connectionVault{omit: true}}, connection: "api"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := connectionTarget()
			if test.namespace != nil {
				target.K8sNamespace = *test.namespace
			}
			_, err := test.workflow.buildSecretsBlob(t.Context(), uid.New(uid.DeploymentPrefix), target, nil, []db.ListAppConnectionsByAppRow{{Name: test.connection}})
			require.Error(t, err)
		})
	}
}

func connectionTarget() db.FindDeployTargetRow {
	return db.FindDeployTargetRow{EnvironmentID: uid.New(uid.EnvironmentPrefix), K8sNamespace: uid.DNS1035()}
}
