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

	// Ingestion lateness buffer: the webhook dispatches the close seconds
	// after the roll, but rows from the period's last minutes are still in
	// flight to ClickHouse (client batching, async-insert flush, retries).
	// Reading the final total before they land under-bills permanently: the
	// invoice is finalized and the backup cron cannot amend it. The sleep is
	// durable and period-relative, so the 00:30 backup cron and re-runs of
	// old periods pass straight through.
	if ingestSafe := p.End().Add(usageIngestLateness); nowTime.Before(ingestSafe) {
		if err := restate.Sleep(ctx, ingestSafe.Sub(nowTime)); err != nil {
			return nil, fmt.Errorf("wait for usage ingestion: %w", err)
		}
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
	// Track workspaces whose push failed: finalizing them now would bill the
	// stale hourly value and drop the period's final usage. Their drafts stay
	// open for the backup cron to re-push and close.
	pushFailedWorkspaces := make(map[string]bool)
	for i, fut := range pushFutures {
		if _, perr := fut.Response(); perr != nil {
			pushesFailed++
			pushFailedWorkspaces[tasks[i].workspaceID] = true
			logger.Error("final usage push failed; leaving this workspace's invoice open for the backup close",
				"billing_period", period,
				"workspace_id", tasks[i].workspaceID,
				"error", perr,
			)
			continue
		}
		workspacesPushed++
	}

	// Meter aggregation delay: a successful push only means Stripe accepted
	// the meter events; folding them into the draft invoice's metered lines
	// is asynchronous on Stripe's side. Finalizing immediately can freeze the
	// previous hourly total onto the invoice. The sleep is durable, and zero
	// in test wiring (no Stripe key) where there is nothing to aggregate.
	if h.finalizeDelay > 0 && workspacesPushed > 0 {
		if err := restate.Sleep(ctx, h.finalizeDelay); err != nil {
			return nil, fmt.Errorf("wait for meter aggregation: %w", err)
		}
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

	// Drop workspaces whose final push failed: finalizing them now would bill
	// the stale value. Their drafts stay open for the backup close. Filtering
	// before the batched fan-out keeps the journaled step order stable.
	deferred := 0
	if len(pushFailedWorkspaces) > 0 {
		kept := make([]db.ListDeployBillableWorkspacesRow, 0, len(workspaces))
		for _, ws := range workspaces {
			if pushFailedWorkspaces[ws.ID] {
				deferred++
				continue
			}
			kept = append(kept, ws)
		}
		workspaces = kept
	}

	finalized, skipped := 0, 0
	for start := 0; start < len(workspaces); start += finalizeConcurrency {
		end := min(start+finalizeConcurrency, len(workspaces))
		batch := workspaces[start:end]

		// One journaled step per workspace: list and finalize its closed-period
		// drafts. Safe to replay: listing is read-only and a finalize race
		// counts as done.
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

	// Withhold the heartbeat when any workspace's push failed (and so was left
	// unfinalized) so month-end monitoring fires; the run itself still
	// completes and the backup cron's fresh retry sweeps the open drafts.
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
		"workspaces_push_failed", pushesFailed,
		"invoices_finalized", finalized,
		"invoices_skipped", skipped,
		"invoices_deferred", deferred,
	)

	// Some final pushes failed, so we finalized the healthy workspaces and left
	// the rest open for the backup close. Surface the run as failed (terminal,
	// so this keyed invocation does not retry in place and replay the journaled
	// push failure) on top of the withheld heartbeat above, so the failure is
	// visible and the fresh-key backup cron picks up the deferred invoices.
	if pushesFailed > 0 {
		return nil, restate.TerminalError(
			fmt.Errorf("deploy billing close for %s deferred %d workspace(s) after %d failed push(es)", period, deferred, pushesFailed),
		)
	}

	return &hydrav1.RunDeployBillingCloseResponse{
		WorkspacesPushed:  int32(workspacesPushed),
		InvoicesFinalized: int32(finalized),
		InvoicesSkipped:   int32(skipped),
	}, nil
}

// finalizeConcurrency bounds how many workspaces we finalize invoices for at
// once, so the close does not hammer the provider's rate limits.
const finalizeConcurrency = 16

// usageIngestLateness is how long after the period ends the close waits
// before taking the final ClickHouse read. Bounds the age of rows the close
// can still pick up; anything arriving later than this after month end is
// accepted as lost to the closed invoice.
const usageIngestLateness = 15 * time.Minute

// DefaultFinalizeDelay is the production wait between the final meter push
// and invoice finalization, giving Stripe time to aggregate the events into
// the draft's metered lines. Stripe documents aggregation as asynchronous
// without a hard bound; its own auto-finalize waits about an hour after
// invoice creation, so twenty minutes on top of an explicit final push is
// conservative without dragging the close far past the roll.
const DefaultFinalizeDelay = 20 * time.Minute

// closeResult is the journaled outcome of closing one workspace's invoices.
type closeResult struct {
	Finalized           int
	Skipped             int
	FinalizedInvoiceIDs []string
}

// closeWorkspace finalizes the workspace's subscription_cycle drafts whose
// period has ended; proration and next-period drafts are skipped. Failed-push
// workspaces are filtered out by the caller, so they never reach here.
func (h *Handler) closeWorkspace(
	rc restate.RunContext,
	p billingperiod.Period,
	ws db.ListDeployBillableWorkspacesRow,
) (closeResult, error) {
	result := closeResult{Finalized: 0, Skipped: 0, FinalizedInvoiceIDs: nil}

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
