// Package billing resolves which rate card is in force for an end-user
// identity and billing period, and prices billable quantities against it.
// It is the single amount-computation path shared by the billing export API
// and the period-close push, so both produce identical money (R19).
package billing

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ratecard"
	"github.com/unkeyed/unkey/pkg/uid"
)

// ErrNoRateCard means no rate card resolves for the identity: no valid
// end-user selection, no valid assignment, and no workspace default.
var ErrNoRateCard = errors.New("no rate card resolves for this identity")

// ResolvedFrom names the precedence step that produced the card, matching
// the billing_period_rate_cards.resolved_from enum.
type ResolvedFrom string

const (
	ResolvedFromSelection        ResolvedFrom = "selection"
	ResolvedFromAssignment       ResolvedFrom = "assignment"
	ResolvedFromWorkspaceDefault ResolvedFrom = "workspace_default"
)

// ResolvedRateCard is the card in force for one identity and period.
type ResolvedRateCard struct {
	Card         db.RateCard
	Config       ratecard.Config
	ResolvedFrom ResolvedFrom
	// Recorded is true when the resolution came from (or was persisted to)
	// the period record, making it immutable for this period (R18).
	Recorded bool
	// AlreadyPushed is true when the period record has been stamped as pushed
	// to the billing provider. The period-close skips these so a closed period
	// bills exactly once even though the cron re-ticks hourly (defeats the
	// double-bill that Stripe's 24h idempotency window cannot).
	AlreadyPushed bool
}

// Resolver implements the KTD7 precedence: recorded period row, else
// end-user selection (if still selectable), else customer assignment, else
// workspace default. Mid-period card changes only affect periods that have
// not been recorded yet.
type Resolver struct {
	database db.Database
}

func NewResolver(database db.Database) *Resolver {
	return &Resolver{database: database}
}

// Resolve returns the card in force for the identity and period without
// persisting anything. Use ResolveAndRecord on billing paths.
func (r *Resolver) Resolve(ctx context.Context, workspaceID, identityID string, year, month int) (ResolvedRateCard, error) {
	recorded, err := db.Query.FindBillingPeriodRateCard(ctx, r.database.RO(), db.FindBillingPeriodRateCardParams{
		WorkspaceID: workspaceID,
		IdentityID:  identityID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err == nil {
		card, cardErr := r.loadCard(ctx, workspaceID, recorded.RateCardID, false)
		if cardErr != nil {
			return ResolvedRateCard{}, cardErr
		}
		card.ResolvedFrom = ResolvedFrom(recorded.ResolvedFrom)
		card.Recorded = true
		card.AlreadyPushed = recorded.PushedAt.Valid
		return card, nil
	}
	if !db.IsNotFound(err) {
		return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("failed to look up period rate card record"))
	}

	return r.resolveLive(ctx, workspaceID, identityID)
}

// ResolveAndRecord resolves the card and persists it against the period,
// first write wins: a concurrent recorder or an earlier billing run keeps
// its card, and this call returns whatever ended up recorded (R18).
func (r *Resolver) ResolveAndRecord(ctx context.Context, workspaceID, identityID string, year, month int) (ResolvedRateCard, error) {
	resolved, err := r.Resolve(ctx, workspaceID, identityID, year, month)
	if err != nil {
		return ResolvedRateCard{}, err
	}
	if resolved.Recorded {
		return resolved, nil
	}

	err = db.Query.InsertBillingPeriodRateCard(ctx, r.database.RW(), db.InsertBillingPeriodRateCardParams{
		ID:           uid.New(uid.BillingPeriodRateCardPrefix),
		WorkspaceID:  workspaceID,
		IdentityID:   identityID,
		Year:         int32(year),
		Month:        int32(month),
		RateCardID:   resolved.Card.ID,
		ResolvedFrom: db.BillingPeriodRateCardsResolvedFrom(resolved.ResolvedFrom),
		CreatedAt:    time.Now().UnixMilli(),
	})
	if err != nil {
		return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("failed to record period rate card"))
	}

	// Re-read: INSERT IGNORE means a concurrent writer may have won.
	return r.Resolve(ctx, workspaceID, identityID, year, month)
}

