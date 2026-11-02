package keyleak_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	keydb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/keyleak"
)

func TestNotifyBatchDisablesOnlyMatchingLiveKeysAndAuditsOnceUnderConcurrentInvocations(t *testing.T) {
	f := newBatchFixture(t)
	workspace := f.seeder.Resources.UserWorkspace
	foreign := f.seeder.CreateWorkspace(t.Context())
	wrongOwner := f.seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: f.seeder.Resources.RootWorkspace.ID, KeySpaceID: f.seeder.Resources.RootKeySpace.ID,
		ForWorkspaceID: &workspace.ID,
	})
	insertBatchRoot(t, f.database, workspace.ID, wrongOwner)
	wrongCustomer := f.seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: workspace.ID, KeySpaceID: f.customerKeySpaceID,
	})
	disabled := f.seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: workspace.ID, KeySpaceID: f.customerKeySpaceID, Disabled: true,
	})
	deleted := f.seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: workspace.ID, KeySpaceID: f.customerKeySpaceID, Deleted: true,
	})
	old := f.seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: f.seeder.Resources.RootWorkspace.ID, KeySpaceID: f.seeder.Resources.RootKeySpace.ID,
		ForWorkspaceID: &workspace.ID,
	})
	replacement := seed.CreateKeyResponse{KeyID: old.KeyID, Key: uid.New(uid.TestPrefix)}
	insertBatchRoot(t, f.database, workspace.ID, replacement)
	f.matches = append(f.matches,
		batchMatch(wrongOwner, foreign, true), batchMatch(wrongCustomer, foreign, false),
		batchMatch(disabled, workspace, false), batchMatch(deleted, workspace, false),
		batchMatch(old, workspace, true),
	)
	f.tokens = append(f.tokens, wrongOwner.Key, wrongCustomer.Key, disabled.Key, deleted.Key, old.Key, replacement.Key)

	auditEntered := make(chan int, 1)
	auditReady := make(chan struct{})
	releaseAudit := sync.OnceFunc(func() { close(auditReady) })
	var auditCalls atomic.Int32
	service := keyleak.New(keyleak.Config{
		DB: f.database,
		Auditlogs: batchAuditWriter(func(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
			if auditCalls.Add(1) == 1 {
				auditEntered <- len(logs)
				select {
				case <-auditReady:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return f.auditlogs.Insert(ctx, tx, logs)
		}),
		Members: memberLookup(func(context.Context, string) ([]string, error) { return nil, nil }),
	})
	runtime := containers.Restate(t, service.Definition())
	t.Cleanup(releaseAudit)
	client := hydrav1.NewKeyLeakServiceIngressClient(runtime.IngressClient)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	first, err := client.NotifyBatch().Send(ctx, &hydrav1.KeyLeakBatchRequest{
		ReportId: uid.New(uid.TestPrefix), Keys: f.matches,
	})
	require.NoError(t, err)
	select {
	case count := <-auditEntered:
		require.Equal(t, 4, count, "all changed logical keys must be audited in one Insert")
	case <-ctx.Done():
		t.Fatal("batch never reached audit insertion")
	}
	invocations := []ingress.InvocationHandle[*hydrav1.KeyLeakResponse]{first}
	for range 3 {
		invocation, err := client.NotifyBatch().Send(ctx, &hydrav1.KeyLeakBatchRequest{
			ReportId: uid.New(uid.TestPrefix), Keys: f.matches,
		})
		require.NoError(t, err)
		require.NotEqual(t, first.Id(), invocation.Id(), "exercise distinct invocations, not Restate deduplication")
		invocations = append(invocations, invocation)
	}
	releaseAudit()
	for _, invocation := range invocations {
		_, err := invocation.Attach(ctx)
		require.NoError(t, err)
	}

	for _, test := range []struct {
		name    string
		keyID   string
		enabled bool
		deleted bool
	}{
		{name: "migrated legacy root", keyID: f.root.KeyID},
		{name: "customer key", keyID: f.customer.KeyID},
		{name: "old root hash", keyID: old.KeyID},
		{name: "wrong root workspace", keyID: wrongOwner.KeyID, enabled: true},
		{name: "wrong customer workspace", keyID: wrongCustomer.KeyID, enabled: true},
		{name: "already disabled", keyID: disabled.KeyID},
		{name: "deleted", keyID: deleted.KeyID, enabled: true, deleted: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			stored, err := keydb.Query.FindKeyByID(ctx, f.database.RW(), test.keyID)
			require.NoError(t, err)
			require.Equal(t, test.enabled, stored.Enabled)
			require.Equal(t, test.deleted, stored.DeletedAtM.Valid)
		})
	}
	for _, test := range []struct {
		keyID   string
		enabled bool
	}{
		{keyID: f.root.KeyID}, {keyID: f.rootNew.KeyID},
		{keyID: wrongOwner.KeyID, enabled: true}, {keyID: replacement.KeyID, enabled: true},
	} {
		stored, err := keydb.Query.FindUnkeyRootKeyByID(ctx, f.database.RW(), test.keyID)
		require.NoError(t, err)
		require.Equal(t, test.enabled, stored.Enabled, "new-store root %s", test.keyID)
	}
	events := batchEvents(t, f.database, workspace.ID)
	require.Len(t, events, 4, "concurrent invocations must audit each changed logical key once")
	want := map[string]*hydrav1.KeyLeakMatch{
		f.root.KeyID: f.matches[0], f.customer.KeyID: f.matches[1], f.rootNew.KeyID: f.matches[2],
		old.KeyID: f.matches[len(f.matches)-1],
	}
	for _, event := range events {
		require.Len(t, event.Targets, 1)
		keyID := event.Targets[0].ID
		match, ok := want[keyID]
		require.True(t, ok, "unexpected or duplicate audit for %s", keyID)
		delete(want, keyID)
		eventName := "key.leaked"
		if match.GetRootKey() {
			eventName = "rootKey.leaked"
		}
		require.Equal(t, eventName, event.Event)
		require.Equal(t, workspace.ID, event.WorkspaceID)
		require.Equal(t, map[string]any{"source": "github"}, event.Actor.Meta)
		require.Equal(t, string(auditlog.KeyTargetType), event.Targets[0].Type)
		require.Equal(t, map[string]any{
			"source": match.GetNotification().GetSource(), "url": match.GetNotification().GetUrl(), "enabled": false,
		}, event.Targets[0].Meta)
		payload, err := json.Marshal(event)
		require.NoError(t, err)
		for _, token := range f.tokens {
			require.NotContains(t, string(payload), token)
		}
	}
	require.Empty(t, want)
	require.Empty(t, batchEvents(t, f.database, foreign.ID))
}

