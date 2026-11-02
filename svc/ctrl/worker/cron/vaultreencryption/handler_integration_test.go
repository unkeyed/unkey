package vaultreencryption_test

import (
	"context"
	"encoding/base64"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/vaultreencryption"
	vaulttestutil "github.com/unkeyed/unkey/svc/vault/testutil"
	"google.golang.org/protobuf/proto"
)

// A single invocation must drain full and partial pages.
func TestReEncryptionDrainsTablesAndPreservesPlaintext(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	client := vault.NewConnectVaultServiceClient(testVault.Client)
	keyring := uid.New(uid.WorkspacePrefix)
	keyIDs := make([]string, 201)
	for i := range keyIDs {
		keyIDs[i] = uid.New(uid.KeyPrefix)
		encrypted, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    keyIDs[i],
		})
		require.NoError(t, err)
		require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
			WorkspaceID:     keyring,
			KeyID:           keyIDs[i],
			Encrypted:       encrypted.GetEncrypted(),
			EncryptionKeyID: encrypted.GetKeyId(),
			CreatedAt:       time.Now().UnixMilli(),
		}))
	}
	keyIDNew, err := testVault.CreateDEK(t.Context(), keyring)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)

	handler, err := vaultreencryption.New(vaultreencryption.Config{
		DB:        database,
		Vault:     client,
		Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)
	server := &cronService{handler: handler}
	runtime := containers.Restate(t, hydrav1.NewCronServiceServer(server,
		restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
	))
	cronClient := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption")
	res, err := cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.NoError(t, err)
	require.Zero(t, proto.Size(res), "the cron returns an empty response after re-encryption")

	_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.NoError(t, err)

	for _, keyID := range keyIDs {
		stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyID)
		require.NoError(t, err)
		require.Equal(t, keyIDNew, stored.EncryptionKeyID)
		decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: stored.Encrypted,
		})
		require.NoError(t, err)
		require.Equal(t, keyID, decrypted.GetPlaintext())
	}
}

// Inserts after a full page are scanned now; inserts after a short page wait
// for the next sweep.
func TestReEncryptionPaginatesConcurrentInserts(t *testing.T) {
	for _, tt := range []struct {
		name    string
		rows    int
		rotated bool
	}{
		{
			name:    "short page",
			rows:    99,
			rotated: false,
		},
		{
			name:    "full page",
			rows:    100,
			rotated: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.Close()) })
			testVault := vaulttestutil.StartTestVaultWithMemory(t)
			client := vault.NewConnectVaultServiceClient(testVault.Client)
			keyring := uid.New(uid.WorkspacePrefix)
			encrypted, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    "concurrent insert",
			})
			require.NoError(t, err)
			for range tt.rows {
				require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
					WorkspaceID:     keyring,
					KeyID:           uid.New(uid.KeyPrefix),
					Encrypted:       encrypted.GetEncrypted(),
					EncryptionKeyID: encrypted.GetKeyId(),
					CreatedAt:       time.Now().UnixMilli(),
				}))
			}
			keyIDNew, err := testVault.CreateDEK(t.Context(), keyring)
			require.NoError(t, err)
			testVault.Clock.Tick(25 * time.Hour)
			keyIDLater := uid.New(uid.KeyPrefix)
			concurrent := &insertingVault{
				VaultServiceClient: client,
				database:           database,
				key: pkgdb.InsertKeyEncryptionParams{
					WorkspaceID:     keyring,
					KeyID:           keyIDLater,
					Encrypted:       encrypted.GetEncrypted(),
					EncryptionKeyID: encrypted.GetKeyId(),
					CreatedAt:       time.Now().UnixMilli(),
				},
			}
			handler, err := vaultreencryption.New(vaultreencryption.Config{
				DB:        database,
				Vault:     concurrent,
				Heartbeat: healthcheck.NewNoop(),
			})
			require.NoError(t, err)
			runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
				restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
			))
			cronClient := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption")
			_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
			require.NoError(t, err)
			stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyIDLater)
			require.NoError(t, err)
			if tt.rotated {
				require.Equal(t, keyIDNew, stored.EncryptionKeyID)
			} else {
				require.Equal(t, encrypted.GetEncrypted(), stored.Encrypted)
			}

			_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
			require.NoError(t, err)
			stored, err = pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyIDLater)
			require.NoError(t, err)
			require.Equal(t, keyIDNew, stored.EncryptionKeyID)
			decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: stored.Encrypted,
			})
			require.NoError(t, err)
			require.Equal(t, "concurrent insert", decrypted.GetPlaintext())
		})
	}
}