// MarkPushed stamps the period record as successfully pushed to the billing
// provider. First-write-wins on the timestamp (the query COALESCEs), so a
// retry after a crash keeps the original push time. A subsequent Resolve for
// the same period then reports AlreadyPushed, which the period-close uses to
// bill a closed period exactly once across hourly re-ticks.
func (r *Resolver) MarkPushed(ctx context.Context, workspaceID, identityID string, year, month int) error {
	now := time.Now().UnixMilli()
	err := db.Query.MarkBillingPeriodRateCardPushed(ctx, r.database.RW(), db.MarkBillingPeriodRateCardPushedParams{
		PushedAt:    sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
		WorkspaceID: workspaceID,
		IdentityID:  identityID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to mark period rate card pushed"))
	}
	return nil
}

// ResolveLive applies the live precedence (selection -> assignment ->
// workspace default) ignoring any recorded period — the card that will
// govern periods not yet billed.
func (r *Resolver) ResolveLive(ctx context.Context, workspaceID, identityID string) (ResolvedRateCard, error) {
	return r.resolveLive(ctx, workspaceID, identityID)
}

// resolveLive applies selection -> assignment -> workspace default.
func (r *Resolver) resolveLive(ctx context.Context, workspaceID, identityID string) (ResolvedRateCard, error) {
	identity, err := db.Query.FindIdentityByID(ctx, r.database.RO(), db.FindIdentityByIDParams{
		WorkspaceID: workspaceID,
		IdentityID:  identityID,
		Deleted:     false,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("identity not found"))
		}
		return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("failed to load identity"))
	}

	load := func(rateCardID string, rejectArchived bool) (ResolvedRateCard, error) {
		return r.loadCard(ctx, workspaceID, rateCardID, rejectArchived)
	}
	defaultCardID := func() (sql.NullString, error) {
		settings, sErr := db.Query.FindWorkspaceBillingSettings(ctx, r.database.RO(), workspaceID)
		if sErr != nil {
			if db.IsNotFound(sErr) {
				return sql.NullString{}, nil
			}
			return sql.NullString{}, fault.Wrap(sErr, fault.Internal("failed to load workspace billing settings"))
		}
		return settings.DefaultRateCardID, nil
	}
	return resolveLiveForIdentity(identity, load, defaultCardID)
}

// cardLoader fetches and parses a rate card, treating an archived card as not
// found when rejectArchived is set. A batch caller backs it with a cache.
type cardLoader func(rateCardID string, rejectArchived bool) (ResolvedRateCard, error)

// resolveLiveForIdentity applies the KTD7 live precedence — end-user selection
// (only while still selectable), then customer assignment, then workspace
// default — for an already-loaded identity. Card reads go through load and the
// workspace default id through defaultCardID, so a batch caller can cache both;
// the single-shot and batch paths share this one precedence so R19 amount
// parity cannot drift between them. defaultCardID is a thunk so the settings
// read only happens when no selection/assignment card wins.
func resolveLiveForIdentity(identity db.Identity, load cardLoader, defaultCardID func() (sql.NullString, error)) (ResolvedRateCard, error) {
	// End-user selection wins only while the card is still in the workspace's
	// selectable set; a revoked or archived card falls through silently.
	if identity.SelectedRateCardID.Valid {
		card, cardErr := load(identity.SelectedRateCardID.String, true)
		if cardErr == nil && card.Card.Selectable {
			card.ResolvedFrom = ResolvedFromSelection
			return card, nil
		}
	}

	if identity.RateCardID.Valid {
		card, cardErr := load(identity.RateCardID.String, true)
		if cardErr == nil {
			card.ResolvedFrom = ResolvedFromAssignment
			return card, nil
		}
	}

	dc, err := defaultCardID()
	if err != nil {
		return ResolvedRateCard{}, err
	}
	if dc.Valid {
		card, cardErr := load(dc.String, true)
		if cardErr == nil {
			card.ResolvedFrom = ResolvedFromWorkspaceDefault
			return card, nil
		}
	}

	return ResolvedRateCard{}, ErrNoRateCard
}

