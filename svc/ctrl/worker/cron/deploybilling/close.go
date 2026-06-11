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

// HandleClose runs the month-end close for the CLOSED period in the VO key
// ("YYYY-MM"): a final full-period usage push (stamped just inside the period
// so the "last"-formula meters bill the full month), then finalization of each
// Deploy workspace's draft renewal invoice. Invoked by the invoice.created
// webhook at the roll and by the 00:30 UTC backup cron; both are safe to
// re-run (the push converges, finalization only touches ended drafts).
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
	_, workspacesPushed, metersPushed, failedWorkspaceIDs, err := h.pushUsage(ctx, period, p, p.End().UnixMilli(), closeTimestamp)
	if err != nil {
		return nil, err
	}

	// A workspace whose final push failed is left unfinalized: finalizing now
	// would freeze an under-billed total onto the invoice. Its draft stays open
	// for the backup cron to re-push and close; the withheld heartbeat alerts.
	failedPush := make(map[string]struct{}, len(failedWorkspaceIDs))
	for _, id := range failedWorkspaceIDs {
		failedPush[id] = struct{}{}
	}
	if len(failedPush) > 0 {
		logger.Error("final usage push failed for some workspaces; leaving their invoices in draft for the backup cron to retry",
			"billing_period", period,
			"workspaces_failed", len(failedPush),
		)
	}

	// Finalize only active Deploy workspaces' subscription_cycle drafts whose
	// period has ended; proration and next-period drafts are skipped.
	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListDeployBillableWorkspacesRow, error) {
		return db.Query.ListDeployBillableWorkspaces(rc, h.db.RO())
	}, restate.WithName("list deploy billable workspaces"))
	if err != nil {
		return nil, fmt.Errorf("list deploy billable workspaces: %w", err)
	}
	// Stable order so the journaled per-workspace steps replay identically.
	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].ID < workspaces[j].ID })

	finalized, skipped, deferred := 0, 0, 0
	for start := 0; start < len(workspaces); start += pushConcurrency {
		end := min(start+pushConcurrency, len(workspaces))
		batch := workspaces[start:end]

		// One journaled step per workspace: list and finalize its closed-period
		// drafts. Safe to replay: listing is read-only and a finalize race
		// counts as done.
		futures := make([]restate.RunAsyncFuture[closeResult], len(batch))
		for i, ws := range batch {
			futures[i] = restate.RunAsync(ctx, func(rc restate.RunContext) (closeResult, error) {
				return h.closeWorkspace(rc, p, ws, failedPush)
			}, restate.WithName("close "+ws.ID))
		}

		for i, fut := range futures {
			result, closeErr := fut.Result()
			if closeErr != nil {
				return nil, fmt.Errorf("close workspace %s: %w", batch[i].ID, closeErr)
			}
			finalized += result.Finalized
			skipped += result.Skipped
			deferred += result.Deferred
			for _, invoiceID := range result.FinalizedInvoiceIDs {
				logger.Info("finalized deploy invoice",
					"billing_period", period,
					"workspace_id", batch[i].ID,
					"invoice_id", invoiceID,
				)
			}
		}
	}

	// Withhold the heartbeat when any workspace's push failed (and so was left
	// unfinalized) so month-end monitoring fires; the run itself still
	// completes and the backup cron's fresh retry sweeps the open drafts.
	if len(failedPush) == 0 {
		if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, h.closeHeartbeat.Ping(rc)
		}, restate.WithName("send heartbeat")); err != nil {
			return nil, fmt.Errorf("send heartbeat: %w", err)
		}
	}

	logger.Info("deploy billing close complete",
		"billing_period", period,
		"workspaces_pushed", workspacesPushed,
		"meters_pushed", metersPushed,
		"invoices_finalized", finalized,
		"invoices_skipped", skipped,
		"workspaces_deferred", deferred,
	)

	return &hydrav1.RunDeployBillingCloseResponse{
		WorkspacesPushed:  int32(workspacesPushed),
		InvoicesFinalized: int32(finalized),
		InvoicesSkipped:   int32(skipped),
	}, nil
}

// closeResult is the journaled outcome of closing one workspace's invoices.
type closeResult struct {
	Finalized int
	Skipped   int
	// Deferred is set when the workspace's final push failed and its drafts
	// were intentionally left open for the backup cron to retry.
	Deferred            int
	FinalizedInvoiceIDs []string
}

// closeWorkspace finalizes the workspace's subscription_cycle drafts whose
// period has ended; proration and next-period drafts are skipped. A workspace
// whose push failed is left untouched for the backup cron.
func (h *Handler) closeWorkspace(
	rc restate.RunContext,
	p billingperiod.Period,
	ws db.ListDeployBillableWorkspacesRow,
	failedPush map[string]struct{},
) (closeResult, error) {
	result := closeResult{Finalized: 0, Skipped: 0, Deferred: 0, FinalizedInvoiceIDs: nil}

	if _, failed := failedPush[ws.ID]; failed {
		result.Deferred++
		return result, nil
	}

	if !ws.StripeCustomerID.Valid || ws.StripeCustomerID.String == "" {
		return result, nil
	}

	// Drafts are listed per subscription, not per customer: a customer with
	// another subscription besides Deploy must not have its unrelated renewal
	// drafts finalized early by this close. Every Deploy subscribe path writes
	// stripe_subscription_id, so a missing one is a data bug, not a normal
	// state; skip it loudly and leave its drafts alone.
	if !ws.StripeSubscriptionID.Valid || ws.StripeSubscriptionID.String == "" {
		logger.Error("deploy workspace has no stripe subscription id; skipping invoice close",
			"workspace_id", ws.ID,
			"stripe_customer_id", ws.StripeCustomerID.String,
		)
		result.Skipped++
		return result, nil
	}

	drafts, err := h.closer.ListDraftInvoices(rc, ws.StripeSubscriptionID.String)
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