func TestReEncryptionRotatesRecentCiphertext(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	client := vault.NewConnectVaultServiceClient(testVault.Client)
	keyring := uid.New(uid.WorkspacePrefix)
	encrypted, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    "old ciphertext on the current DEK",
	})
	require.NoError(t, err)
	encoded, err := base64.StdEncoding.DecodeString(encrypted.GetEncrypted())
	require.NoError(t, err)
	var envelope vaultv1.Encrypted
	require.NoError(t, proto.Unmarshal(encoded, &envelope))
	// Age is envelope metadata, not part of the authenticated plaintext.
	envelope.Time = time.Now().Add(-48 * time.Hour).UnixMilli()
	encoded, err = proto.Marshal(&envelope)
	require.NoError(t, err)
	ciphertextOld := base64.StdEncoding.EncodeToString(encoded)
	keyID := uid.New(uid.KeyPrefix)
	require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
		WorkspaceID:     keyring,
		KeyID:           keyID,
		Encrypted:       ciphertextOld,
		EncryptionKeyID: encrypted.GetKeyId(),
		CreatedAt:       envelope.GetTime(),
	}))
	keyringRecent := uid.New(uid.WorkspacePrefix)
	recent, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{
		Keyring: keyringRecent,
		Data:    "recent ciphertext",
	})
	require.NoError(t, err)
	keyIDRecent := uid.New(uid.KeyPrefix)
	require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
		WorkspaceID:     keyringRecent,
		KeyID:           keyIDRecent,
		Encrypted:       recent.GetEncrypted(),
		EncryptionKeyID: recent.GetKeyId(),
		CreatedAt:       time.Now().UnixMilli(),
	}))
	keyIDRecentNew, err := testVault.CreateDEK(t.Context(), keyringRecent)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)
	handler, err := vaultreencryption.New(vaultreencryption.Config{
		DB:        database,
		Vault:     client,
		Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)
	runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
		restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
	))
	cronClient := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption")
	_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.NoError(t, err)
	stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyID)
	require.NoError(t, err)
	require.Equal(t, encrypted.GetKeyId(), stored.EncryptionKeyID)
	decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: stored.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, "old ciphertext on the current DEK", decrypted.GetPlaintext())
	storedRecent, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyIDRecent)
	require.NoError(t, err)
	require.Equal(t, keyIDRecentNew, storedRecent.EncryptionKeyID)
	decrypted, err = client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
		Keyring:   keyringRecent,
		Encrypted: storedRecent.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, "recent ciphertext", decrypted.GetPlaintext())

	keyIDNew, err := testVault.CreateDEK(t.Context(), keyring)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)
	_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.NoError(t, err)
	stored, err = pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyID)
	require.NoError(t, err)
	require.Equal(t, keyIDNew, stored.EncryptionKeyID)
	decrypted, err = client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: stored.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, "old ciphertext on the current DEK", decrypted.GetPlaintext())
	storedRecentAgain, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyIDRecent)
	require.NoError(t, err)
	require.Equal(t, keyIDRecentNew, storedRecentAgain.EncryptionKeyID)
	decrypted, err = client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
		Keyring:   keyringRecent,
		Encrypted: storedRecentAgain.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, "recent ciphertext", decrypted.GetPlaintext())
}

