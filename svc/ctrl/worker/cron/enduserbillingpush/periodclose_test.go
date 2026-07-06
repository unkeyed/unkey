package enduserbillingpush

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/enduserbilling"
)

type fakeUsage struct {
	rows map[string][]clickhouse.IdentityBillableUsage
}

func (f *fakeUsage) GetBillableUsagePerIdentity(ctx context.Context, workspaceID string, year, month int, enabledKeyspaces, enabledNamespaces []string) ([]clickhouse.IdentityBillableUsage, error) {
	return f.rows[workspaceID], nil
}

// fakeVault "decrypts" by stripping a prefix, mirroring encrypt-side storage
// of "enc:" + plaintext in this test.
type fakeVault struct{}

func (f *fakeVault) Decrypt(ctx context.Context, keyring, encrypted string) (string, error) {
	return encrypted[len("enc:"):], nil
}

type recordingPusher struct {
	requests []enduserbilling.PushRequest
}

func (r *recordingPusher) Push(ctx context.Context, req enduserbilling.PushRequest) (int, error) {
	r.requests = append(r.requests, req)
	return len(req.Records), nil
}

func TestPeriodClose(t *testing.T) {
	cfg := containers.MySQL(t)
	database, err := db.New(db.Config{PrimaryDSN: cfg.DSN, ReadOnlyDSN: ""})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()

	workspaceID := uid.New(uid.WorkspacePrefix)

	// Rate card: 1 cent per verification, workspace default.
	cents := "1"
	config, err := json.Marshal(ratecard.Config{
		Verifications: []ratecard.Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: &cents}},
		Credits:       nil,
		Ratelimits:    nil,
	})
	require.NoError(t, err)
	cardID := uid.New(uid.RateCardPrefix)
	require.NoError(t, db.Query.InsertRateCard(ctx, database.RW(), db.InsertRateCardParams{
		ID:          cardID,
		WorkspaceID: workspaceID,
		Name:        "default",
		Currency:    "USD",
		Config:      config,
		Selectable:  false,
		CreatedAt:   time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.UpsertWorkspaceBillingSettingsDefaultRateCard(ctx, database.RW(), db.UpsertWorkspaceBillingSettingsDefaultRateCardParams{
		ID:                uid.New(uid.RateCardPrefix),
		WorkspaceID:       workspaceID,
		DefaultRateCardID: sql.NullString{Valid: true, String: cardID},
		CreatedAt:         time.Now().UnixMilli(),
	}))
	// Link a "connected account" (fake vault: ciphertext is enc:acct_ws1).
	require.NoError(t, db.Query.SetWorkspaceBillingStripeConnect(ctx, database.RW(), db.SetWorkspaceBillingStripeConnectParams{
		ID:                           uid.New(uid.RateCardPrefix),
		WorkspaceID:                  workspaceID,
		StripeConnectEncrypted:       sql.NullString{Valid: true, String: "enc:acct_ws1"},
		StripeConnectEncryptionKeyID: sql.NullString{Valid: true, String: "key_1"},
		StripeConnectStatus: db.NullWorkspaceBillingSettingsStripeConnectStatus{
			Valid: true,
			WorkspaceBillingSettingsStripeConnectStatus: db.WorkspaceBillingSettingsStripeConnectStatusLinked,
		},
		CreatedAt: time.Now().UnixMilli(),
	}))

	// Enable one keyspace for billing so the workspace is not skipped as
	// "nothing enabled". The fake usage reader ignores the filter, so which
	// keyspace is enabled does not change the returned rows here — this only
	// exercises the enablement gate in runWorkspace.
	require.NoError(t, db.Query.UpsertBillingBillableResource(ctx, database.RW(), db.UpsertBillingBillableResourceParams{
		ID:           uid.New(uid.RateCardPrefix),
		WorkspaceID:  workspaceID,
		ResourceType: db.BillingBillableResourcesResourceTypeKeyspace,
		ResourceID:   uid.New(uid.KeySpacePrefix),
		CreatedAt:    time.Now().UnixMilli(),
	}))

	// Two identities bound to stripe_connect (one with a provider customer),
	// one bound to export (must be skipped on the push path).
	newIdentity := func(externalID string, provider db.IdentitiesBillingProvider, customer string) string {
		id := uid.New(uid.IdentityPrefix)
		require.NoError(t, db.Query.InsertIdentity(ctx, database.RW(), db.InsertIdentityParams{
			ID:          id,
			ExternalID:  externalID,
			WorkspaceID: workspaceID,
			Environment: "default",
			CreatedAt:   time.Now().UnixMilli(),
			Meta:        json.RawMessage("{}"),
		}))
		require.NoError(t, db.Query.UpdateIdentityBillingBinding(ctx, database.RW(), db.UpdateIdentityBillingBindingParams{
			BillingProvider:           provider,
			BillingExternalCustomerID: sql.NullString{Valid: customer != "", String: customer},
			WorkspaceID:               workspaceID,
			IdentityID:                id,
		}))
		return id
	}
	stripeBound := newIdentity("user_stripe", db.IdentitiesBillingProviderStripeConnect, "cus_1")
	exportBound := newIdentity("user_export", db.IdentitiesBillingProviderExport, "")

	usage := &fakeUsage{rows: map[string][]clickhouse.IdentityBillableUsage{
		workspaceID: {
			{WorkspaceID: workspaceID, IdentityID: stripeBound, ExternalID: "user_stripe", Verifications: 120, SpentCredits: 0, RatelimitsPassed: 0},
			{WorkspaceID: workspaceID, IdentityID: exportBound, ExternalID: "user_export", Verifications: 999, SpentCredits: 0, RatelimitsPassed: 0},
		},
	}}
	pusher := &recordingPusher{requests: nil}

	pc, err := New(Config{DB: database, Usage: usage, Vault: &fakeVault{}, Pusher: pusher})
	require.NoError(t, err)

	summary, err := pc.Run(ctx, 2026, 6)
	require.NoError(t, err)
	require.Empty(t, summary.Errors)
	require.GreaterOrEqual(t, summary.Workspaces, 1)
	require.Equal(t, 1, summary.RecordsPushed)

	// Find our workspace's push (a reused MySQL container may carry linked
	// workspaces from other tests; those have no usage and push nothing).
	var req *enduserbilling.PushRequest
	for i := range pusher.requests {
		if pusher.requests[i].WorkspaceID == workspaceID {
			req = &pusher.requests[i]
		}
	}
	require.NotNil(t, req)
	require.Equal(t, "acct_ws1", req.ConnectedAccountID, "vault-decrypted account id reaches the pusher")
	require.Len(t, req.Records, 1, "export-bound identities are skipped on the push path")
	record := req.Records[0]
	require.Equal(t, stripeBound, record.IdentityID)
	require.Equal(t, "cus_1", record.ProviderCustomerID)
	require.Equal(t, cardID, record.RateCardID)
	require.Equal(t, int64(120), record.Verifications)
	require.Equal(t, int64(120), record.VerificationsCents, "120 verifications at 1 cent each")
	require.Equal(t, "USD", record.Currency)

	// The run pinned the period card (R18) and stamped it pushed so re-ticks
	// skip it (run-once: the durable defense against double-billing once
	// Stripe's 24h invoice-item idempotency window lapses).
	recorded, err := db.Query.FindBillingPeriodRateCard(ctx, database.RO(), db.FindBillingPeriodRateCardParams{
		WorkspaceID: workspaceID,
		IdentityID:  stripeBound,
		Year:        2026,
		Month:       6,
	})
	require.NoError(t, err)
	require.Equal(t, cardID, recorded.RateCardID)
	require.True(t, recorded.PushedAt.Valid, "a successful push stamps pushed_at")

	// Re-running the same closed period pushes NOTHING for this workspace: the
	// identity is already marked pushed, so the additive invoice item is not
	// re-sent on the hourly cadence.
	before := len(pusher.requests)
	summary, err = pc.Run(ctx, 2026, 6)
	require.NoError(t, err)
	require.Empty(t, summary.Errors)
	require.Equal(t, 0, summary.RecordsPushed, "an already-pushed period bills nothing on re-run")
	for i := before; i < len(pusher.requests); i++ {
		require.NotEqual(t, workspaceID, pusher.requests[i].WorkspaceID,
			"already-pushed identity must not be pushed again")
	}
}

// TestPeriodCloseSkipsWorkspaceWithNothingEnabled proves KTD3: a
// Stripe-connected workspace that has enabled no keyspaces or namespaces bills
// nothing, even when the identity has usage and resolves to a rate card.
func TestPeriodCloseSkipsWorkspaceWithNothingEnabled(t *testing.T) {
	cfg := containers.MySQL(t)
	database, err := db.New(db.Config{PrimaryDSN: cfg.DSN, ReadOnlyDSN: ""})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	ctx := context.Background()

	workspaceID := uid.New(uid.WorkspacePrefix)

	cents := "1"
	config, err := json.Marshal(ratecard.Config{
		Verifications: []ratecard.Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: &cents}},
		Credits:       nil,
		Ratelimits:    nil,
	})
	require.NoError(t, err)
	cardID := uid.New(uid.RateCardPrefix)
	require.NoError(t, db.Query.InsertRateCard(ctx, database.RW(), db.InsertRateCardParams{
		ID: cardID, WorkspaceID: workspaceID, Name: "default", Currency: "USD",
		Config: config, Selectable: false, CreatedAt: time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.UpsertWorkspaceBillingSettingsDefaultRateCard(ctx, database.RW(), db.UpsertWorkspaceBillingSettingsDefaultRateCardParams{
		ID: uid.New(uid.RateCardPrefix), WorkspaceID: workspaceID,
		DefaultRateCardID: sql.NullString{Valid: true, String: cardID}, CreatedAt: time.Now().UnixMilli(),
	}))
	require.NoError(t, db.Query.SetWorkspaceBillingStripeConnect(ctx, database.RW(), db.SetWorkspaceBillingStripeConnectParams{
		ID: uid.New(uid.RateCardPrefix), WorkspaceID: workspaceID,
		StripeConnectEncrypted:       sql.NullString{Valid: true, String: "enc:acct_skip"},
		StripeConnectEncryptionKeyID: sql.NullString{Valid: true, String: "key_1"},
		StripeConnectStatus: db.NullWorkspaceBillingSettingsStripeConnectStatus{
			Valid: true,
			WorkspaceBillingSettingsStripeConnectStatus: db.WorkspaceBillingSettingsStripeConnectStatusLinked,
		},
		CreatedAt: time.Now().UnixMilli(),
	}))

	// A stripe-bound identity WITH usage, but NO billing_billable_resources row.
	identityID := uid.New(uid.IdentityPrefix)
	require.NoError(t, db.Query.InsertIdentity(ctx, database.RW(), db.InsertIdentityParams{
		ID: identityID, ExternalID: "user_skip", WorkspaceID: workspaceID,
		Environment: "default", CreatedAt: time.Now().UnixMilli(), Meta: json.RawMessage("{}"),
	}))
	require.NoError(t, db.Query.UpdateIdentityBillingBinding(ctx, database.RW(), db.UpdateIdentityBillingBindingParams{
		BillingProvider:           db.IdentitiesBillingProviderStripeConnect,
		BillingExternalCustomerID: sql.NullString{Valid: true, String: "cus_skip"},
		WorkspaceID:               workspaceID,
		IdentityID:                identityID,
	}))

	usage := &fakeUsage{rows: map[string][]clickhouse.IdentityBillableUsage{
		workspaceID: {{WorkspaceID: workspaceID, IdentityID: identityID, ExternalID: "user_skip", Verifications: 500}},
	}}
	pusher := &recordingPusher{requests: nil}
	pc, err := New(Config{DB: database, Usage: usage, Vault: &fakeVault{}, Pusher: pusher})
	require.NoError(t, err)

	summary, err := pc.Run(ctx, 2026, 6)
	require.NoError(t, err)
	require.Empty(t, summary.Errors)
	require.Equal(t, 0, summary.RecordsPushed, "workspace with nothing enabled bills nothing")
	for i := range pusher.requests {
		require.NotEqual(t, workspaceID, pusher.requests[i].WorkspaceID,
			"a workspace with no enabled resources must not be pushed")
	}
}
