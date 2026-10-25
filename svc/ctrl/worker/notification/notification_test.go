package notification

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	vaulttestutil "github.com/unkeyed/unkey/svc/vault/testutil"
	"google.golang.org/protobuf/proto"
)

type testDirectory struct {
	enabled          bool
	audienceTooLarge bool
}

type testPayload struct {
	Text string `json:"text"`
}

func (d *testDirectory) Members(context.Context, string) ([]workos.Member, error) {
	if d.audienceTooLarge {
		return nil, workos.ErrDirectoryLimit
	}
	return []workos.Member{{UserID: "alice", Email: "alice@example.com", Admin: true}, {UserID: "bob", Email: "bob@example.com"}}, nil
}

func (d *testDirectory) LookupMembers(_ context.Context, _ string, userIDs []string) ([]workos.Member, error) {
	var members []workos.Member
	for _, userID := range userIDs {
		switch userID {
		case "alice":
			members = append(members, workos.Member{UserID: "alice", Email: "alice@example.com", Admin: true})
		case "bob":
			members = append(members, workos.Member{UserID: "bob", Email: "bob@example.com"})
		}
	}
	return members, nil
}

func (d *testDirectory) CanEmail(context.Context, string, string, string) (bool, error) {
	return d.enabled, nil
}

type unavailableDirectory struct {
	workos.Directory
	unavailable atomic.Bool
}

func (d *unavailableDirectory) CanEmail(ctx context.Context, orgID, userID, email string) (bool, error) {
	if d.unavailable.Load() {
		return false, errors.New("directory unavailable")
	}
	return d.Directory.CanEmail(ctx, orgID, userID, email)
}

type testProvider struct {
	mu       sync.Mutex
	requests []notification.Request
	failure  error
	window   time.Duration
	channel  notification.Channel
}

func (p *testProvider) Name() string                     { return "test" }
func (p *testProvider) Channel() notification.Channel    { return p.channel }
func (p *testProvider) IdempotencyWindow() time.Duration { return p.window }
func (p *testProvider) Send(_ context.Context, r notification.Request) (notification.Receipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requests = append(p.requests, r)
	return notification.Receipt{MessageID: "provider-receipt"}, p.failure
}

type failedReceiptDB struct {
	db.Database
	fail  atomic.Bool
	plans atomic.Int32
}

func (d *failedReceiptDB) FindNotification(ctx context.Context, arg db.FindNotificationParams) (db.Notification, error) {
	d.plans.Add(1)
	return d.Database.FindNotification(ctx, arg)
}

func (d *failedReceiptDB) UpdateNotificationDelivery(ctx context.Context, arg db.UpdateNotificationDeliveryParams) (int64, error) {
	if d.fail.Swap(false) {
		return 0, errors.New("lost database response")
	}
	return d.Database.UpdateNotificationDelivery(ctx, arg)
}

type blockedClaimDB struct {
	db.Database
	waiting      chan struct{}
	release      <-chan struct{}
	calls        atomic.Int32
	participants int32
}

