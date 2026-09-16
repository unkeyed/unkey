package integration_test

import (
	"maps"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/internal/vault"
	"github.com/unkeyed/unkey/svc/vault/keys"
)

var observabilityRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(observabilityRegistry)
	os.Exit(m.Run())
}

// A new key needs two writes, and decrypt doesn't reuse the cached LATEST alias.
func TestS3Metrics_CountStorageOperationsForBulkRPCs(t *testing.T) {
	s3 := containers.S3(t)
	store, err := storage.NewS3(storage.S3Config{
		S3URL: s3.URL, S3Bucket: s3.CreateBucket(t),
		S3AccessKeyID: s3.AccessKeyID, S3AccessKeySecret: s3.SecretAccessKey,
	})
	require.NoError(t, err)
	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)
	v, err := vault.New(vault.Config{Storage: store, MasterKey: masterKey, BearerToken: "bearer"})
	require.NoError(t, err)

	s3Count := func(operation, outcome string) float64 {
		return counterValue(t, "unkey_vault_s3_operations_total", map[string]string{"operation": operation, "outcome": outcome, "error_code": ""})
	}
	reads, writes := s3Count("get", "success"), s3Count("put", "success")
	plaintexts := map[string]string{"one": "first secret", "two": "a longer second secret", "three": "third"}
	req := connect.NewRequest(&vaultv1.EncryptBulkRequest{Keyring: "ring", Items: plaintexts})
	req.Header().Set("Authorization", "Bearer bearer")
	encrypted, err := v.EncryptBulk(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, encrypted.Msg.GetItems(), 3)
	require.Equal(t, reads+1, s3Count("get", "success"))
	require.Equal(t, writes+2, s3Count("put", "success"))

	ciphertexts := map[string]string{}
	for id, item := range encrypted.Msg.GetItems() {
		ciphertexts[id] = item.GetEncrypted()
	}
	decryptReq := connect.NewRequest(&vaultv1.DecryptBulkRequest{Keyring: "ring", Items: ciphertexts})
	decryptReq.Header().Set("Authorization", "Bearer bearer")
	decrypted, err := v.DecryptBulk(t.Context(), decryptReq)
	require.NoError(t, err)
	require.Equal(t, plaintexts, decrypted.Msg.GetItems())
	require.Equal(t, reads+2, s3Count("get", "success"))
	require.Equal(t, writes+2, s3Count("put", "success"))
}

func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := observabilityRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			actualLabels := map[string]string{}
			for _, label := range metric.GetLabel() {
				actualLabels[label.GetName()] = label.GetValue()
			}
			if maps.Equal(labels, actualLabels) {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}