func TestReEncryptionSkipsInvalidRowsAndContinuesPaging(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	client := vault.NewConnectVaultServiceClient(testVault.Client)
	keyringOld := uid.New(uid.WorkspacePrefix)
	old, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{
		Keyring: keyringOld,
		Data:    "beyond the first batch",
	})
	require.NoError(t, err)
	keyID := uid.New(uid.KeyPrefix)
	keyring := uid.New(uid.WorkspacePrefix)
	current, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    "current",
	})
	require.NoError(t, err)
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = uid.New(uid.KeyPrefix)
		ciphertext := current.GetEncrypted()
		if i == 0 {
			ciphertext = "invalid-base64"
		}
		require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
			WorkspaceID:     keyring,
			KeyID:           keys[i],
			Encrypted:       ciphertext,
			EncryptionKeyID: current.GetKeyId(),
			CreatedAt:       time.Now().UnixMilli(),
		}))
	}
	require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
		WorkspaceID:     keyringOld,
		KeyID:           keyID,
		Encrypted:       old.GetEncrypted(),
		EncryptionKeyID: old.GetKeyId(),
		CreatedAt:       time.Now().UnixMilli(),
	}))
	_, err = testVault.CreateDEK(t.Context(), keyringOld)
	require.NoError(t, err)
	keyIDCurrentNew, err := testVault.CreateDEK(t.Context(), keyring)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)
	handler, err := vaultreencryption.New(vaultreencryption.Config{
		DB:        database,
		Vault:     client,
		Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)
	runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
		restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
	))
	cronClient := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption")
	for range 2 {
		updatedBefore := metricCounter(t, "unkey_control_vault_reencryptions_data_total", prometheus.Labels{
			"table":   "encrypted_keys",
			"outcome": "updated",
		})
		skippedBefore := metricCounter(t, "unkey_control_vault_reencryptions_data_total", prometheus.Labels{
			"table":   "encrypted_keys",
			"outcome": "skipped",
		})
		_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
		require.NoError(t, err)
		require.Equal(t, updatedBefore+100, metricCounter(t, "unkey_control_vault_reencryptions_data_total", prometheus.Labels{
			"table":   "encrypted_keys",
			"outcome": "updated",
		}))
		require.Equal(t, skippedBefore, metricCounter(t, "unkey_control_vault_reencryptions_data_total", prometheus.Labels{
			"table":   "encrypted_keys",
			"outcome": "skipped",
		}))
		storedInvalid, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keys[0])
		require.NoError(t, err)
		require.Equal(t, "invalid-base64", storedInvalid.Encrypted)
		storedCurrent, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keys[1])
		require.NoError(t, err)
		require.Equal(t, keyIDCurrentNew, storedCurrent.EncryptionKeyID, "an invalid row must not block valid rows in the same page")
		decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: storedCurrent.Encrypted,
		})
		require.NoError(t, err)
		require.Equal(t, "current", decrypted.GetPlaintext())
		stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyID)
		require.NoError(t, err)
		require.NotEqual(t, old.GetKeyId(), stored.EncryptionKeyID, "an invalid row must not block later pages")
		decrypted, err = client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
			Keyring:   keyringOld,
			Encrypted: stored.Encrypted,
		})
		require.NoError(t, err)
		require.Equal(t, "beyond the first batch", decrypted.GetPlaintext())
	}
}

type insertingVault struct {
	vault.VaultServiceClient
	database db.Database
	key      pkgdb.InsertKeyEncryptionParams
	inserted bool
}

func (v *insertingVault) ReEncryptBulk(ctx context.Context, req *vaultv1.ReEncryptBulkRequest) (*vaultv1.ReEncryptBulkResponse, error) {
	if !v.inserted {
		if err := pkgdb.Query.InsertKeyEncryption(ctx, v.database.RW(), v.key); err != nil {
			return nil, err
		}
		v.inserted = true
	}
	return v.VaultServiceClient.ReEncryptBulk(ctx, req)
}

