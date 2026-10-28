package vaultreencryption_test

import (
	"context"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// A single invocation must drain partial and exact pages across tables.
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
		encrypted, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: keyIDs[i]})
		require.NoError(t, err)
		require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
			WorkspaceID: keyring, KeyID: keyIDs[i], Encrypted: encrypted.GetEncrypted(),
			EncryptionKeyID: encrypted.GetKeyId(), CreatedAt: time.Now().UnixMilli(),
		}))
	}
	shared, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: "shared secret"})
	require.NoError(t, err)
	for range 100 {
		require.NoError(t, pkgdb.Query.InsertSharedSecret(t.Context(), database.RW(), pkgdb.InsertSharedSecretParams{
			ID: uid.New(uid.TestPrefix), WorkspaceID: keyring, Encrypted: shared.GetEncrypted(),
			EncryptionKeyID: shared.GetKeyId(), CreatedAt: time.Now().UnixMilli(), ExpiresAt: time.Now().Add(time.Hour).UnixMilli(),
		}))
	}
	keyIDNew, err := testVault.CreateDEK(t.Context(), keyring)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)

	handler, err := vaultreencryption.New(vaultreencryption.Config{DB: database, Vault: client, Heartbeat: healthcheck.NewNoop()})
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
		decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: stored.Encrypted})
		require.NoError(t, err)
		require.Equal(t, keyID, decrypted.GetPlaintext())
	}
	secrets, err := database.ListVaultSharedSecrets(t.Context(), db.ListVaultSharedSecretsParams{Limit: 101})
	require.NoError(t, err)
	require.Len(t, secrets, 100)
	for _, secret := range secrets {
		encoded, err := base64.StdEncoding.DecodeString(secret.Ciphertext)
		require.NoError(t, err)
		var envelope vaultv1.Encrypted
		require.NoError(t, proto.Unmarshal(encoded, &envelope))
		require.Equal(t, keyIDNew, envelope.GetEncryptionKeyId())
		decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: secret.Ciphertext})
		require.NoError(t, err)
		require.Equal(t, "shared secret", decrypted.GetPlaintext())
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
		{name: "short page", rows: 99, rotated: false},
		{name: "full page", rows: 100, rotated: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, database.Close()) })
			testVault := vaulttestutil.StartTestVaultWithMemory(t)
			client := vault.NewConnectVaultServiceClient(testVault.Client)
			keyring := uid.New(uid.WorkspacePrefix)
			encrypted, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: "concurrent insert"})
			require.NoError(t, err)
			for range tt.rows {
				require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
					WorkspaceID: keyring, KeyID: uid.New(uid.KeyPrefix), Encrypted: encrypted.GetEncrypted(),
					EncryptionKeyID: encrypted.GetKeyId(), CreatedAt: time.Now().UnixMilli(),
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
					WorkspaceID: keyring, KeyID: keyIDLater, Encrypted: encrypted.GetEncrypted(),
					EncryptionKeyID: encrypted.GetKeyId(), CreatedAt: time.Now().UnixMilli(),
				},
			}
			handler, err := vaultreencryption.New(vaultreencryption.Config{DB: database, Vault: concurrent, Heartbeat: healthcheck.NewNoop()})
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
			decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: stored.Encrypted})
			require.NoError(t, err)
			require.Equal(t, "concurrent insert", decrypted.GetPlaintext())
		})
	}
}

