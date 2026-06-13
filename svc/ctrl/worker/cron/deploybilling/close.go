package deploybilling

import (
	"fmt"
	"sort"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
)

// HandleClose performs the month-end close for the CLOSED billing period in
// the VO key ("YYYY-MM"): one final usage push covering the full period,
// timestamped just inside it so the "last"-formula meters bill the final
// total, then finalization of every Deploy workspace's draft renewal invoice.
//
// Both entry points are safe to repeat: the Stripe invoice.created webhook
// (via ctrl-api) invokes it at the period roll, and the 01:05 UTC backup cron
// invokes it again as a fallback. The push converges by construction (same
// absolute value, same in-period timestamp) and finalization only touches
// drafts, so a period where the webhook already ran closes as a no-op sweep.
func (h *Handler) HandleClose(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeployBillingCloseRequest,
) (*hydrav1.RunDeployBillingCloseResponse, error) {
	period := restate.Key(ctx)
	logger.Info("running deploy billing close", "billing_period", period)

	if h.usage == nil {
		logger.Info("deploy billing close disabled (no usage reader configured)")
		return &hydrav1.RunDeployBillingCloseResponse{}, nil
	}

	p, err := billingperiod.Parse(period)
	if err != nil {
		return nil, restate.TerminalError(fmt.Errorf("invalid billing period %q: %w", period, err))
	}

	// Refuse to close a period that has not ended: the final push would
	// freeze a partial total as the month's "last" value, and the renewal
	// invoices do not exist yet anyway.
	nowTime, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	if nowTime.Before(p.End()) {
		return nil, restate.TerminalError(
			fmt.Errorf("billing period %s has not ended yet (ends %s)", period, p.End().Format(time.RFC3339)),
		)
	}

	// Final push: the full period's usage, stamped one second before the
	// period ends so the meters' aggregation window for the closing invoice
	// picks it up as the last (and therefore billed) value.
	closeTimestamp := p.End().Add(-time.Second).Unix()
	tasks, _, err := h.resolvePushTasks(ctx, period, p, p.End().UnixMilli(), closeTimestamp)
	if err != nil {
		return nil, err
	}

	// Unlike the hourly tick, the close AWAITS each push: the closing invoice
	// must see the final total before we finalize it. The pushes run as
	// independent invocations (each with its own bounded retry), so awaiting
	// them concurrently isolates failures without blocking siblings.
	type pushFuture = restate.ResponseFuture[*hydrav1.PushWorkspaceUsageResponse]
	pushFutures := make([]pushFuture, len(tasks))
	for i, task := range tasks {
		pushFutures[i] = hydrav1.NewDeployBillingPushServiceClient(ctx, task.workspaceID).
			PushWorkspaceUsage().RequestFuture(task.pushRequest())
	}
	workspacesPushed, pushesFailed := 0, 0
	for i, fut := range pushFutures {
		if _, perr := fut.Response(); perr != nil {
			// Tolerate the failure and finalize anyway: parking the invoice in
			// draft would strand it until next month's backup cron (auto_advance
			// is off by then), which is worse than finalizing with the last
			// hourly value (at most an hour stale). The error log plus the
			// withheld heartbeat below make sure someone looks.
			pushesFailed++
			logger.Error("final usage push failed; invoice will finalize with the last hourly value",
				"billing_period", period,
				"workspace_id", tasks[i].workspaceID,
				"error", perr,
			)
			continue
		}
		workspacesPushed++
	}

	// Finalize the draft renewal invoices. Scope: only workspaces with an
	// active Deploy plan, and only subscription_cycle drafts whose period has
	// ended — subscribe/upgrade proration invoices and the next period's
	// drafts are never touched.
	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListDeployBillableWorkspacesRow, error) {
		return db.Query.ListDeployBillableWorkspaces(rc, h.db.RO())
	}, restate.WithName("list deploy billable workspaces"))
	if err != nil {
		return nil, fmt.Errorf("list deploy billable workspaces: %w", err)
	}
	// Stable order so the journaled per-workspace steps replay identically.
	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].ID < workspaces[j].ID })

	finalized, skipped := 0, 0
	for start := 0; start < len(workspaces); start += finalizeConcurrency {
		end := min(start+finalizeConcurrency, len(workspaces))
		batch := workspaces[start:end]

		// One journaled step per workspace: list drafts and finalize the
		// closed period's renewal invoices in a single closure, fanned out in
		// bounded batches like the usage push. Re-running the combined step is
		// safe: listing is read-only and a finalize race counts as done, so a
		// partial failure simply re-sweeps that workspace's drafts.
		futures := make([]restate.RunAsyncFuture[closeResult], len(batch))
		for i, ws := range batch {
			futures[i] = restate.RunAsync(ctx, func(rc restate.RunContext) (closeResult, error) {
				return h.closeWorkspace(rc, p, ws)
			}, restate.WithName("close "+ws.ID))
		}

		for i, fut := range futures {
			result, closeErr := fut.Result()
			if closeErr != nil {
				return nil, fmt.Errorf("close workspace %s: %w", batch[i].ID, closeErr)
			}
			finalized += result.Finalized
			skipped += result.Skipped
			for _, invoiceID := range result.FinalizedInvoiceIDs {
				logger.Info("finalized deploy invoice",
					"billing_period", period,
					"workspace_id", batch[i].ID,
					"invoice_id", invoiceID,
				)
			}
		}
	}

	// Withhold the heartbeat on partial failure so month-end monitoring
	// fires; the run itself still completes (re-runs converge).
	if pushesFailed == 0 {
		if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, h.closeHeartbeat.Ping(rc)
		}, restate.WithName("send heartbeat")); err != nil {
			return nil, fmt.Errorf("send heartbeat: %w", err)
		}
	}

	logger.Info("deploy billing close complete",
		"billing_period", period,
		"workspaces_pushed", workspacesPushed,
		"invoices_finalized", finalized,
		"invoices_skipped", skipped,
	)

	return &hydrav1.RunDeployBillingCloseResponse{
		WorkspacesPushed:  int32(workspacesPushed),
		InvoicesFinalized: int32(finalized),
		InvoicesSkipped:   int32(skipped),
	}, nil
}