// loadCard fetches a card and parses its config. When rejectArchived is set,
// archived cards are treated as not found — they stay resolvable only through
// recorded period rows.
func (r *Resolver) loadCard(ctx context.Context, workspaceID, rateCardID string, rejectArchived bool) (ResolvedRateCard, error) {
	card, err := db.Query.FindRateCardByID(ctx, r.database.RO(), db.FindRateCardByIDParams{
		WorkspaceID: workspaceID,
		RateCardID:  rateCardID,
	})
	if err != nil {
		return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("rate card not found"))
	}
	if rejectArchived && card.Archived {
		return ResolvedRateCard{}, fault.New("rate card is archived")
	}
	cfg, err := ratecard.ParseConfig(card.Config)
	if err != nil {
		return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("failed to parse rate card config"))
	}
	return ResolvedRateCard{Card: card, Config: cfg, ResolvedFrom: "", Recorded: false, AlreadyPushed: false}, nil
}

// Price computes the exact per-dimension and total cents for the quantities
// under the resolved card.
func (c ResolvedRateCard) Price(verifications, credits, ratelimits int64) (ratecard.Amounts, error) {
	return c.Config.Price(verifications, credits, ratelimits)
}

// BatchResolver resolves rate cards for many identities in ONE workspace and
// period while caching the reads that are constant across the batch: the
// workspace billing settings (hence the default card id) are read once, and
// each rate card is fetched and parsed once and reused across every identity
// that resolves to it. It resolves from an already-loaded identity row, so it
// never re-reads identities the caller already holds. This turns the O(N)
// workspace-settings and rate-card reads of calling ResolveAndRecord per
// identity into O(1) settings + O(distinct cards) reads.
//
// It assumes a single writer for the period's billing_period_rate_cards rows —
// the period-close cron is single-flight per period (Restate serializes the VO
// key) and is the only caller that records — so after recording a card it
// trusts the insert instead of re-reading to detect a concurrent writer. Not
// safe for concurrent use; do not share one BatchResolver across workspaces.
type BatchResolver struct {
	r           *Resolver
	workspaceID string

	settingsLoaded bool
	defaultCardID  sql.NullString

	// cards memoizes raw card loads (fetch + parse, no archived rejection) by
	// rate card id; the archived/selectable checks are applied per call site.
	// Only SUCCESSFUL loads are cached: caching an error would make one
	// transient DB hiccup on a shared card fail every identity that resolves to
	// it for the whole batch, instead of letting each retry.
	cards map[string]ResolvedRateCard
}

// NewBatch returns a BatchResolver scoped to one workspace. Reuse it across all
// identities in that workspace and period, then discard it.
func (r *Resolver) NewBatch(workspaceID string) *BatchResolver {
	return &BatchResolver{
		r:              r,
		workspaceID:    workspaceID,
		settingsLoaded: false,
		defaultCardID:  sql.NullString{},
		cards:          make(map[string]ResolvedRateCard),
	}
}

// rawCard fetches and parses a card once, caching successful loads so repeated
// resolutions to the same card cost one read per batch. Errors are not cached
// (see the cards field) — a failed load re-attempts on the next identity.
func (b *BatchResolver) rawCard(ctx context.Context, rateCardID string) (ResolvedRateCard, error) {
	if hit, ok := b.cards[rateCardID]; ok {
		return hit, nil
	}
	card, err := b.r.loadCard(ctx, b.workspaceID, rateCardID, false)
	if err != nil {
		return ResolvedRateCard{}, err
	}
	b.cards[rateCardID] = card
	return card, nil
}