func TestReEncryptionCoversStoredFormats(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	client := vault.NewConnectVaultServiceClient(testVault.Client)
	workspaceID := uid.New(uid.WorkspacePrefix)
	environmentID := uid.New(uid.EnvironmentPrefix)
	appID := uid.New(uid.AppPrefix)
	keysOld := make(map[string]string)
	encrypt := func(keyring, plaintext string) string {
		t.Helper()
		res, encryptErr := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: plaintext})
		require.NoError(t, encryptErr)
		keysOld[keyring] = res.GetKeyId()
		return res.GetEncrypted()
	}
	now := time.Now().UnixMilli()
	for _, kind := range []pkgdb.AppEnvironmentVariablesType{pkgdb.AppEnvironmentVariablesTypeRecoverable, pkgdb.AppEnvironmentVariablesTypeWriteonly} {
		require.NoError(t, pkgdb.Query.InsertAppEnvironmentVariable(t.Context(), database.RW(), pkgdb.InsertAppEnvironmentVariableParams{
			ID: uid.New(uid.TestPrefix), WorkspaceID: workspaceID, AppID: appID, EnvironmentID: environmentID,
			EnvKey: string(kind), Type: kind, Value: encrypt(environmentID, string(kind)), CreatedAt: now,
		}))
	}
	secrets := &ctrlv1.SecretsConfig{Secrets: map[string]string{
		"TOKEN": encrypt(environmentID, "snapshot token"), "HOST": encrypt(environmentID, "private host"),
	}}
	for i := range 99 {
		name := "VALUE_" + strconv.Itoa(i)
		secrets.Secrets[name] = encrypt(environmentID, name)
	}
	snapshot, err := protojson.Marshal(secrets)
	require.NoError(t, err)
	require.NoError(t, database.InsertDeployment(t.Context(), db.InsertDeploymentParams{
		ID: uid.New(uid.DeploymentPrefix), K8sName: uid.New(uid.TestPrefix), WorkspaceID: workspaceID,
		ProjectID: uid.New(uid.ProjectPrefix), AppID: appID, EnvironmentID: environmentID,
		Source: db.DeploymentsSourceOci, EncryptedEnvironmentVariables: snapshot, SentinelConfig: []byte{},
		Status: "failed", ShutdownSignal: db.DeploymentsShutdownSignalSIGTERM,
		UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1, DeploymentTrigger: db.DeploymentsTriggerApi, CreatedAt: now,
	}))
	configs := []*logdrainv1.Config{
		{Destination: &logdrainv1.Config_Http{Http: &logdrainv1.HttpConfig{
			Url: "https://logs.example.com", Format: logdrainv1.HttpBodyFormat_HTTP_BODY_FORMAT_NDJSON,
			Headers: []*logdrainv1.HttpHeader{
				{Name: "Authorization", EncryptedValue: encrypt(workspaceID, "bearer token")},
				{Name: "X-Custom", EncryptedValue: encrypt(workspaceID, "custom value")},
			},
		}}, BatchSize: 37},
		{Destination: &logdrainv1.Config_Axiom{Axiom: &logdrainv1.AxiomConfig{Dataset: "audit", EncryptedToken: encrypt(workspaceID, "axiom token")}}},
	}
	for _, config := range configs {
		config.ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x07})
		encoded, marshalErr := proto.Marshal(config)
		require.NoError(t, marshalErr)
		require.NoError(t, pkgdb.Query.InsertLogdrain(t.Context(), database.RW(), pkgdb.InsertLogdrainParams{
			ID: uid.New(uid.LogdrainPrefix), WorkspaceID: workspaceID, Name: "rotation test", Stream: pkgdb.LogdrainsStreamAuditLogs,
			Config: encoded, CreatedAt: now,
		}))
	}
	require.NoError(t, database.InsertCertificate(t.Context(), db.InsertCertificateParams{
		ID: uid.New(uid.TestPrefix), WorkspaceID: workspaceID, Hostname: "test.example.com", Certificate: "public certificate",
		EncryptedPrivateKey: encrypt(workspaceID, "private key"), CreatedAt: now,
	}))
	require.NoError(t, database.InsertAcmeUser(t.Context(), db.InsertAcmeUserParams{
		ID: uid.New(uid.TestPrefix), WorkspaceID: "acme", EncryptedKey: encrypt("acme", "account key"), CreatedAt: now,
	}))
	require.NoError(t, pkgdb.Query.InsertClickhouseWorkspaceSettings(t.Context(), database.RW(), pkgdb.InsertClickhouseWorkspaceSettingsParams{
		WorkspaceID: workspaceID, Username: uid.New(uid.TestPrefix), PasswordEncrypted: encrypt(workspaceID, "database password"), CreatedAt: now,
	}))
	require.NoError(t, pkgdb.Query.InsertSharedSecret(t.Context(), database.RW(), pkgdb.InsertSharedSecretParams{
		ID: uid.New(uid.TestPrefix), WorkspaceID: workspaceID, Encrypted: encrypt(workspaceID, "one time secret"),
		EncryptionKeyID: keysOld[workspaceID], CreatedAt: now, ExpiresAt: now + time.Hour.Milliseconds(),
	}))
	keysNew := make(map[string]string)
	for _, keyring := range []string{workspaceID, environmentID, "acme"} {
		keysNew[keyring], err = testVault.CreateDEK(t.Context(), keyring)
		require.NoError(t, err)
	}
	testVault.Clock.Tick(25 * time.Hour)
	handler, err := vaultreencryption.New(vaultreencryption.Config{DB: database, Vault: client, Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)
	runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
		restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
	))
	cronClient := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption")
	_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.NoError(t, err)
	_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.NoError(t, err)
	check := func(keyring, encrypted, plaintext string) {
		t.Helper()
		encoded, decodeErr := base64.StdEncoding.DecodeString(encrypted)
		require.NoError(t, decodeErr)
		var envelope vaultv1.Encrypted
		require.NoError(t, proto.Unmarshal(encoded, &envelope))
		require.Equal(t, keysNew[keyring], envelope.GetEncryptionKeyId())
		decrypted, decryptErr := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: encrypted})
		require.NoError(t, decryptErr)
		require.Equal(t, plaintext, decrypted.GetPlaintext())
	}
	variables, err := pkgdb.Query.ListAppEnvVarsByAppAndEnv(t.Context(), database.RW(), pkgdb.ListAppEnvVarsByAppAndEnvParams{
		AppID: appID, EnvironmentID: environmentID, Limit: 10,
	})
	require.NoError(t, err)
	require.Len(t, variables, 2)
	for _, variable := range variables {
		check(environmentID, variable.Value, variable.Key)
	}
	deployments, err := database.ListVaultDeployments(t.Context(), db.ListVaultDeploymentsParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, deployments, 1)
	var snapshotNew ctrlv1.SecretsConfig
	require.NoError(t, protojson.Unmarshal(deployments[0].Ciphertext, &snapshotNew))
	require.Len(t, snapshotNew.Secrets, 101)
	check(environmentID, snapshotNew.Secrets["TOKEN"], "snapshot token")
	check(environmentID, snapshotNew.Secrets["HOST"], "private host")
	for i := range 99 {
		name := "VALUE_" + strconv.Itoa(i)
		check(environmentID, snapshotNew.Secrets[name], name)
	}
	drains, err := database.ListVaultLogdrains(t.Context(), db.ListVaultLogdrainsParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, drains, 2)
	for i, drain := range drains {
		var configNew logdrainv1.Config
		require.NoError(t, proto.Unmarshal(drain.Ciphertext, &configNew))
		if configNew.GetHttp() != nil {
			check(workspaceID, configNew.GetHttp().Headers[0].EncryptedValue, "bearer token")
			check(workspaceID, configNew.GetHttp().Headers[1].EncryptedValue, "custom value")
			configNew.GetHttp().Headers = configs[i].GetHttp().Headers
		} else {
			check(workspaceID, configNew.GetAxiom().EncryptedToken, "axiom token")
			configNew.GetAxiom().EncryptedToken = configs[i].GetAxiom().EncryptedToken
		}
		require.True(t, proto.Equal(configs[i], &configNew), "non-secret fields and unknown fields must survive")
	}
	certificate, err := database.FindCertificateByHostname(t.Context(), "test.example.com")
	require.NoError(t, err)
	check(workspaceID, certificate.EncryptedPrivateKey, "private key")
	require.Equal(t, "public certificate", certificate.Certificate)
	acme, err := database.FindAcmeUserByWorkspaceID(t.Context(), "acme")
	require.NoError(t, err)
	check("acme", acme.EncryptedKey, "account key")
	settings, err := database.ListVaultClickhouseSettings(t.Context(), db.ListVaultClickhouseSettingsParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, settings, 1)
	check(workspaceID, settings[0].Ciphertext, "database password")
	shared, err := database.ListVaultSharedSecrets(t.Context(), db.ListVaultSharedSecretsParams{Limit: 10})
	require.NoError(t, err)
	require.Len(t, shared, 1)
	check(workspaceID, shared[0].Ciphertext, "one time secret")
}