func (d *blockedClaimDB) StartNotificationDeliveryAttempt(ctx context.Context, arg db.StartNotificationDeliveryAttemptParams) (int64, error) {
	if d.calls.Add(1) <= d.participants {
		d.waiting <- struct{}{}
		select {
		case <-d.release:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	return d.Database.StartNotificationDeliveryAttempt(ctx, arg)
}

func TestNotificationIntegration(t *testing.T) {
	database, err := db.New(containers.MySQLIsolated(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	crypto := vault.NewConnectVaultServiceClient(vaulttestutil.StartTestVaultWithMemory(t).Client)
	newService := func(t *testing.T) (*Service, notification.Event, *testProvider, *clock.TestClock) {
		t.Helper()
		clk := clock.NewTestClock(time.Now())
		workspaceID := uid.New("ws")
		require.NoError(t, database.InsertWorkspace(t.Context(), db.InsertWorkspaceParams{
			ID: workspaceID, OrgID: uid.New("org"), Name: "Notifications", Slug: uid.New("slug"), CreatedAt: clk.Now().UnixMilli(), K8sNamespace: uid.DNS1035(),
		}))
		provider := &testProvider{channel: notification.Email}
		svc, err := New(Config{
			DB: database, Vault: crypto, Clock: clk, Directory: &testDirectory{enabled: true},
			Providers: []notification.Provider{provider}, Defaults: map[notification.Channel]string{notification.Email: "test"},
			Audiences: map[notificationv1.Audience]AudienceResolver{notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS: WorkspaceAdmins},
			Definitions: []notification.Definition{notification.JSONDefinition[testPayload]{
				Name: "test.event", SchemaVersion: 1,
				RenderPayload: func(p testPayload) (notification.Content, error) {
					return notification.Content{Subject: "Test event", Text: p.Text}, nil
				},
			}},
		})
		require.NoError(t, err)
		return svc, notification.Event{WorkspaceID: workspaceID, SourceEventID: "occurrence-1", Type: "test.event", SchemaVersion: 1, Payload: json.RawMessage(`{"text":"Original content"}`), OccurredAt: clk.Now().UnixMilli()}, provider, clk
	}
	addTarget := func(t *testing.T, event notification.Event, userID string, audience sql.NullInt32, project sql.NullString) {
		t.Helper()
		subID := uid.New("sub")
		_, err := database.RW().ExecContext(t.Context(), `INSERT INTO notification_subscriptions (id, workspace_id, name, notification_type, project_id, enabled, created_at) VALUES (?, ?, 'rule', ?, ?, TRUE, 1)`, subID, event.WorkspaceID, event.Type, project)
		require.NoError(t, err)
		_, err = database.RW().ExecContext(t.Context(), `INSERT INTO notification_subscription_targets (id, workspace_id, subscription_id, channel, user_id, audience, created_at) VALUES (?, ?, ?, 'email', ?, ?, 1)`, uid.New("target"), event.WorkspaceID, subID, nullable(userID), audience)
		require.NoError(t, err)
	}
	addAlice := func(t *testing.T, event notification.Event) {
		t.Helper()
		addTarget(t, event, "alice", sql.NullInt32{}, sql.NullString{})
	}
	planOne := func(t *testing.T, svc *Service, event notification.Event) deliveryRef {
		t.Helper()
		refs, err := svc.plan(t.Context(), event)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		return refs[0]
	}

	for _, overlap := range []string{"simultaneous claims", "after acceptance", "after rate limit"} {
		t.Run(overlap, func(t *testing.T) {
			svc, event, provider, _ := newService(t)
			addAlice(t, event)
			ref := planOne(t, svc, event)
			release := make(chan struct{})
			proxy := &blockedClaimDB{Database: database, waiting: make(chan struct{}, 2), release: release, participants: 1}
			if overlap == "simultaneous claims" {
				proxy.participants = 2
			}
			if overlap == "after rate limit" {
				provider.failure = &notification.SendError{Kind: notification.Retryable, Code: "rate_limited", RetryAfter: time.Minute}
			}
			svc.db = proxy
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			var workers sync.WaitGroup
			t.Cleanup(func() { cancel(); workers.Wait() })
			results := make(chan error, 2)
			start := func() {
				workers.Go(func() {
					_, err := svc.attempt(ctx, ref)
					results <- err
				})
			}
			waitForClaim := func() {
				t.Helper()
				select {
				case <-proxy.waiting:
				case <-ctx.Done():
					t.Fatal("delivery did not reach claim")
				}
			}
			start()
			waitForClaim()
			if overlap == "simultaneous claims" {
				start()
				waitForClaim()
			} else {
				_, err := svc.attempt(ctx, ref)
				require.NoError(t, err)
			}
			close(release)
			workers.Wait()
			close(results)
			for err := range results {
				require.NoError(t, err)
			}
			require.Len(t, provider.requests, 1)
			row, err := database.FindNotificationDelivery(t.Context(), db.FindNotificationDeliveryParams{WorkspaceID: event.WorkspaceID, ID: ref.ID})
			require.NoError(t, err)
			require.Equal(t, uint32(1), row.AttemptCount)
			if overlap == "after acceptance" {
				require.Equal(t, db.NotificationDeliveriesStatusAccepted, row.Status)
			}
			if overlap == "after rate limit" {
				require.Equal(t, db.NotificationDeliveriesStatusPending, row.Status)
				require.True(t, row.NextAttemptAt.Valid)
			}
		})
	}

	t.Run("stale result cannot overwrite replacement attempt", func(t *testing.T) {
		svc, event, _, clk := newService(t)
		addAlice(t, event)
		ref := planOne(t, svc, event)
		for count, status := range []db.NotificationDeliveriesStatus{db.NotificationDeliveriesStatusPending, db.NotificationDeliveriesStatusSending} {
			claimed, err := database.StartNotificationDeliveryAttempt(t.Context(), db.StartNotificationDeliveryAttemptParams{
				WorkspaceID: event.WorkspaceID, ID: ref.ID, Now: sql.NullInt64{Int64: clk.Now().UnixMilli(), Valid: true},
				ExpectedStatus: status, ExpectedAttemptCount: uint32(count),
			})
			require.NoError(t, err)
			require.Equal(t, int64(1), claimed)
		}
		row, err := database.FindNotificationDelivery(t.Context(), db.FindNotificationDeliveryParams{WorkspaceID: event.WorkspaceID, ID: ref.ID})
		require.NoError(t, err)
		stale := row
		stale.AttemptCount = 1
		_, err = svc.record(t.Context(), stale, db.NotificationDeliveriesStatusPending, "stale", "", time.Minute)
		require.ErrorIs(t, err, errDeliveryChanged)
		_, err = svc.record(t.Context(), row, db.NotificationDeliveriesStatusAccepted, "", "new-receipt", 0)
		require.NoError(t, err)
		_, err = svc.record(t.Context(), stale, db.NotificationDeliveriesStatusFailed, "stale", "", 0)
		require.ErrorIs(t, err, errDeliveryChanged)
		row, err = database.FindNotificationDelivery(t.Context(), db.FindNotificationDeliveryParams{WorkspaceID: event.WorkspaceID, ID: ref.ID})
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusAccepted, row.Status)
		require.Equal(t, "new-receipt", row.ProviderMessageID.String)
		require.Equal(t, uint32(2), row.AttemptCount)
	})

	t.Run("normalization applies to stored event and snapshot", func(t *testing.T) {
		svc, event, provider, _ := newService(t)
		addAlice(t, event)
		event.Payload = json.RawMessage(`{ "text": "Last", "Text": "First" }`)
		ref := planOne(t, svc, event)
		stored, err := database.FindNotification(t.Context(), db.FindNotificationParams{WorkspaceID: event.WorkspaceID, ID: event.ID()})
		require.NoError(t, err)
		require.JSONEq(t, `{"Text":"First","text":"Last"}`, string(stored.Payload))
		_, err = svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, "Last", provider.requests[0].Content.Text)
		require.Equal(t, `{"Text":"First","text":"Last"}`, string(provider.requests[0].Event.Payload))
		event.Payload = json.RawMessage(`{"text":"Changed","text":null}`)
		_, err = svc.plan(t.Context(), event)
		require.Error(t, err)
	})

	t.Run("routing union freezes recipients and rejects identity reuse", func(t *testing.T) {
		svc, event, provider, _ := newService(t)
		addAlice(t, event)
		addTarget(t, event, "", sql.NullInt32{Int32: 1, Valid: true}, sql.NullString{})
		addTarget(t, event, "bob", sql.NullInt32{}, nullable("different-project"))
		foreign := event
		foreign.WorkspaceID = "foreign-workspace"
		addTarget(t, foreign, "bob", sql.NullInt32{}, sql.NullString{})
		ref := planOne(t, svc, event)
		row, err := database.FindNotificationDelivery(t.Context(), db.FindNotificationDeliveryParams{WorkspaceID: event.WorkspaceID, ID: ref.ID})
		require.NoError(t, err)
		require.NotContains(t, row.EncryptedTargetSnapshot, "alice@example.com")
		require.NotContains(t, row.EncryptedTargetSnapshot, "Original content")
		addTarget(t, event, "bob", sql.NullInt32{}, sql.NullString{})
		require.Equal(t, ref, planOne(t, svc, event))
		changed := event
		changed.Payload = json.RawMessage(`{"text":"Changed content"}`)
		_, err = svc.plan(t.Context(), changed)
		require.ErrorContains(t, err, "identity reused")
		result, err := svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusAccepted, result.Status)
		result, err = svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusAccepted, result.Status)
		require.Len(t, provider.requests, 1)
		require.Equal(t, "Original content", provider.requests[0].Content.Text)
		refs, err := svc.plan(t.Context(), event)
		require.NoError(t, err)
		require.Empty(t, refs)
	})

	t.Run("zero recipient event stays empty", func(t *testing.T) {
		svc, event, _, _ := newService(t)
		refs, err := svc.plan(t.Context(), event)
		require.NoError(t, err)
		require.Empty(t, refs)
		addAlice(t, event)
		refs, err = svc.plan(t.Context(), event)
		require.NoError(t, err)
		require.Empty(t, refs)
	})

	t.Run("large encrypted snapshot round trip", func(t *testing.T) {
		svc, event, provider, _ := newService(t)
		text := strings.Repeat("a", 60_000)
		event.Payload = json.RawMessage(`{"text":"` + text + `"}`)
		addAlice(t, event)
		ref := planOne(t, svc, event)
		result, err := svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusAccepted, result.Status)
		require.Equal(t, text, provider.requests[0].Content.Text)
	})

	t.Run("delivery failure rolls back notification transaction", func(t *testing.T) {
		svc, event, _, _ := newService(t)
		addAlice(t, event)
		require.NoError(t, database.InsertNotificationDelivery(t.Context(), db.InsertNotificationDeliveryParams{
			ID: notification.DeliveryID(event.ID(), notification.Email, "user:alice"), WorkspaceID: event.WorkspaceID,
			NotificationID: "other", Channel: db.NotificationDeliveriesChannelEmail, TargetKey: "user:alice", Status: db.NotificationDeliveriesStatusFailed,
		}))
		_, err := svc.plan(t.Context(), event)
		require.Error(t, err)
		_, err = database.FindNotification(t.Context(), db.FindNotificationParams{WorkspaceID: event.WorkspaceID, ID: event.ID()})
		require.True(t, db.IsNotFound(err))
	})

	t.Run("destination snapshot decrypts secrets and live pause stops retry", func(t *testing.T) {
		svc, event, provider, clk := newService(t)
		provider.channel = notification.Webhook
		svc.defaults = map[notification.Channel]string{notification.Webhook: "test"}
		encrypt := func(value string) string {
			t.Helper()
			response, err := crypto.Encrypt(t.Context(), &vaultv1.EncryptRequest{Keyring: event.WorkspaceID, Data: value})
			require.NoError(t, err)
			return response.GetEncrypted()
		}
		config, err := proto.Marshal(&notificationv1.DestinationConfig{Destination: &notificationv1.DestinationConfig_Webhook{
			Webhook: &notificationv1.WebhookConfig{EncryptedUrl: encrypt("https://example.com/hook"), EncryptedSigningSecret: encrypt("signing-secret"),
				Headers: []*notificationv1.WebhookHeader{{Name: "Authorization", EncryptedValue: encrypt("Bearer secret")}},
			},
		}})
		require.NoError(t, err)
		destinationID := uid.New("dest")
		_, err = database.RW().ExecContext(t.Context(), `INSERT INTO notification_destinations (id, workspace_id, name, channel, config, enabled, created_at) VALUES (?, ?, 'Webhook', 'webhook', ?, TRUE, 1)`, destinationID, event.WorkspaceID, config)
		require.NoError(t, err)
		for range 2 {
			subID := uid.New("sub")
			_, err := database.RW().ExecContext(t.Context(), `INSERT INTO notification_subscriptions (id, workspace_id, name, notification_type, enabled, created_at) VALUES (?, ?, 'rule', ?, TRUE, 1)`, subID, event.WorkspaceID, event.Type)
			require.NoError(t, err)
			_, err = database.RW().ExecContext(t.Context(), `INSERT INTO notification_subscription_targets (id, workspace_id, subscription_id, channel, destination_id, created_at) VALUES (?, ?, ?, 'webhook', ?, 1)`, uid.New("target"), event.WorkspaceID, subID, destinationID)
			require.NoError(t, err)
		}
		ref := planOne(t, svc, event)
		provider.failure = &notification.SendError{Kind: notification.Retryable, Code: "rate_limited"}
		result, err := svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, notification.Target{URL: "https://example.com/hook", SigningSecret: "signing-secret", Headers: map[string]string{"Authorization": "Bearer secret"}}, provider.requests[0].Target)
		_, err = database.RW().ExecContext(t.Context(), `UPDATE notification_destinations SET enabled = FALSE WHERE id = ?`, destinationID)
		require.NoError(t, err)
		clk.Tick(result.RetryAfter)
		result, err = svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusSuppressed, result.Status)
		require.Len(t, provider.requests, 1)
	})

	t.Run("invalid destination snapshot records a terminal delivery", func(t *testing.T) {
		svc, event, provider, clk := newService(t)
		provider.channel = notification.Webhook
		svc.defaults = map[notification.Channel]string{notification.Webhook: "test"}
		destinationID := uid.New("dest")
		_, err := database.RW().ExecContext(t.Context(), `INSERT INTO notification_destinations (id, workspace_id, name, channel, config, enabled, created_at) VALUES (?, ?, 'Webhook', 'webhook', '', TRUE, 1)`, destinationID, event.WorkspaceID)
		require.NoError(t, err)
		row, err := svc.freezeDelivery(t.Context(), event, notification.Content{Subject: "Test", Text: "Test"}, recipient{
			key: "destination:" + destinationID, channel: notification.Webhook, config: []byte("invalid protobuf"),
		}, clk.Now().UnixMilli())
		require.NoError(t, err)
		require.NoError(t, database.InsertNotificationDelivery(t.Context(), row))
		result, err := svc.attempt(t.Context(), deliveryRef{WorkspaceID: event.WorkspaceID, ID: row.ID})
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusSuppressed, result.Status)
		stored, err := database.FindNotificationDelivery(t.Context(), db.FindNotificationDeliveryParams{WorkspaceID: event.WorkspaceID, ID: row.ID})
		require.NoError(t, err)
		require.Equal(t, "invalid_snapshot", stored.Reason.String)
		require.Zero(t, stored.AttemptCount)
		require.Empty(t, provider.requests)
	})

	for _, change := range []string{"opt-out", "membership removed", "workspace disabled", "expired"} {
		t.Run(change+" after planning prevents send", func(t *testing.T) {
			svc, event, provider, clk := newService(t)
			event.ExpiresAt = clk.Now().Add(time.Hour).UnixMilli()
			addAlice(t, event)
			ref := planOne(t, svc, event)
			switch change {
			case "opt-out":
				_, err := database.RW().ExecContext(t.Context(), `INSERT INTO notification_preferences (id, workspace_id, user_id, notification_type, channel, enabled, created_at) VALUES (?, ?, 'alice', ?, 'email', FALSE, 1)`, uid.New("pref"), event.WorkspaceID, event.Type)
				require.NoError(t, err)
			case "membership removed":
				svc.directory = &testDirectory{enabled: false}
			case "workspace disabled":
				_, err := database.RW().ExecContext(t.Context(), `UPDATE workspaces SET enabled = FALSE WHERE id = ?`, event.WorkspaceID)
				require.NoError(t, err)
			case "expired":
				clk.Tick(time.Hour)
			}
			result, err := svc.attempt(t.Context(), ref)
			require.NoError(t, err)
			require.Equal(t, db.NotificationDeliveriesStatusSuppressed, result.Status)
			require.Empty(t, provider.requests)
		})
	}

	t.Run("rate limit waits and retries exact delivery", func(t *testing.T) {
		svc, event, provider, clk := newService(t)
		addAlice(t, event)
		ref := planOne(t, svc, event)
		provider.failure = &notification.SendError{Kind: notification.Retryable, Code: "rate_limit", RetryAfter: 37 * time.Second}
		result, err := svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, 37*time.Second, result.RetryAfter)
		_, err = svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Len(t, provider.requests, 1)
		clk.Tick(37 * time.Second)
		provider.failure = nil
		result, err = svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusAccepted, result.Status)
		require.Len(t, provider.requests, 2)
		require.Equal(t, provider.requests[0], provider.requests[1])
	})

	for _, window := range []time.Duration{0, 24 * time.Hour} {
		t.Run("receipt loss with window "+window.String(), func(t *testing.T) {
			svc, event, provider, _ := newService(t)
			provider.window = window
			addAlice(t, event)
			ref := planOne(t, svc, event)
			proxy := &failedReceiptDB{Database: database}
			proxy.fail.Store(true)
			svc.db = proxy
			_, err := svc.attempt(t.Context(), ref)
			require.ErrorContains(t, err, "lost database response")
			row, err := database.FindNotificationDelivery(t.Context(), db.FindNotificationDeliveryParams{WorkspaceID: event.WorkspaceID, ID: ref.ID})
			require.NoError(t, err)
			require.Equal(t, db.NotificationDeliveriesStatusSending, row.Status)
			result, err := svc.attempt(t.Context(), ref)
			require.NoError(t, err)
			if window == 0 {
				require.Equal(t, db.NotificationDeliveriesStatusUnknown, result.Status)
				require.Len(t, provider.requests, 1)
			} else {
				require.Equal(t, db.NotificationDeliveriesStatusAccepted, result.Status)
				require.Len(t, provider.requests, 2)
				require.Equal(t, provider.requests[0], provider.requests[1])
			}
		})
	}

	t.Run("concurrent provider request after timeout retries the same delivery", func(t *testing.T) {
		svc, event, provider, clk := newService(t)
		provider.window = 24 * time.Hour
		addAlice(t, event)
		ref := planOne(t, svc, event)
		for _, failure := range []error{
			errors.New("connection lost after accept"),
			&notification.SendError{Kind: notification.Ambiguous, Code: "concurrent_idempotent_requests"},
		} {
			provider.failure = failure
			result, err := svc.attempt(t.Context(), ref)
			require.NoError(t, err)
			require.Equal(t, db.NotificationDeliveriesStatusSending, result.Status)
			require.Positive(t, result.RetryAfter)
			clk.Tick(result.RetryAfter)
		}
		provider.failure = nil
		result, err := svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusAccepted, result.Status)
		require.Len(t, provider.requests, 3)
		require.Equal(t, provider.requests[0], provider.requests[1])
		require.Equal(t, provider.requests[0], provider.requests[2])
	})

	t.Run("ambiguous result cannot retry after provider key expires", func(t *testing.T) {
		svc, event, provider, clk := newService(t)
		provider.window = 24 * time.Hour
		provider.failure = errors.New("connection lost after accept")
		addAlice(t, event)
		ref := planOne(t, svc, event)
		result, err := svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusSending, result.Status)
		clk.Tick(24*time.Hour - time.Minute)
		result, err = svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusUnknown, result.Status)
		require.Len(t, provider.requests, 1)
	})

	t.Run("bounded retries", func(t *testing.T) {
		svc, event, provider, clk := newService(t)
		provider.failure = &notification.SendError{Kind: notification.Retryable, Code: "rate_limit"}
		addAlice(t, event)
		ref := planOne(t, svc, event)
		for range maxAttempts {
			result, err := svc.attempt(t.Context(), ref)
			require.NoError(t, err)
			clk.Tick(result.RetryAfter)
		}
		result, err := svc.attempt(t.Context(), ref)
		require.NoError(t, err)
		require.Equal(t, db.NotificationDeliveriesStatusFailed, result.Status)
		require.Len(t, provider.requests, 8)
	})

	t.Run("missing audience resolver fails visibly", func(t *testing.T) {
		svc, event, _, _ := newService(t)
		addTarget(t, event, "", sql.NullInt32{Int32: 2, Valid: true}, sql.NullString{})
		_, err := svc.plan(t.Context(), event)
		require.ErrorContains(t, err, "audience has no resolver")
	})

	t.Run("large audience is terminal but explicit recipients still work", func(t *testing.T) {
		svc, event, _, _ := newService(t)
		svc.directory = &testDirectory{enabled: true, audienceTooLarge: true}
		addAlice(t, event)
		refs, err := svc.plan(t.Context(), event)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		addTarget(t, event, "", sql.NullInt32{Int32: int32(notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS), Valid: true}, sql.NullString{})
		event.SourceEventID = "audience-event"
		_, err = svc.plan(t.Context(), event)
		require.True(t, restate.IsTerminalError(err))
		require.ErrorContains(t, err, "directory limit exceeded")
	})

	for _, phase := range []string{"before send", "after send"} {
		t.Run("resume paused delivery "+phase, func(t *testing.T) {
			svc, event, provider, _ := newService(t)
			svc.clock = clock.New()
			directory := &unavailableDirectory{Directory: svc.directory}
			directory.unavailable.Store(phase == "before send")
			svc.directory = directory
			proxy := &failedReceiptDB{Database: database}
			proxy.fail.Store(phase == "after send")
			svc.db = proxy
			addAlice(t, event)
			definition := restate.NewObject(notification.DeliveryServiceName).
				Handler("Deliver", restate.NewObjectHandler(svc.Deliver, restate.WithInvocationRetryPolicy(
					restate.WithMaxRetryAttempts(1), restate.PauseOnMaxAttempts(),
				)))
			runtime := containers.Restate(t, svc.Definition(), definition)
			client, err := notification.NewClient(runtime.IngressClient)
			require.NoError(t, err)
			_, err = client.Notify(t.Context(), event)
			require.NoError(t, err)
			var paused invocationState
			require.Eventually(t, func() bool {
				rows, err := notificationInvocations(t.Context(), runtime.AdminURL)
				if err != nil {
					t.Error(err)
					return false
				}
				for _, row := range rows {
					if row.Status == "paused" {
						paused = row
						return true
					}
				}
				return false
			}, 30*time.Second, 50*time.Millisecond)
			deliveryID := notification.DeliveryID(event.ID(), notification.Email, "user:alice")
			require.Equal(t, deliveryID, paused.Key)
			require.NotEmpty(t, paused.DeploymentID)
			directory.unavailable.Store(false)
			status, _, err := notificationAdminRequest(t.Context(), runtime.AdminURL, http.MethodPatch, "/invocations/"+paused.ID+"/resume?deployment=keep", nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)
			require.Eventually(t, func() bool {
				rows, err := notificationInvocations(t.Context(), runtime.AdminURL)
				if err != nil {
					t.Error(err)
					return false
				}
				for _, row := range rows {
					if row.ID == paused.ID && row.Status == "completed" {
						return row.DeploymentID == paused.DeploymentID
					}
				}
				return false
			}, 30*time.Second, 50*time.Millisecond)
			row, err := database.FindNotificationDelivery(t.Context(), db.FindNotificationDeliveryParams{WorkspaceID: event.WorkspaceID, ID: deliveryID})
			require.NoError(t, err)
			want := db.NotificationDeliveriesStatusAccepted
			if phase == "after send" {
				want = db.NotificationDeliveriesStatusUnknown
			}
			require.Equal(t, want, row.Status)
			status, _, err = notificationAdminRequest(t.Context(), runtime.AdminURL, http.MethodPatch, "/invocations/"+paused.ID+"/resume?deployment=keep", nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusConflict, status)
			provider.mu.Lock()
			defer provider.mu.Unlock()
			require.Len(t, provider.requests, 1)
			require.Equal(t, deliveryID, provider.requests[0].DeliveryID)
		})
	}

	t.Run("restate fanout receipt loss and repeated ingress", func(t *testing.T) {
		svc, event, provider, _ := newService(t)
		svc.clock = clock.New()
		provider.window = 24 * time.Hour
		proxy := &failedReceiptDB{Database: database}
		proxy.fail.Store(true)
		svc.db = proxy
		addAlice(t, event)
		addTarget(t, event, "bob", sql.NullInt32{}, sql.NullString{})
		runtime := containers.Restate(t, svc.Definition(), svc.DeliveryDefinition())
		client, err := notification.NewClient(runtime.IngressClient)
		require.NoError(t, err)
		for range 10 {
			id, err := client.Notify(t.Context(), event)
			require.NoError(t, err)
			require.Equal(t, event.ID(), id)
		}
		_, err = ingress.Object[notification.Event, string](runtime.IngressClient, notification.ServiceName, event.ID(), "Notify").Request(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, int32(2), proxy.plans.Load())
		require.Eventually(t, func() bool {
			rows, err := database.ListNotificationDeliveries(t.Context(), db.ListNotificationDeliveriesParams{WorkspaceID: event.WorkspaceID, NotificationID: event.ID()})
			return err == nil && len(rows) == 2 && rows[0].Status == db.NotificationDeliveriesStatusAccepted && rows[1].Status == db.NotificationDeliveriesStatusAccepted
		}, 30*time.Second, 50*time.Millisecond)
		provider.mu.Lock()
		defer provider.mu.Unlock()
		require.Len(t, provider.requests, 3)
		unique := make(map[string]notification.Request)
		for _, request := range provider.requests {
			if previous, found := unique[request.DeliveryID]; found {
				require.Equal(t, previous, request)
			}
			unique[request.DeliveryID] = request
		}
		require.Len(t, unique, 2)
	})
}

type invocationState struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Key          string `json:"target_service_key"`
	DeploymentID string `json:"pinned_deployment_id"`
}

type invocationQueryResult struct {
	Rows []invocationState `json:"rows"`
}

func notificationInvocations(ctx context.Context, adminURL string) ([]invocationState, error) {
	query := map[string]string{"query": "SELECT id, status, target_service_key, pinned_deployment_id FROM sys_invocation WHERE target_service_name = 'notification.v1.DeliveryService'"}
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	status, response, err := notificationAdminRequest(ctx, adminURL, http.MethodPost, "/query", body)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("invocation query returned %d: %s", status, response)
	}
	var result invocationQueryResult
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, err
	}
	return result.Rows, nil
}

func notificationAdminRequest(ctx context.Context, adminURL, method, path string, body []byte) (int, []byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, adminURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, err
	}
	data, err := io.ReadAll(response.Body)
	err = errors.Join(err, response.Body.Close())
	return response.StatusCode, data, err
}
