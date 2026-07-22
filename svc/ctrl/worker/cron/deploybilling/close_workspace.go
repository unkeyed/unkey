package deploybilling

import (
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// HandleCloseWorkspace closes one workspace's Deploy renewal invoice. The
// invoice.created webhook dispatches this per draft; the 00:30 UTC cron runs
// the fleet sweep (HandleClose) instead. Shares the fleet close's ingestion
// buffer and meter-aggregation wait before finalizing.
func (h *Handler) HandleCloseWorkspace(
	ctx restate.ObjectContext,
	req *hydrav1.CloseDeployBillingWorkspaceRequest,
) (*hydrav1.CloseDeployBillingWorkspaceResponse, error) {
	workspaceID := restate.Key(ctx)
	period := req.GetPeriod()
	logger.Info("running deploy billing close for workspace",
		"workspace_id", workspaceID,
		"billing_period", period,
		"invoice_id", req.GetInvoiceId(),
	)

	if h.usage == nil {
		logger.Info("deploy billing close disabled (no usage reader configured)")
		return &hydrav1.CloseDeployBillingWorkspaceResponse{}, nil
	}

	if req.GetInvoiceId() == "" {
		return nil, restate.TerminalError(fmt.Errorf("invoice_id is required"))
	}

	p, err := billingperiod.Parse(period)
	if err != nil {
		return nil, restate.TerminalError(fmt.Errorf("invalid billing period %q: %w", period, err))
	}

	nowTime, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	if !p.CloseAllowed(nowTime, req.GetPeriodEnd()) {
		return nil, restate.TerminalError(
			fmt.Errorf("billing period %s has not ended yet (ends %s)", period, p.End().Format(time.RFC3339)),
		)
	}

	if err := waitForUsageIngestion(ctx, p, nowTime); err != nil {
		return nil, err
	}

	closeTimestamp := p.End().Add(-time.Second).Unix()
	push, err := h.pushSingleWorkspace(ctx, period, p, workspaceID, p.End().UnixMilli(), closeTimestamp)
	if err != nil {
		return nil, err
	}
	if push.failed {
		logger.Error("final usage push failed; leaving invoice in draft for the backup cron",
			"workspace_id", workspaceID,
			"billing_period", period,
			"invoice_id", req.GetInvoiceId(),
		)
		return &hydrav1.CloseDeployBillingWorkspaceResponse{}, nil
	}

	if err := h.waitForMeterAggregation(ctx, push.pushed); err != nil {
		return nil, err
	}

	alreadyDone, err := restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
		return h.closer.FinalizeInvoice(rc, req.GetInvoiceId())
	}, restate.WithName("finalize invoice"))
	if err != nil {
		logger.Error("finalize invoice failed; leaving draft for backup cron",
			"workspace_id", workspaceID,
			"invoice_id", req.GetInvoiceId(),
			"error", err,
		)
		return &hydrav1.CloseDeployBillingWorkspaceResponse{}, nil
	}
	if alreadyDone {
		logger.Info("deploy renewal invoice already finalized",
			"workspace_id", workspaceID,
			"invoice_id", req.GetInvoiceId(),
		)
	} else {
		logger.Info("finalized deploy invoice",
			"workspace_id", workspaceID,
			"billing_period", period,
			"invoice_id", req.GetInvoiceId(),
		)
	}

	return &hydrav1.CloseDeployBillingWorkspaceResponse{}, nil
}

type workspacePushOutcome struct {
	pushed bool
	failed bool
}

// pushSingleWorkspace pushes the closed period's final usage for one workspace.
func (h *Handler) pushSingleWorkspace(
	ctx restate.ObjectContext,
	period string,
	p billingperiod.Period,
	workspaceID string,
	endMillis int64,
	eventTimestamp int64,
) (workspacePushOutcome, error) {
	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return h.usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID:  workspaceID,
			WorkspaceIDs: nil,
			Start:        p.Start().UnixMilli(),
			End:          endMillis,
		})
	}, restate.WithName("get workspace period usage"))
	if err != nil {
		return workspacePushOutcome{}, fmt.Errorf("get workspace period usage: %w", err)
	}

	valuesByWorkspace := AggregateUsage(rows)
	values, ok := valuesByWorkspace[workspaceID]
	if !ok || !values.Positive() {
		logger.Info("no deploy usage to push for workspace close",
			"workspace_id", workspaceID,
			"billing_period", period,
		)
		return workspacePushOutcome{pushed: false, failed: false}, nil
	}

	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesForDeployBillingByIDsRow, error) {
		return h.db.ListWorkspacesForDeployBillingByIDs(rc, []string{workspaceID})
	}, restate.WithName("fetch workspace billing identity"))
	if err != nil {
		return workspacePushOutcome{}, fmt.Errorf("fetch workspace billing identity: %w", err)
	}
	if len(workspaces) == 0 {
		logger.Info("workspace not billable; skipping final push",
			"workspace_id", workspaceID,
			"billing_period", period,
		)
		return workspacePushOutcome{pushed: false, failed: false}, nil
	}
	ws := workspaces[0]
	if !ws.StripeCustomerID.Valid || ws.StripeCustomerID.String == "" {
		logger.Info("workspace has usage but no stripe customer; skipping final push",
			"workspace_id", workspaceID,
			"billing_period", period,
		)
		return workspacePushOutcome{pushed: false, failed: false}, nil
	}

	// Awaited per-workspace push invocation, same as the fleet close: the
	// invocation carries its own bounded retry, and an error here means the
	// push exhausted retries, so the invoice stays open for the backup close.
	task := pushTask{
		workspaceID: workspaceID,
		req: billingmeter.PushRequest{
			StripeCustomerID: ws.StripeCustomerID.String,
			Values:           values,
			Timestamp:        eventTimestamp,
		},
	}
	if _, perr := hydrav1.NewDeployBillingPushServiceClient(ctx, workspaceID).
		PushWorkspaceUsage().Request(task.pushRequest()); perr != nil {
		logger.Error("final usage push failed; leaving this workspace's invoice open for the backup close",
			"billing_period", period,
			"workspace_id", workspaceID,
			"error", perr,
		)
		return workspacePushOutcome{pushed: false, failed: true}, nil
	}
	return workspacePushOutcome{pushed: true, failed: false}, nil
}
