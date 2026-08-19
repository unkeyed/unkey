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
// a fleet sweep; the invoice.created webhook fans out to CloseDeployBillingWorkspace
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

	p, nowTime, err := h.closeAllowed(ctx, period, req)
	if err != nil {
		return nil, err
	}

	// Claim every renewal draft BEFORE the ingestion wait. Stripe auto-finalizes an
	// unclaimed draft roughly an hour after creating it, and the wait below is 24
	// hours, so anything the invoice.created webhook failed to claim used to be
	// closed by Stripe a full day before this sweep took its first look. The sweep
	// then found no drafts, deferred nothing, logged nothing, and reported success.
	if err := h.claimDrafts(ctx, period, p); err != nil {
		return nil, err
	}

	if err := waitForUsageIngestion(ctx, p, nowTime); err != nil {
		return nil, err
	}

	push, err := h.pushFinalUsage(ctx, period, p)
	if err != nil {
		return nil, err
	}

	finalize, err := h.finalizeDrafts(ctx, period, p, push)
	if err != nil {
		return nil, err
	}

	if err := h.heartbeatIfClean(ctx, push, finalize); err != nil {
		return nil, err
	}

	return h.closeSummary(period, push, finalize)
}

type closePushResult struct {
	workspacesPushed int
	pushesFailed     int
	failedWorkspaces map[string]bool
}

type closeFinalizeResult struct {
	finalized int
	skipped   int
	deferred  int
}

// closeAllowed parses the period and checks the close may run. It deliberately
// does NOT wait for usage ingestion: the caller claims drafts first, then waits,
// so Stripe cannot finalize an unclaimed invoice underneath the sleep.
func (h *Handler) closeAllowed(
	ctx restate.ObjectContext,
	period string,
	req *hydrav1.RunDeployBillingCloseRequest,
) (billingperiod.Period, time.Time, error) {
	p, err := billingperiod.Parse(period)
	if err != nil {
		return billingperiod.Period{}, time.Time{}, restate.TerminalError(fmt.Errorf("invalid billing period %q: %w", period, err))
	}

	nowTime, err := restateutil.Now(ctx)
	if err != nil {
		return billingperiod.Period{}, time.Time{}, fmt.Errorf("get current time: %w", err)
	}
	var stripePeriodEnd int64
	if req != nil {
		stripePeriodEnd = req.GetPeriodEnd()
	}
	if !p.CloseAllowed(nowTime, stripePeriodEnd) {
		return billingperiod.Period{}, time.Time{}, restate.TerminalError(
			fmt.Errorf("billing period %s has not ended yet (ends %s)", period, p.End().Format(time.RFC3339)),
		)
	}

	return p, nowTime, nil
}

// claimBackstop is how long past period end a claimed draft is allowed to sit
// before Stripe finalizes it anyway. Matches the invoice.created webhook's
// backstop so both paths agree, and leaves headroom over the close's own
// finalize at period end + 25h (24h ingest wait + 1h meter aggregation).
const claimBackstop = 48 * time.Hour

// claimDrafts extends Stripe's automatic finalization on every renewal draft of
// every billable workspace, so the ingestion wait cannot be outrun. Best-effort
// per workspace: a claim failure is logged and the workspace continues, because
// failing the whole sweep would leave the remaining workspaces unclaimed too.
func (h *Handler) claimDrafts(
	ctx restate.ObjectContext,
	period string,
	p billingperiod.Period,
) error {
	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListDeployBillableWorkspacesRow, error) {
		return h.db.ListDeployBillableWorkspaces(rc)
	}, restate.WithName("list deploy billable workspaces for claim"))
	if err != nil {
		return fmt.Errorf("list deploy billable workspaces: %w", err)
	}
	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].ID < workspaces[j].ID })

	finalizeAt := p.End().Add(claimBackstop).Unix()
	claimed := 0

	stepErrs := runBatched(ctx, workspaces,
		func(ws db.ListDeployBillableWorkspacesRow) string { return "claim " + ws.ID },
		func(rc restate.RunContext, ws db.ListDeployBillableWorkspacesRow) (int, error) {
			if !ws.StripeDeploySubscriptionID.Valid || ws.StripeDeploySubscriptionID.String == "" {
				return 0, nil
			}
			drafts, err := h.closer.ListDraftInvoices(rc, ws.StripeDeploySubscriptionID.String)
			if err != nil {
				logger.Error("list draft invoices failed during claim; leaving this workspace to Stripe's schedule",
					"workspace_id", ws.ID, "error", err)
				return 0, nil
			}
			n := 0
			for _, draft := range drafts {
				if draft.BillingReason != "subscription_cycle" {
					continue
				}
				// An old-period rerun can see the current renewal draft. Never
				// replace its backstop with a deadline derived from the old period.
				if draft.PeriodStart >= p.End().Unix() {
					continue
				}
				if err := h.closer.ClaimInvoice(rc, draft.ID, finalizeAt); err != nil {
					logger.Error("claim invoice failed; Stripe may finalize it before the close completes",
						"workspace_id", ws.ID, "invoice_id", draft.ID, "error", err)
					continue
				}
				n++
			}
			return n, nil
		},
		func(_ db.ListDeployBillableWorkspacesRow, n int) { claimed += n },
	)
	for _, stepErr := range stepErrs {
		logger.Error("claim step failed", "billing_period", period, "error", stepErr)
	}

	logger.Info("deploy billing close claimed drafts",
		"billing_period", period,
		"invoices_claimed", claimed,
		"finalize_at", finalizeAt,
	)
	return nil
}