// finalizeConcurrency bounds how many workspaces we finalize invoices for at
// once, so the close does not hammer the provider's rate limits.
const finalizeConcurrency = 16

// closeResult is the journaled outcome of closing one workspace's invoices.
type closeResult struct {
	Finalized           int
	Skipped             int
	FinalizedInvoiceIDs []string
}

// closeWorkspace lists the workspace's draft invoices and finalizes the ones
// belonging to closed periods: subscription_cycle drafts whose period has
// ended. Subscribe/upgrade proration invoices and the next period's drafts
// are skipped.
func (h *Handler) closeWorkspace(
	rc restate.RunContext,
	p billingperiod.Period,
	ws db.ListDeployBillableWorkspacesRow,
) (closeResult, error) {
	result := closeResult{Finalized: 0, Skipped: 0, FinalizedInvoiceIDs: nil}
	if !ws.StripeCustomerID.Valid || ws.StripeCustomerID.String == "" {
		return result, nil
	}

	drafts, err := h.closer.ListDraftInvoices(rc, ws.StripeCustomerID.String)
	if err != nil {
		return result, fmt.Errorf("list draft invoices: %w", err)
	}

	for _, draft := range drafts {
		if draft.BillingReason != "subscription_cycle" || draft.PeriodEnd > p.End().Unix() {
			result.Skipped++
			continue
		}

		alreadyDone, err := h.closer.FinalizeInvoice(rc, draft.ID)
		if err != nil {
			return result, fmt.Errorf("finalize invoice %s: %w", draft.ID, err)
		}
		if alreadyDone {
			result.Skipped++
		} else {
			result.Finalized++
			result.FinalizedInvoiceIDs = append(result.FinalizedInvoiceIDs, draft.ID)
		}
	}
	return result, nil
}