func TestNotifyBatchRollsBackEntireBatchAndRetriesBeforeRecipientLookup(t *testing.T) {
	f := newBatchFixture(t)
	retryEntered := make(chan int, 1)
	recovery := make(chan struct{})
	var recovered atomic.Bool
	release := sync.OnceFunc(func() {
		recovered.Store(true)
		close(recovery)
	})
	var attempts atomic.Int32
	var lookedUpBeforeRecovery atomic.Bool
	delivered := make(chan email.Email, len(f.matches))
	service := keyleak.New(keyleak.Config{
		DB: f.database,
		Auditlogs: batchAuditWriter(func(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
			if attempts.Add(1) == 1 {
				if err := f.auditlogs.Insert(ctx, tx, logs); err != nil {
					return err
				}
				return errors.New("audit transaction failed after writing outbox")
			}
			select {
			case retryEntered <- len(logs):
			case <-ctx.Done():
				return ctx.Err()
			}
			select {
			case <-recovery:
			case <-ctx.Done():
				return ctx.Err()
			}
			return f.auditlogs.Insert(ctx, tx, logs)
		}),
		Members: memberLookup(func(context.Context, string) ([]string, error) {
			if !recovered.Load() {
				lookedUpBeforeRecovery.Store(true)
			}
			return []string{"member@example.com"}, nil
		}),
		Email: batchEmailSender(func(ctx context.Context, message email.Email) error {
			select {
			case delivered <- message:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}),
	})
	runtime := containers.Restate(t, service.Definition())
	t.Cleanup(release)
	client := hydrav1.NewKeyLeakServiceIngressClient(runtime.IngressClient)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	invocation, err := client.NotifyBatch().Send(ctx, &hydrav1.KeyLeakBatchRequest{
		ReportId: uid.New(uid.TestPrefix), Keys: f.matches,
	})
	require.NoError(t, err)
	select {
	case count := <-retryEntered:
		require.Equal(t, len(f.matches), count, "durable retry must still see every key enabled")
	case <-ctx.Done():
		t.Fatal("audit failure was not retried by Restate")
	}
	for _, key := range []seed.CreateKeyResponse{f.root, f.customer} {
		stored, err := keydb.Query.FindKeyByID(ctx, f.database.RW(), key.KeyID)
		require.NoError(t, err)
		require.True(t, stored.Enabled, "audit failure must roll back every legacy update")
	}
	for _, key := range []seed.CreateKeyResponse{f.root, f.rootNew} {
		stored, err := keydb.Query.FindUnkeyRootKeyByID(ctx, f.database.RW(), key.KeyID)
		require.NoError(t, err)
		require.True(t, stored.Enabled, "audit failure must roll back every new-store update")
	}
	require.Empty(t, batchEvents(t, f.database, f.seeder.Resources.UserWorkspace.ID), "failed audit inserts must roll back too")
	require.False(t, lookedUpBeforeRecovery.Load())
	require.Empty(t, delivered)
	release()
	_, err = invocation.Attach(ctx)
	require.NoError(t, err)
	want := make(map[string]*hydrav1.KeyLeakNotification, len(f.matches))
	for _, match := range f.matches {
		want[match.GetNotification().GetUrl()] = match.GetNotification()
	}
	for range f.matches {
		select {
		case message := <-delivered:
			notification, ok := want[message.Variables["URL"]]
			require.True(t, ok, "unexpected or duplicate notification")
			delete(want, message.Variables["URL"])
			require.Equal(t, notification.GetSource(), message.Variables["SOURCE"])
			require.Equal(t, []string{"member@example.com"}, message.To)
			payload, err := json.Marshal(message)
			require.NoError(t, err)
			for _, token := range f.tokens {
				require.NotContains(t, string(payload), token)
			}
		case <-ctx.Done():
			t.Fatal("recovered batch did not deliver every notification")
		}
	}
	require.False(t, lookedUpBeforeRecovery.Load(), "no recipient lookup may precede batch recovery")
	require.Equal(t, int32(2), attempts.Load())
	for _, key := range []seed.CreateKeyResponse{f.root, f.customer} {
		stored, err := keydb.Query.FindKeyByID(ctx, f.database.RW(), key.KeyID)
		require.NoError(t, err)
		require.False(t, stored.Enabled)
	}
	for _, key := range []seed.CreateKeyResponse{f.root, f.rootNew} {
		stored, err := keydb.Query.FindUnkeyRootKeyByID(ctx, f.database.RW(), key.KeyID)
		require.NoError(t, err)
		require.False(t, stored.Enabled)
	}
	require.Len(t, batchEvents(t, f.database, f.seeder.Resources.UserWorkspace.ID), len(f.matches))
}

type batchFixture struct {
	database           db.Database
	auditlogs          auditlogs.AuditLogService
	seeder             *seed.Seeder
	customerKeySpaceID string
	root               seed.CreateKeyResponse
	customer           seed.CreateKeyResponse
	rootNew            seed.CreateKeyResponse
	matches            []*hydrav1.KeyLeakMatch
	tokens             []string
}

func newBatchFixture(t *testing.T) batchFixture {
	t.Helper()
	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	auditlogsSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	seeder := seed.New(t, database, nil)
	seeder.Seed(t.Context())
	workspace := seeder.Resources.UserWorkspace
	api := seeder.CreateAPI(t.Context(), seed.CreateApiRequest{WorkspaceID: workspace.ID})
	root := seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: seeder.Resources.RootWorkspace.ID, KeySpaceID: seeder.Resources.RootKeySpace.ID,
		ForWorkspaceID: &workspace.ID,
	})
	insertBatchRoot(t, database, workspace.ID, root)
	customer := seeder.CreateKey(t.Context(), seed.CreateKeyRequest{
		WorkspaceID: workspace.ID, KeySpaceID: api.KeyAuthID.String,
	})
	rootNew := seed.CreateKeyResponse{KeyID: uid.New(uid.KeyPrefix), Key: uid.New(uid.TestPrefix)}
	insertBatchRoot(t, database, workspace.ID, rootNew)
	return batchFixture{
		database: database, auditlogs: auditlogsSvc, seeder: seeder, customerKeySpaceID: api.KeyAuthID.String,
		root: root, customer: customer, rootNew: rootNew,
		matches: []*hydrav1.KeyLeakMatch{
			batchMatch(root, workspace, true), batchMatch(customer, workspace, false), batchMatch(rootNew, workspace, true),
		},
		tokens: []string{root.Key, customer.Key, rootNew.Key},
	}
}