// cardLoader returns a loader that applies the archived check on top of the
// per-batch card cache.
func (b *BatchResolver) cardLoader(ctx context.Context) cardLoader {
	return func(rateCardID string, rejectArchived bool) (ResolvedRateCard, error) {
		card, err := b.rawCard(ctx, rateCardID)
		if err != nil {
			return ResolvedRateCard{}, err
		}
		if rejectArchived && card.Card.Archived {
			return ResolvedRateCard{}, fault.New("rate card is archived")
		}
		return card, nil
	}
}

// resolveDefaultCardID reads the workspace billing settings once per batch.
func (b *BatchResolver) resolveDefaultCardID(ctx context.Context) (sql.NullString, error) {
	if !b.settingsLoaded {
		settings, err := db.Query.FindWorkspaceBillingSettings(ctx, b.r.database.RO(), b.workspaceID)
		if err != nil && !db.IsNotFound(err) {
			return sql.NullString{}, fault.Wrap(err, fault.Internal("failed to load workspace billing settings"))
		}
		if err == nil {
			b.defaultCardID = settings.DefaultRateCardID
		}
		b.settingsLoaded = true
	}
	return b.defaultCardID, nil
}

// ResolveAndRecord resolves the card for the already-loaded identity and
// persists it against the period (first write wins, R18), returning the card
// in force. Equivalent to Resolver.ResolveAndRecord but sharing this batch's
// caches; see BatchResolver for the single-writer assumption.
func (b *BatchResolver) ResolveAndRecord(ctx context.Context, identity db.Identity, year, month int) (ResolvedRateCard, error) {
	recorded, err := db.Query.FindBillingPeriodRateCard(ctx, b.r.database.RO(), db.FindBillingPeriodRateCardParams{
		WorkspaceID: b.workspaceID,
		IdentityID:  identity.ID,
		Year:        int32(year),
		Month:       int32(month),
	})
	if err == nil {
		// Recorded rows keep an archived card (rejectArchived=false): the
		// period was pinned to it and must re-price to the same amount.
		card, cardErr := b.rawCard(ctx, recorded.RateCardID)
		if cardErr != nil {
			return ResolvedRateCard{}, cardErr
		}
		card.ResolvedFrom = ResolvedFrom(recorded.ResolvedFrom)
		card.Recorded = true
		card.AlreadyPushed = recorded.PushedAt.Valid
		return card, nil
	}
	if !db.IsNotFound(err) {
		return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("failed to look up period rate card record"))
	}

	resolved, err := resolveLiveForIdentity(identity, b.cardLoader(ctx), func() (sql.NullString, error) {
		return b.resolveDefaultCardID(ctx)
	})
	if err != nil {
		return ResolvedRateCard{}, err
	}

	err = db.Query.InsertBillingPeriodRateCard(ctx, b.r.database.RW(), db.InsertBillingPeriodRateCardParams{
		ID:           uid.New(uid.BillingPeriodRateCardPrefix),
		WorkspaceID:  b.workspaceID,
		IdentityID:   identity.ID,
		Year:         int32(year),
		Month:        int32(month),
		RateCardID:   resolved.Card.ID,
		ResolvedFrom: db.BillingPeriodRateCardsResolvedFrom(resolved.ResolvedFrom),
		CreatedAt:    time.Now().UnixMilli(),
	})
	if err != nil {
		return ResolvedRateCard{}, fault.Wrap(err, fault.Internal("failed to record period rate card"))
	}
	// Single writer (see type doc): the row we just INSERT IGNOREd is ours, so
	// skip the re-read the shared ResolveAndRecord does to detect a concurrent
	// recorder. It is newly recorded and therefore not yet pushed.
	resolved.Recorded = true
	resolved.AlreadyPushed = false
	return resolved, nil
}

// MarkPushed stamps the identity+period as pushed; see Resolver.MarkPushed.
func (b *BatchResolver) MarkPushed(ctx context.Context, identityID string, year, month int) error {
	return b.r.MarkPushed(ctx, b.workspaceID, identityID, year, month)
}