func TestReEncryptionRetriesFailuresWithoutLosingRows(t *testing.T) {
	for name, failure := range map[string]struct {
		count int32
		write bool
	}{
		"transient vault": {
			count: 1,
			write: false,
		},
		"persistent vault": {
			count: 1000,
			write: false,
		},
		"transient write": {
			count: 1,
			write: true,
		},
		"persistent write": {
			count: 1000,
			write: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.Close()) })
			testVault := vaulttestutil.StartTestVaultWithMemory(t)
			client := vault.NewConnectVaultServiceClient(testVault.Client)
			keyring := uid.New(uid.WorkspacePrefix)
			keyID := uid.New(uid.KeyPrefix)
			encrypted, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    "retry without losing rows",
			})
			require.NoError(t, err)
			require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
				WorkspaceID:     keyring,
				KeyID:           keyID,
				Encrypted:       encrypted.GetEncrypted(),
				EncryptionKeyID: encrypted.GetKeyId(),
				CreatedAt:       time.Now().UnixMilli(),
			}))
			keyIDNew, err := testVault.CreateDEK(t.Context(), keyring)
			require.NoError(t, err)
			testVault.Clock.Tick(25 * time.Hour)
			vaultFailing := &failingVault{VaultServiceClient: client}
			databaseFailing := &failingDatabase{Database: database}
			failuresPending := &vaultFailing.failures
			message := "failed to re-encrypt rows"
			if failure.write {
				failuresPending = &databaseFailing.failures
				message = "failed to update"
			}
			failuresPending.Store(failure.count)
			handler, err := vaultreencryption.New(vaultreencryption.Config{
				DB:        databaseFailing,
				Vault:     vaultFailing,
				Heartbeat: healthcheck.NewNoop(),
			})
			require.NoError(t, err)
			runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
				restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(5), restate.KillOnMaxAttempts()),
			))
			cronClient := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption")
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			t.Cleanup(cancel)
			failureLabels := prometheus.Labels{
				"table":   "encrypted_keys",
				"outcome": "error",
			}
			updatedLabels := prometheus.Labels{
				"table":   "encrypted_keys",
				"outcome": "updated",
			}
			failuresBefore := metricCounter(t, "unkey_control_vault_reencryptions_data_total", failureLabels)
			updatedBefore := metricCounter(t, "unkey_control_vault_reencryptions_data_total", updatedLabels)
			_, err = cronClient.RunVaultReEncryption().Request(ctx, &hydrav1.RunVaultReEncryptionRequest{})
			if failure.count > 1 {
				require.ErrorContains(t, err, message)
				require.NotContains(t, err.Error(), "sensitive upstream error")
				require.Equal(t, updatedBefore, metricCounter(t, "unkey_control_vault_reencryptions_data_total", updatedLabels))
				stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyID)
				require.NoError(t, err)
				require.Equal(t, encrypted.GetEncrypted(), stored.Encrypted)
				failuresPending.Store(0)
				_, err = cronClient.RunVaultReEncryption().Request(ctx, &hydrav1.RunVaultReEncryptionRequest{})
				require.NoError(t, err)
			} else {
				require.NoError(t, err)
			}
			failuresAfter := metricCounter(t, "unkey_control_vault_reencryptions_data_total", failureLabels)
			switch {
			case !failure.write:
				require.Equal(t, failuresBefore, failuresAfter)
			case failure.count == 1:
				require.Equal(t, failuresBefore+1, failuresAfter)
			default:
				require.Greater(t, failuresAfter, failuresBefore)
			}
			require.Equal(t, updatedBefore+1, metricCounter(t, "unkey_control_vault_reencryptions_data_total", updatedLabels))
			stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyID)
			require.NoError(t, err)
			require.Equal(t, keyIDNew, stored.EncryptionKeyID)
			decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: stored.Encrypted,
			})
			require.NoError(t, err)
			require.Equal(t, "retry without losing rows", decrypted.GetPlaintext())
		})
	}
}

type failingVault struct {
	vault.VaultServiceClient
	failures atomic.Int32
}

func (v *failingVault) ReEncryptBulk(ctx context.Context, req *vaultv1.ReEncryptBulkRequest) (*vaultv1.ReEncryptBulkResponse, error) {
	if v.failures.Add(-1) >= 0 {
		return nil, errors.New("sensitive upstream error")
	}
	return v.VaultServiceClient.ReEncryptBulk(ctx, req)
}

type failingDatabase struct {
	db.Database
	failures atomic.Int32
}

func (d *failingDatabase) UpdateVaultEncryptedKey(ctx context.Context, params db.UpdateVaultEncryptedKeyParams) (int64, error) {
	if d.failures.Add(-1) >= 0 {
		return 0, errors.New("sensitive upstream error")
	}
	return d.Database.UpdateVaultEncryptedKey(ctx, params)
}

func TestReEncryptionWithoutVaultFailsAtInvocationNotStartup(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	handler, err := vaultreencryption.New(vaultreencryption.Config{
		DB:        database,
		Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)
	runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
		restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
	))
	_, err = hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption").
		RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.ErrorContains(t, err, "Vault is not configured")
}

type cronService struct {
	hydrav1.UnimplementedCronServiceServer
	handler *vaultreencryption.Handler
}

func (s *cronService) RunVaultReEncryption(ctx restate.ObjectContext, req *hydrav1.RunVaultReEncryptionRequest) (*hydrav1.RunVaultReEncryptionResponse, error) {
	return s.handler.Handle(ctx, req)
}

func metricCounter(t *testing.T, name string, labels prometheus.Labels) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			match := len(labels) == len(metric.GetLabel())
			for _, label := range metric.GetLabel() {
				if labels[label.GetName()] != label.GetValue() {
					match = false
					break
				}
			}
			if match {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}
