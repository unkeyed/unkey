package deploybilling

import (
	"fmt"
	"sort"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// HandleClose bills the closed month in the VO key ("YYYY-MM"): final usage
// push, then finalize renewal drafts. The 00:30 UTC backup cron runs this as
// a fleet sweep; the invoice.created webhook fans out to HandleCloseWorkspace
// per invoice instead. Safe to re-run.
func (h *Handler) HandleClose(
	ctx restate.ObjectContext,
	req *hydrav1.RunDeployBillingCloseRequest,
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

	// Period still open on wall clock and Stripe has not rolled it yet.
	nowTime, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	var stripePeriodEnd int64
	if req != nil {
		stripePeriodEnd = req.GetPeriodEnd()
	}
	if !p.CloseAllowed(nowTime, stripePeriodEnd) {
		return nil, restate.TerminalError(
			fmt.Errorf("billing period %s has not ended yet (ends %s)", period, p.End().Format(time.RFC3339)),
		)
	}

	// Stamp one second before period end so "last" meters bill the full month.
	closeTimestamp := p.End().Add(-time.Second).Unix()
	_, workspacesPushed, metersPushed, failedWorkspaceIDs, err := h.pushUsage(ctx, period, p, p.End().UnixMilli(), closeTimestamp)
	if err != nil {
		return nil, err
	}

	// Push failed: leave the draft open. Finalizing now freezes an undercount.
	// Backup cron retries; we skip the heartbeat so monitoring fires.
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

	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListDeployBillableWorkspacesRow, error) {
		return h.db.ListDeployBillableWorkspaces(rc)
	}, restate.WithName("list deploy billable workspaces"))
	if err != nil {
		return nil, fmt.Errorf("list deploy billable workspaces: %w", err)
	}
	// Stable order for deterministic Restate replay.
	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].ID < workspaces[j].ID })

	finalized, skipped, deferred := 0, 0, 0
	for start := 0; start < len(workspaces); start += pushConcurrency {
		end := min(start+pushConcurrency, len(workspaces))
		batch := workspaces[start:end]

		// One journaled step per workspace. List is read-only; finalize races count as done.
		futures := make([]restate.RunAsyncFuture[closeResult], len(batch))
		for i, ws := range batch {
			futures[i] = restate.RunAsync(ctx, func(rc restate.RunContext) (closeResult, error) {
				return h.closeWorkspace(rc, p, ws, failedPush)
			}, restate.WithName("close "+ws.ID))
		}

		for i, fut := range futures {
			result, closeErr := fut.Result()
			if closeErr != nil {
				// RunAsync step failed unexpectedly; defer this workspace.
				logger.Error("deploy billing close step failed",
					"workspace_id", batch[i].ID,
					"error", closeErr,
				)
				deferred++
				continue
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

	if deferred == 0 {
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

type closeResult struct {
	Finalized int
	Skipped   int
	// Push failed; draft left open for backup cron.
	Deferred            int
	FinalizedInvoiceIDs []string
}

// closeWorkspace finalizes ended subscription_cycle drafts; skips proration and next period.
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

	// List by subscription so we do not finalize another product's renewal draft.
	if !ws.StripeSubscriptionID.Valid || ws.StripeSubscriptionID.String == "" {
		logger.Error("deploy workspace has no stripe subscription id; deferring close",
			"workspace_id", ws.ID,
			"stripe_customer_id", ws.StripeCustomerID.String,
		)
		result.Deferred++
		return result, nil
	}

	drafts, err := h.closer.ListDraftInvoices(rc, ws.StripeSubscriptionID.String)
	if err != nil {
		logger.Error("list draft invoices failed; deferring close",
			"workspace_id", ws.ID,
			"stripe_subscription_id", ws.StripeSubscriptionID.String,
			"error", err,
		)
		result.Deferred++
		return result, nil
	}

	for _, draft := range drafts {
		if draft.BillingReason != "subscription_cycle" || draft.PeriodEnd > p.End().Unix() {
			result.Skipped++
			continue
		}

		alreadyDone, err := h.closer.FinalizeInvoice(rc, draft.ID)
		if err != nil {
			logger.Error("finalize invoice failed; deferring close",
				"workspace_id", ws.ID,
				"invoice_id", draft.ID,
				"error", err,
			)
			result.Deferred++
			return result, nil
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