func insertBatchRoot(t *testing.T, database db.Database, workspaceID string, key seed.CreateKeyResponse) {
	t.Helper()
	require.NoError(t, keydb.Query.InsertUnkeyRootKey(t.Context(), database.RW(), keydb.InsertUnkeyRootKeyParams{
		ID: key.KeyID, WorkspaceID: workspaceID, Hash: hash.Sha256(key.Key),
		Prefix: "test", Start: key.Key[:8], End: key.Key[len(key.Key)-4:], Enabled: true,
		CreatedAt: time.Now().UnixMilli(),
	}))
}

func batchMatch(key seed.CreateKeyResponse, workspace db.Workspace, root bool) *hydrav1.KeyLeakMatch {
	keyType, source, path := "unkey_key", "commit", "commit/"
	if root {
		keyType, source, path = "unkey_root_key", "content", "blob/main/"
	}
	return &hydrav1.KeyLeakMatch{
		Hash: hash.Sha256(key.Key), WorkspaceId: workspace.ID, RootKey: root,
		Notification: &hydrav1.KeyLeakNotification{
			KeyId: key.KeyID, OrgId: workspace.OrgID, WorkspaceName: workspace.Name,
			Type: keyType, Source: source, Url: "https://github.com/unkeyed/example/" + path + key.KeyID,
			Date: "Tue Oct 27 2026",
		},
	}
}

func batchEvents(t *testing.T, database db.Database, workspaceID string) []auditlog.Event {
	t.Helper()
	rows, err := keydb.Query.ListClickhouseOutboxByWorkspace(t.Context(), database.RW(), workspaceID)
	require.NoError(t, err)
	events := make([]auditlog.Event, 0, len(rows))
	for _, row := range rows {
		var event auditlog.Event
		require.NoError(t, json.Unmarshal(row.Payload, &event))
		events = append(events, event)
	}
	return events
}

type batchAuditWriter func(context.Context, db.DBTX, []auditlog.AuditLog) error

func (f batchAuditWriter) Insert(ctx context.Context, tx db.DBTX, logs []auditlog.AuditLog) error {
	return f(ctx, tx, logs)
}

type batchEmailSender func(context.Context, email.Email) error

func (f batchEmailSender) Send(ctx context.Context, message email.Email) error {
	return f(ctx, message)
}