func TestReEncryptionRotatesRecentCiphertext(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	client := vault.NewConnectVaultServiceClient(testVault.Client)
	keyring := uid.New(uid.WorkspacePrefix)
	encrypted, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: "old ciphertext on the current DEK"})
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
		WorkspaceID: keyring, KeyID: keyID, Encrypted: ciphertextOld, EncryptionKeyID: encrypted.GetKeyId(),
		CreatedAt: envelope.GetTime(),
	}))
	keyringRecent := uid.New(uid.WorkspacePrefix)
	recent, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyringRecent, Data: "recent ciphertext"})
	require.NoError(t, err)
	keyIDRecent := uid.New(uid.KeyPrefix)
	require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
		WorkspaceID: keyringRecent, KeyID: keyIDRecent, Encrypted: recent.GetEncrypted(), EncryptionKeyID: recent.GetKeyId(),
		CreatedAt: time.Now().UnixMilli(),
	}))
	keyIDRecentNew, err := testVault.CreateDEK(t.Context(), keyringRecent)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)
	handler, err := vaultreencryption.New(vaultreencryption.Config{DB: database, Vault: client, Heartbeat: healthcheck.NewNoop()})
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
	decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: stored.Encrypted})
	require.NoError(t, err)
	require.Equal(t, "old ciphertext on the current DEK", decrypted.GetPlaintext())
	storedRecent, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyIDRecent)
	require.NoError(t, err)
	require.Equal(t, keyIDRecentNew, storedRecent.EncryptionKeyID)
	decrypted, err = client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyringRecent, Encrypted: storedRecent.Encrypted})
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
	decrypted, err = client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: stored.Encrypted})
	require.NoError(t, err)
	require.Equal(t, "old ciphertext on the current DEK", decrypted.GetPlaintext())
	storedRecentAgain, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyIDRecent)
	require.NoError(t, err)
	require.Equal(t, keyIDRecentNew, storedRecentAgain.EncryptionKeyID)
	decrypted, err = client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyringRecent, Encrypted: storedRecentAgain.Encrypted})
	require.NoError(t, err)
	require.Equal(t, "recent ciphertext", decrypted.GetPlaintext())
}

