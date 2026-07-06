package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func newTestDB(t *testing.T) db.Database {
	t.Helper()
	cfg := containers.MySQL(t)
	database, err := db.New(db.Config{
		PrimaryDSN:  cfg.DSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func insertCard(t *testing.T, database db.Database, workspaceID, name string, selectable bool, centsPerUnit string) string {
	t.Helper()
	id := uid.New(uid.RateCardPrefix)
	config, err := json.Marshal(ratecard.Config{
		Verifications: []ratecard.Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: &centsPerUnit}},
		Credits:       nil,
		Ratelimits:    nil,
	})
	require.NoError(t, err)
	require.NoError(t, db.Query.InsertRateCard(context.Background(), database.RW(), db.InsertRateCardParams{
		ID:          id,
		WorkspaceID: workspaceID,
		Name:        name,
		Currency:    "USD",
		Config:      config,
		Selectable:  selectable,
		CreatedAt:   time.Now().UnixMilli(),
	}))
	return id
}

func insertIdentity(t *testing.T, database db.Database, workspaceID, externalID string) string {
	t.Helper()
	id := uid.New(uid.IdentityPrefix)
	require.NoError(t, db.Query.InsertIdentity(context.Background(), database.RW(), db.InsertIdentityParams{
		ID:          id,
		ExternalID:  externalID,
		WorkspaceID: workspaceID,
		Environment: "default",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        json.RawMessage("{}"),
	}))
	return id
}

func setDefault(t *testing.T, database db.Database, workspaceID, rateCardID string) {
	t.Helper()
	require.NoError(t, db.Query.UpsertWorkspaceBillingSettingsDefaultRateCard(context.Background(), database.RW(), db.UpsertWorkspaceBillingSettingsDefaultRateCardParams{
		ID:                uid.New(uid.RateCardPrefix),
		WorkspaceID:       workspaceID,
		DefaultRateCardID: sql.NullString{Valid: rateCardID != "", String: rateCardID},
		CreatedAt:         time.Now().UnixMilli(),
	}))
}

func TestResolvePrecedence(t *testing.T) {
	database := newTestDB(t)
	resolver := NewResolver(database)
	ctx := context.Background()

	workspaceID := uid.New(uid.WorkspacePrefix)
	defaultCard := insertCard(t, database, workspaceID, "default", false, "0.1")
	assignedCard := insertCard(t, database, workspaceID, "assigned", false, "0.2")
	selectableCard := insertCard(t, database, workspaceID, "pro", true, "0.3")
	notSelectableCard := insertCard(t, database, workspaceID, "private", false, "0.4")
	setDefault(t, database, workspaceID, defaultCard)

	t.Run("falls back to workspace default", func(t *testing.T) {
		identityID := insertIdentity(t, database, workspaceID, "user_default")
		resolved, err := resolver.Resolve(ctx, workspaceID, identityID, 2026, 6)
		require.NoError(t, err)
		require.Equal(t, defaultCard, resolved.Card.ID)
		require.Equal(t, ResolvedFromWorkspaceDefault, resolved.ResolvedFrom)
		require.False(t, resolved.Recorded)
	})

	t.Run("assignment beats default", func(t *testing.T) {
		identityID := insertIdentity(t, database, workspaceID, "user_assigned")
		require.NoError(t, db.Query.UpdateIdentityRateCard(ctx, database.RW(), db.UpdateIdentityRateCardParams{
			RateCardID:  sql.NullString{Valid: true, String: assignedCard},
			WorkspaceID: workspaceID,
			IdentityID:  identityID,
		}))
		resolved, err := resolver.Resolve(ctx, workspaceID, identityID, 2026, 6)
		require.NoError(t, err)
		require.Equal(t, assignedCard, resolved.Card.ID)
		require.Equal(t, ResolvedFromAssignment, resolved.ResolvedFrom)
	})

	t.Run("selection beats assignment", func(t *testing.T) {
		identityID := insertIdentity(t, database, workspaceID, "user_selected")
		require.NoError(t, db.Query.UpdateIdentityRateCard(ctx, database.RW(), db.UpdateIdentityRateCardParams{
			RateCardID:  sql.NullString{Valid: true, String: assignedCard},
			WorkspaceID: workspaceID,
			IdentityID:  identityID,
		}))
		require.NoError(t, db.Query.UpdateIdentitySelectedRateCard(ctx, database.RW(), db.UpdateIdentitySelectedRateCardParams{
			SelectedRateCardID: sql.NullString{Valid: true, String: selectableCard},
			WorkspaceID:        workspaceID,
			IdentityID:         identityID,
		}))
		resolved, err := resolver.Resolve(ctx, workspaceID, identityID, 2026, 6)
		require.NoError(t, err)
		require.Equal(t, selectableCard, resolved.Card.ID)
		require.Equal(t, ResolvedFromSelection, resolved.ResolvedFrom)
	})

	t.Run("selection of a non-selectable card is ignored", func(t *testing.T) {
		identityID := insertIdentity(t, database, workspaceID, "user_bad_selection")
		require.NoError(t, db.Query.UpdateIdentitySelectedRateCard(ctx, database.RW(), db.UpdateIdentitySelectedRateCardParams{
			SelectedRateCardID: sql.NullString{Valid: true, String: notSelectableCard},
			WorkspaceID:        workspaceID,
			IdentityID:         identityID,
		}))
		resolved, err := resolver.Resolve(ctx, workspaceID, identityID, 2026, 6)
		require.NoError(t, err)
		require.Equal(t, defaultCard, resolved.Card.ID)
		require.Equal(t, ResolvedFromWorkspaceDefault, resolved.ResolvedFrom)
	})

	t.Run("no cards anywhere yields ErrNoRateCard", func(t *testing.T) {
		otherWorkspace := uid.New(uid.WorkspacePrefix)
		identityID := insertIdentity(t, database, otherWorkspace, "user_nocard")
		_, err := resolver.Resolve(ctx, otherWorkspace, identityID, 2026, 6)
		require.ErrorIs(t, err, ErrNoRateCard)
	})
}

func TestResolveAndRecordPinsThePeriod(t *testing.T) {
	database := newTestDB(t)
	resolver := NewResolver(database)
	ctx := context.Background()

	workspaceID := uid.New(uid.WorkspacePrefix)
	firstCard := insertCard(t, database, workspaceID, "first", false, "0.1")
	secondCard := insertCard(t, database, workspaceID, "second", false, "0.2")
	setDefault(t, database, workspaceID, firstCard)
	identityID := insertIdentity(t, database, workspaceID, "user_pinned")

	resolved, err := resolver.ResolveAndRecord(ctx, workspaceID, identityID, 2026, 6)
	require.NoError(t, err)
	require.Equal(t, firstCard, resolved.Card.ID)
	require.True(t, resolved.Recorded)

	// A mid-period change (assignment to secondCard) must not re-price the
	// recorded period, but applies to the next one.
	require.NoError(t, db.Query.UpdateIdentityRateCard(ctx, database.RW(), db.UpdateIdentityRateCardParams{
		RateCardID:  sql.NullString{Valid: true, String: secondCard},
		WorkspaceID: workspaceID,
		IdentityID:  identityID,
	}))

	resolved, err = resolver.ResolveAndRecord(ctx, workspaceID, identityID, 2026, 6)
	require.NoError(t, err)
	require.Equal(t, firstCard, resolved.Card.ID, "recorded period must keep its card")

	next, err := resolver.ResolveAndRecord(ctx, workspaceID, identityID, 2026, 7)
	require.NoError(t, err)
	require.Equal(t, secondCard, next.Card.ID, "next period picks up the change")
	require.Equal(t, ResolvedFromAssignment, next.ResolvedFrom)
}

func TestResolvedCardPrices(t *testing.T) {
	database := newTestDB(t)
	resolver := NewResolver(database)
	ctx := context.Background()

	workspaceID := uid.New(uid.WorkspacePrefix)
	cardID := insertCard(t, database, workspaceID, "metered", false, "0.5")
	setDefault(t, database, workspaceID, cardID)
	identityID := insertIdentity(t, database, workspaceID, "user_priced")

	resolved, err := resolver.Resolve(ctx, workspaceID, identityID, 2026, 6)
	require.NoError(t, err)

	amounts, err := resolved.Price(100, 0, 0)
	require.NoError(t, err)
	require.Equal(t, "50", ratecard.CentsString(amounts.VerificationsCents))
	require.Equal(t, "50", ratecard.CentsString(amounts.TotalCents))
}

// TestBatchResolverMatchesSingleShot proves the caching batch path resolves and
// pins the same cards the single-shot resolver reads back (R19 parity), and
// that a repeated resolution in the same batch is stable (served from cache).
func TestBatchResolverMatchesSingleShot(t *testing.T) {
	database := newTestDB(t)
	resolver := NewResolver(database)
	ctx := context.Background()

	workspaceID := uid.New(uid.WorkspacePrefix)
	defaultCard := insertCard(t, database, workspaceID, "default", false, "0.1")
	selectableCard := insertCard(t, database, workspaceID, "selectable", true, "0.2")
	setDefault(t, database, workspaceID, defaultCard)

	// idA falls through to the workspace default; idB has an end-user selection.
	idA := insertIdentity(t, database, workspaceID, "user_a")
	idB := insertIdentity(t, database, workspaceID, "user_b")
	require.NoError(t, db.Query.UpdateIdentitySelectedRateCard(ctx, database.RW(), db.UpdateIdentitySelectedRateCardParams{
		SelectedRateCardID: sql.NullString{Valid: true, String: selectableCard},
		WorkspaceID:        workspaceID,
		IdentityID:         idB,
	}))

	loadIdentity := func(id string) db.Identity {
		row, err := db.Query.FindIdentityByID(ctx, database.RO(), db.FindIdentityByIDParams{
			WorkspaceID: workspaceID, IdentityID: id, Deleted: false,
		})
		require.NoError(t, err)
		return row
	}

	year, month := 2027, 3
	batch := resolver.NewBatch(workspaceID)

	ra, err := batch.ResolveAndRecord(ctx, loadIdentity(idA), year, month)
	require.NoError(t, err)
	require.Equal(t, defaultCard, ra.Card.ID)
	require.Equal(t, ResolvedFromWorkspaceDefault, ra.ResolvedFrom)
	require.True(t, ra.Recorded)
	require.False(t, ra.AlreadyPushed)

	rb, err := batch.ResolveAndRecord(ctx, loadIdentity(idB), year, month)
	require.NoError(t, err)
	require.Equal(t, selectableCard, rb.Card.ID)
	require.Equal(t, ResolvedFromSelection, rb.ResolvedFrom)

	// The single-shot resolver reads back the SAME pinned cards.
	sa, err := resolver.Resolve(ctx, workspaceID, idA, year, month)
	require.NoError(t, err)
	require.Equal(t, ra.Card.ID, sa.Card.ID)
	require.Equal(t, ra.ResolvedFrom, sa.ResolvedFrom)
	require.True(t, sa.Recorded)

	sb, err := resolver.Resolve(ctx, workspaceID, idB, year, month)
	require.NoError(t, err)
	require.Equal(t, rb.Card.ID, sb.Card.ID)
	require.Equal(t, rb.ResolvedFrom, sb.ResolvedFrom)

	// Re-resolving idA in the same batch is stable (cache-served) and stays
	// recorded — not pushed until MarkPushed runs.
	again, err := batch.ResolveAndRecord(ctx, loadIdentity(idA), year, month)
	require.NoError(t, err)
	require.Equal(t, defaultCard, again.Card.ID)
	require.True(t, again.Recorded)
	require.False(t, again.AlreadyPushed)

	// After marking pushed, the batch reports it (run-once signal).
	require.NoError(t, batch.MarkPushed(ctx, idA, year, month))
	pushed, err := resolver.NewBatch(workspaceID).ResolveAndRecord(ctx, loadIdentity(idA), year, month)
	require.NoError(t, err)
	require.True(t, pushed.AlreadyPushed)
}