func (h *Handler) pushFinalUsage(
	ctx restate.ObjectContext,
	period string,
	p billingperiod.Period,
) (closePushResult, error) {
	result := closePushResult{
		workspacesPushed: 0,
		pushesFailed:     0,
		failedWorkspaces: make(map[string]bool),
	}

	closeTimestamp := p.End().Add(-time.Second).Unix()
	tasks, _, err := h.resolvePushTasks(ctx, period, p, p.End().UnixMilli(), closeTimestamp)
	if err != nil {
		return result, err
	}

	type pushFuture = restate.ResponseFuture[*hydrav1.PushWorkspaceUsageResponse]
	pushFutures := make([]pushFuture, len(tasks))
	for i, task := range tasks {
		pushFutures[i] = hydrav1.NewDeployBillingPushServiceClient(ctx, task.workspaceID).
			PushWorkspaceUsage().RequestFuture(task.pushRequest())
	}

	for i, fut := range pushFutures {
		if _, perr := fut.Response(); perr != nil {
			result.pushesFailed++
			result.failedWorkspaces[tasks[i].workspaceID] = true
			logger.Error("final usage push failed; leaving this workspace's invoice open for the backup close",
				"billing_period", period,
				"workspace_id", tasks[i].workspaceID,
				"error", perr,
			)
			continue
		}
		result.workspacesPushed++
	}

	if err := h.waitForMeterAggregation(ctx, result.workspacesPushed > 0); err != nil {
		return result, err
	}

	return result, nil
}

func (h *Handler) finalizeDrafts(
	ctx restate.ObjectContext,
	period string,
	p billingperiod.Period,
	push closePushResult,
) (closeFinalizeResult, error) {
	result := closeFinalizeResult{
		finalized: 0,
		skipped:   0,
		deferred:  0,
	}

	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListDeployBillableWorkspacesRow, error) {
		return h.db.ListDeployBillableWorkspaces(rc)
	}, restate.WithName("list deploy billable workspaces"))
	if err != nil {
		return result, fmt.Errorf("list deploy billable workspaces: %w", err)
	}
	sort.Slice(workspaces, func(i, j int) bool { return workspaces[i].ID < workspaces[j].ID })

	if len(push.failedWorkspaces) > 0 {
		kept := make([]db.ListDeployBillableWorkspacesRow, 0, len(workspaces))
		for _, ws := range workspaces {
			if push.failedWorkspaces[ws.ID] {
				result.deferred++
				continue
			}
			kept = append(kept, ws)
		}
		workspaces = kept
	}

	stepErrs := runBatched(ctx, workspaces,
		func(ws db.ListDeployBillableWorkspacesRow) string { return "close " + ws.ID },
		func(rc restate.RunContext, ws db.ListDeployBillableWorkspacesRow) (closeResult, error) {
			return h.closeWorkspace(rc, p, ws)
		},
		func(ws db.ListDeployBillableWorkspacesRow, workspaceResult closeResult) {
			result.finalized += workspaceResult.Finalized
			result.skipped += workspaceResult.Skipped
			result.deferred += workspaceResult.Deferred
			for _, invoiceID := range workspaceResult.FinalizedInvoiceIDs {
				logger.Info("finalized deploy invoice",
					"billing_period", period,
					"workspace_id", ws.ID,
					"invoice_id", invoiceID,
				)
			}
		},
	)

	// Each failed step defers just its own workspace; the rest still finalize.
	for _, stepErr := range stepErrs {
		logger.Error("deploy billing close step failed; deferring workspace", "error", stepErr)
		result.deferred++
	}

	return result, nil
}