func TestReEncryptionRetriesFailedPageWithoutPartialWrites(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	client := vault.NewConnectVaultServiceClient(testVault.Client)
	keyringOld := uid.New(uid.WorkspacePrefix)
	old, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyringOld, Data: "beyond the first batch"})
	require.NoError(t, err)
	keyID := uid.New(uid.KeyPrefix)
	keyring := uid.New(uid.WorkspacePrefix)
	current, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: "current"})
	require.NoError(t, err)
	keys := make([]string, 100)
	for i := range keys {
		keys[i] = uid.New(uid.KeyPrefix)
		ciphertext := current.GetEncrypted()
		if i == 0 {
			ciphertext = "invalid-base64"
		}
		require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
			WorkspaceID: keyring, KeyID: keys[i], Encrypted: ciphertext,
			EncryptionKeyID: current.GetKeyId(), CreatedAt: time.Now().UnixMilli(),
		}))
	}
	require.NoError(t, pkgdb.Query.InsertKeyEncryption(t.Context(), database.RW(), pkgdb.InsertKeyEncryptionParams{
		WorkspaceID: keyringOld, KeyID: keyID, Encrypted: old.GetEncrypted(),
		EncryptionKeyID: old.GetKeyId(), CreatedAt: time.Now().UnixMilli(),
	}))
	_, err = testVault.CreateDEK(t.Context(), keyringOld)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)
	handler, err := vaultreencryption.New(vaultreencryption.Config{DB: database, Vault: client, Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)
	runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
		restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
	))
	cronClient := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption")
	for range 2 {
		_, err = cronClient.RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
		require.Error(t, err, "each invocation must retry the failed page")
		storedCurrent, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keys[1])
		require.NoError(t, err)
		require.Equal(t, current.GetEncrypted(), storedCurrent.Encrypted, "a failed batch must not write partial results")
		stored, err := pkgdb.Query.FindKeyEncryptionByKeyID(t.Context(), database.RW(), keyID)
		require.NoError(t, err)
		require.Equal(t, old.GetEncrypted(), stored.Encrypted, "a failed page must not advance the cursor")
	}
}

