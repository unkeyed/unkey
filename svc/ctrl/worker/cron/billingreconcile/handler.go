package billingreconcile

import (
	"context"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	reconcile "github.com/unkeyed/unkey/svc/ctrl/internal/billingreconcile"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Engine is the reconcile comparison engine seam. The production implementation
// is *reconcile.Reconciler (svc/ctrl/internal/billingreconcile), built over the
// Stripe invoice/price readers and the ClickHouse usage reader. It takes a plain
// context.Context because a reconcile is read-only and idempotent; the cron
// wraps each call in restate.Run for replay-safe journaling. Faked in tests so
// the orchestration (enumerate -> reconcile -> classify -> emit) is unit-tested
// without a real Stripe or ClickHouse.
type Engine interface {
	ReconcileWorkspace(ctx context.Context, ws reconcile.WorkspaceRef, p billingperiod.Period) (reconcile.Result, error)
}

// billableLister is the DB seam: the single enumeration of billable workspaces.
// db.Database satisfies it. Kept narrow so tests supply a fake without the whole
// database interface.
type billableLister interface {
	ListDeployBillableWorkspaces(ctx context.Context) ([]db.ListDeployBillableWorkspacesRow, error)
}

// Config holds the reconcile cron handler's dependencies.
type Config struct {
	// Engine reconciles one workspace's invoice against live usage. Optional:
	// when nil (no Stripe key or no ClickHouse configured) the handler is a
	// no-op, matching the billing push and spend check.
	Engine Engine
	// DB enumerates the billable workspaces to reconcile. Must not be nil.
	DB billableLister
	// SlackWebhookURL pages engineering on structural findings. Optional: empty
	// falls back to an error-level log (which pages via log alerting).
	SlackWebhookURL string
}

// Handler executes RunDeployBillingReconcile: it enumerates billable workspaces
// once, fans out the read-only reconcile engine per workspace, logs each
// verdict, and pages Slack on structural findings.
//
// No heartbeat: the pass runs monthly, but incident.io alerts cap at 48h, so a
// monthly heartbeat can never be monitored. A run failure surfaces as a Restate
// terminal error (visible as a failed invocation); structural findings page via
// Slack.
type Handler struct {
	engine          Engine
	db              billableLister
	slackWebhookURL string
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
	); err != nil {
		return nil, err
	}
	return &Handler{
		engine:          cfg.Engine,
		db:              cfg.DB,
		slackWebhookURL: cfg.SlackWebhookURL,
	}, nil
}

// buildRefs turns the enumerated DB rows into engine WorkspaceRefs, skipping the
// rows that carry no Stripe subscription id to reconcile against. A billable
// workspace (active plan + Stripe customer) with no compute subscription id is a
// data gap the caller logs per workspace; it is not a verdict, because the
// engine has nothing to compare. Returns the reconcilable refs and the ids of
// the skipped rows.
//
// TODO(deferred): this is the DB->Stripe direction only. The reverse
// direction -- orphan Stripe subscriptions with no matching billable DB
// workspace -- is not built here. It needs a Stripe-side subscription
// enumeration keyed on our product, cross-checked against this list.
func buildRefs(rows []db.ListDeployBillableWorkspacesRow) (refs []reconcile.WorkspaceRef, skipped []string) {
	for _, row := range rows {
		if !row.StripeDeploySubscriptionID.Valid || row.StripeDeploySubscriptionID.String == "" {
			skipped = append(skipped, row.ID)
			continue
		}
		refs = append(refs, reconcile.WorkspaceRef{
			WorkspaceID:          row.ID,
			StripeSubscriptionID: row.StripeDeploySubscriptionID.String,
		})
	}
	return refs, skipped
}

// summary aggregates the pass's per-workspace verdicts into the counts the
// monthly summary line and response report.
type summary struct {
	clean             int
	lateDataUnderbill int
	overbill          int
	structural        int
	skipped           int
}

// summarize folds a slice of engine Results into per-verdict counts. Pure so the
// classification is unit-tested independent of the Restate wiring.
func summarize(results []reconcile.Result, skipped int) summary {
	s := summary{clean: 0, lateDataUnderbill: 0, overbill: 0, structural: 0, skipped: skipped}
	for _, r := range results {
		switch r.Verdict {
		case reconcile.VerdictClean:
			s.clean++
		case reconcile.VerdictLateDataUnderbill:
			s.lateDataUnderbill++
		case reconcile.VerdictOverbill:
			s.overbill++
		case reconcile.VerdictStructural:
			s.structural++
		default:
			// An unknown verdict ranks with structural in the engine, so count it
			// there rather than dropping it from the summary.
			s.structural++
		}
	}
	return s
}

// structuralResults filters to the results that must page: a structural verdict
// is a code or catalog bug, not a money decision. Pure so the paging predicate
// is unit-tested.
func structuralResults(results []reconcile.Result) []reconcile.Result {
	var out []reconcile.Result
	for _, r := range results {
		if r.Verdict == reconcile.VerdictStructural {
			out = append(out, r)
		}
	}
	return out
}