func (h *Handler) heartbeatIfClean(
	ctx restate.ObjectContext,
	push closePushResult,
	finalize closeFinalizeResult,
) error {
	if push.pushesFailed > 0 || finalize.deferred > 0 {
		return nil
	}
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.closeHeartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	return nil
}

func (h *Handler) closeSummary(
	period string,
	push closePushResult,
	finalize closeFinalizeResult,
) (*hydrav1.RunDeployBillingCloseResponse, error) {
	logger.Info("deploy billing close complete",
		"billing_period", period,
		"workspaces_pushed", push.workspacesPushed,
		"workspaces_push_failed", push.pushesFailed,
		"invoices_finalized", finalize.finalized,
		"invoices_skipped", finalize.skipped,
		"invoices_deferred", finalize.deferred,
	)

	if push.pushesFailed > 0 {
		return nil, restate.TerminalError(
			fmt.Errorf("deploy billing close for %s deferred %d workspace(s) after %d failed push(es)", period, finalize.deferred, push.pushesFailed),
		)
	}

	return &hydrav1.RunDeployBillingCloseResponse{
		WorkspacesPushed:  int32(push.workspacesPushed),
		InvoicesFinalized: int32(finalize.finalized),
		InvoicesSkipped:   int32(finalize.skipped),
	}, nil
}

type closeResult struct {
	Finalized           int
	Skipped             int
	Deferred            int
	FinalizedInvoiceIDs []string
}

func (h *Handler) closeWorkspace(
	rc restate.RunContext,
	p billingperiod.Period,
	ws db.ListDeployBillableWorkspacesRow,
) (closeResult, error) {
	result := closeResult{Finalized: 0, Skipped: 0, Deferred: 0, FinalizedInvoiceIDs: nil}

	if !ws.StripeCustomerID.Valid || ws.StripeCustomerID.String == "" {
		return result, nil
	}

	if !ws.StripeDeploySubscriptionID.Valid || ws.StripeDeploySubscriptionID.String == "" {
		logger.Error("deploy workspace has no stripe subscription id; deferring close",
			"workspace_id", ws.ID,
			"stripe_customer_id", ws.StripeCustomerID.String,
		)
		result.Deferred++
		return result, nil
	}

	drafts, err := h.closer.ListDraftInvoices(rc, ws.StripeDeploySubscriptionID.String)
	if err != nil {
		logger.Error("list draft invoices failed; deferring close",
			"workspace_id", ws.ID,
			"stripe_subscription_id", ws.StripeDeploySubscriptionID.String,
			"error", err,
		)
		result.Deferred++
		return result, nil
	}

	for _, draft := range drafts {
		// Not a renewal at all: proration and manual invoices are Stripe's to
		// finalize, and skipping them is not a billing problem.
		if draft.BillingReason != "subscription_cycle" {
			result.Skipped++
			continue
		}

		// An old-period rerun may discover the next renewal draft. Its start
		// boundary identifies it without hiding an anchor drift in the target
		// period, whose start remains before p.End().
		if draft.PeriodStart >= p.End().Unix() {
			result.Skipped++
			continue
		}

		// Period end must land on the calendar-month boundary (a drifted anchor
		// bills the wrong window); a later start is fine (a mid-month subscribe's
		// first cycle is [subscribe_date, 1st]). Defer rather than skip: a deferral
		// withholds the heartbeat and leaves the invoice unbilled until the anchor is
		// fixed, whereas a skip is silent.
		//
		// Do not replace this with a PeriodEnd-based skip. A subscription anchored
		// to day-of-month 1 but not to midnight (created before the anchor config
		// pinned hour/minute/second) has PeriodEnd LATER than p.End(), so that skip
		// would swallow exactly the drift this assertion exists to catch.
		if draft.PeriodEnd != p.End().Unix() || draft.PeriodStart < p.Start().Unix() {
			logger.Error("deploy invoice period does not align to the calendar month; refusing to finalize",
				"workspace_id", ws.ID,
				"invoice_id", draft.ID,
				"invoice_period_start", draft.PeriodStart,
				"invoice_period_end", draft.PeriodEnd,
				"expected_period_start", p.Start().Unix(),
				"expected_period_end", p.End().Unix(),
			)
			result.Deferred++
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
