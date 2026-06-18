package deployspendcheck

import (
	"fmt"
	"sort"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
)

// Config holds the orchestrator's dependencies. The per-workspace check (price,
// overage, alert) runs in CheckHandler; this handler only resolves who to check
// and fans out to it.
type Config struct {
	// DB is the primary application database, used to list workspaces with a
	// configured Deploy spend budget. Must not be nil.
	DB db.Database

	// Heartbeat is pinged when the orchestration completes. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunDeploySpendCheck: it lists budgeted workspaces and fans
// out to DeploySpendCheckService.
type Handler struct {
	db        db.Database
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, heartbeat: cfg.Heartbeat}, nil
}

// Handle lists the workspaces that configured a Deploy spend budget (the VO key
// is the billing period "YYYY-MM") and fans out one check per workspace,
// fire-and-forget. A workspace whose included credit is not yet known is
// skipped: without it the overage can't be priced without counting the full
// gross, which would false-alarm. Each check runs, retries, and fails on its
// own, so a broken workspace cannot stall the others or this tick.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeploySpendCheckRequest,
) (*hydrav1.RunDeploySpendCheckResponse, error) {
	period := restate.Key(ctx)
	logger.Info("running deploy spend check", "billing_period", period)

	budgeted, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesWithDeployBudgetRow, error) {
		return db.Query.ListWorkspacesWithDeployBudget(rc, h.db.RO())
	}, restate.WithName("list budgeted workspaces"))
	if err != nil {
		return nil, fmt.Errorf("list budgeted workspaces: %w", err)
	}

	// Sort by id so the fan-out order is stable across replays: each Send is
	// journaled by position, so a different iteration order on replay would
	// dispatch a different request at the same journal index and diverge.
	sort.Slice(budgeted, func(i, j int) bool { return budgeted[i].ID < budgeted[j].ID })

	var dispatched, skippedNoCredit int32
	for _, ws := range budgeted {
		if !ws.DeploySpendBudgetCents.Valid {
			continue // query filters these out; guard against a future query change
		}

		if !ws.DeployIncludedCreditCents.Valid {
			skippedNoCredit++
			// Error, not info: an unknown credit disables both budget alerts
			// and the spend cap for this workspace for as long as it persists.
			// The dashboard webhook re-persists it on the next invoice event,
			// so a workspace stuck here means that recovery path is broken.
			logger.Error("skip spend check: included credit unknown; alerts and spend cap disabled for this workspace",
				"workspace_id", ws.ID,
				"billing_period", period,
			)
			continue
		}

		// Fire-and-forget: Send is journaled, so a replay does not re-dispatch.
		// The child invocation owns retries, state, and the email send.
		hydrav1.NewDeploySpendCheckServiceClient(ctx, ws.ID).
			CheckWorkspaceSpend().
			Send(&hydrav1.CheckWorkspaceSpendRequest{
				Period:              period,
				BudgetCents:         ws.DeploySpendBudgetCents.Int64,
				IncludedCreditCents: ws.DeployIncludedCreditCents.Int64,
				Stop:                ws.DeploySpendBudgetStop,
				OrgId:               ws.OrgID,
				WorkspaceName:       ws.Name,
				WorkspaceSlug:       ws.Slug,
			})
		dispatched++
	}

	logger.Info("deploy spend check dispatched",
		"billing_period", period,
		"workspaces_dispatched", dispatched,
		"workspaces_skipped_no_credit", skippedNoCredit,
	)

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunDeploySpendCheckResponse{
		WorkspacesDispatched:      dispatched,
		WorkspacesSkippedNoCredit: skippedNoCredit,
	}, nil
}