// Renewal can commit while Vault processes the old private key. The sweep must
// not put that old key back beside the renewed public certificate.
func TestReEncryptionPreservesConcurrentCertificateRenewal(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	client := vault.NewConnectVaultServiceClient(testVault.Client)
	keyring := uid.New(uid.WorkspacePrefix)
	old, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: "old private key"})
	require.NoError(t, err)
	certificate := db.InsertCertificateParams{
		ID: uid.New(uid.CertificatePrefix), WorkspaceID: keyring, Hostname: "renewal.example.com",
		Certificate: "old public certificate", EncryptedPrivateKey: old.GetEncrypted(), CreatedAt: time.Now().UnixMilli(),
	}
	require.NoError(t, database.InsertCertificate(t.Context(), certificate))
	renewed, err := client.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: keyring, Data: "renewed private key"})
	require.NoError(t, err)
	_, err = testVault.CreateDEK(t.Context(), keyring)
	require.NoError(t, err)
	testVault.Clock.Tick(25 * time.Hour)
	certificate.Certificate = "renewed public certificate"
	certificate.EncryptedPrivateKey = renewed.GetEncrypted()
	concurrent := &renewingVault{VaultServiceClient: client, database: database, certificate: certificate}
	handler, err := vaultreencryption.New(vaultreencryption.Config{DB: database, Vault: concurrent, Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)
	runtime := containers.Restate(t, hydrav1.NewCronServiceServer(&cronService{handler: handler},
		restate.WithInvocationRetryPolicy(restate.WithMaxRetryAttempts(1), restate.KillOnMaxAttempts()),
	))
	_, err = hydrav1.NewCronServiceIngressClient(runtime.IngressClient, "vault-re-encryption").
		RunVaultReEncryption().Request(t.Context(), &hydrav1.RunVaultReEncryptionRequest{})
	require.NoError(t, err)
	stored, err := database.FindCertificateByHostname(t.Context(), certificate.Hostname)
	require.NoError(t, err)
	require.Equal(t, certificate.Certificate, stored.Certificate)
	require.Equal(t, renewed.GetEncrypted(), stored.EncryptedPrivateKey)
	decrypted, err := client.Decrypt(t.Context(), &vaultv1.DecryptRequest{Keyring: keyring, Encrypted: stored.EncryptedPrivateKey})
	require.NoError(t, err)
	require.Equal(t, "renewed private key", decrypted.GetPlaintext())
}

type renewingVault struct {
	vault.VaultServiceClient
	database    db.Database
	certificate db.InsertCertificateParams
}

func (v *renewingVault) ReEncryptBulk(ctx context.Context, req *vaultv1.ReEncryptBulkRequest) (*vaultv1.ReEncryptBulkResponse, error) {
	if err := v.database.InsertCertificate(ctx, v.certificate); err != nil {
		return nil, err
	}
	return v.VaultServiceClient.ReEncryptBulk(ctx, req)
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

func TestReEncryptionWithoutVaultFailsAtInvocationNotStartup(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	handler, err := vaultreencryption.New(vaultreencryption.Config{DB: database, Heartbeat: healthcheck.NewNoop()})
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
