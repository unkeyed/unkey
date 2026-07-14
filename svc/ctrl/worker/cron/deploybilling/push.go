package deploybilling

import (
	"fmt"
	"sort"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Handle computes month-to-date Deploy usage for the billing period (the VO
// key, "YYYY-MM") and fans out one push invocation per billable workspace,
// then awaits each child so a failed push withholds the orchestrator heartbeat.
// Each push runs, retries, and fails on its own; the pushed quantity is
// absolute, so re-runs and overlapping ticks converge.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeployBillingPushRequest,
) (*hydrav1.RunDeployBillingPushResponse, error) {
	period := restate.Key(ctx)
	logger.Info("running deploy billing push", "billing_period", period)

	if h.usage == nil {
		logger.Info("deploy billing push disabled (no usage reader configured)")
		return &hydrav1.RunDeployBillingPushResponse{}, nil
	}

	p, err := billingperiod.Parse(period)
	if err != nil {
		return nil, fmt.Errorf("invalid billing period %q: %w", period, err)
	}

	nowTime, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	// The ClickHouse usage query is bounded in milliseconds; the Stripe meter
	// event timestamp is unix seconds. Both come from the one journaled "now"
	// so the window and the event agree.
	nowMillis := nowTime.UnixMilli()
	nowUnixSeconds := nowTime.Unix()

	tasks, workspacesWithUsage, err := h.resolvePushTasks(ctx, period, p, nowMillis, nowUnixSeconds)
	if err != nil {
		return nil, err
	}

	// Dispatch each workspace's push, then await outcomes so a failed child
	// withholds the orchestrator heartbeat. Each push still runs in its own VO
	// with independent retries; awaiting only surfaces success/failure here.
	type pushFuture = restate.ResponseFuture[*hydrav1.PushWorkspaceUsageResponse]
	pushFutures := make([]pushFuture, len(tasks))
	for i, task := range tasks {
		pushFutures[i] = hydrav1.NewDeployBillingPushServiceClient(ctx, task.workspaceID).
			PushWorkspaceUsage().RequestFuture(task.pushRequest())
	}
	pushFailed := 0
	for i, fut := range pushFutures {
		if _, err := fut.Response(); err != nil {
			pushFailed++
			logger.Error("deploy billing push child failed",
				"billing_period", period,
				"workspace_id", tasks[i].workspaceID,
				"error", err,
			)
		}
	}

	logger.Info("deploy billing push complete",
		"billing_period", period,
		"workspaces_with_usage", workspacesWithUsage,
		"workspaces_dispatched", len(tasks),
		"workspaces_push_failed", pushFailed,
	)

	if pushFailed == 0 {
		if err := h.pingHeartbeat(ctx); err != nil {
			return nil, err
		}
	} else {
		logger.Error("deploy billing push withheld heartbeat after child failures",
			"billing_period", period,
			"workspaces_push_failed", pushFailed,
		)
	}
	return &hydrav1.RunDeployBillingPushResponse{}, nil
}

// pingHeartbeat reports a successful run to the monitoring heartbeat.
func (h *Handler) pingHeartbeat(ctx restate.ObjectContext) error {
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	return nil
}

// pushTask is one eligible workspace push, resolved after filtering so the
// fan-out only deals with work that should actually happen.
type pushTask struct {
	workspaceID string
	req         billingmeter.PushRequest
}

// pushRequest builds the protobuf request for a per-workspace push invocation.
func (t pushTask) pushRequest() *hydrav1.PushWorkspaceUsageRequest {
	return &hydrav1.PushWorkspaceUsageRequest{
		StripeCustomerId: t.req.StripeCustomerID,
		CpuSeconds:       t.req.Values.CPUSeconds,
		MemoryGibSeconds: t.req.Values.MemoryGiBSeconds,
		EgressGib:        t.req.Values.EgressGiB,
		DiskGibSeconds:   t.req.Values.DiskGiBSeconds,
		ActiveKeys:       t.req.Values.ActiveKeys,
		EventTimestamp:   t.req.Timestamp,
	}
}

// resolvePushTasks computes billable usage for [p.Start(), endMillis) and
// returns one task per workspace that should be pushed, stamping the meter
// events with eventTimestamp. Shared by the hourly push (end = timestamp =
// now) and the month-end close (end = period end, timestamp just inside the
// closed period so the "last"-formula meters bill the final total).
//
// It does not push: the caller fans out the tasks to the per-workspace push
// service and awaits the outcomes.
func (h *Handler) resolvePushTasks(
	ctx restate.ObjectContext,
	period string,
	p billingperiod.Period,
	endMillis int64,
	eventTimestamp int64,
) (tasks []pushTask, workspacesWithUsage int, err error) {
	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return h.usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: "", // all workspaces; we filter to billable ones below
			Start:       p.Start().UnixMilli(),
			End:         endMillis,
		})
	}, restate.WithName("get period usage"))
	if err != nil {
		return nil, 0, fmt.Errorf("get period usage: %w", err)
	}

	keyRows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.ActiveKeysUsage, error) {
		return h.usage.GetActiveKeysUsage(rc, clickhouse.GetActiveKeysUsageRequest{
			WorkspaceID: "", // all workspaces; we filter to billable ones below
			Year:        p.Year,
			Month:       p.Month,
		})
	}, restate.WithName("get active keys"))
	if err != nil {
		return nil, 0, fmt.Errorf("get active keys: %w", err)
	}

	valuesByWorkspace := aggregateUsage(rows)
	mergeActiveKeys(valuesByWorkspace, keyRows)
	if len(valuesByWorkspace) == 0 {
		logger.Info("no deploy usage this period", "billing_period", period)
		return nil, 0, nil
	}

	// Stable order so the journaled fan-out steps replay identically.
	workspaceIDs := make([]string, 0, len(valuesByWorkspace))
	for id := range valuesByWorkspace {
		workspaceIDs = append(workspaceIDs, id)
	}
	sort.Strings(workspaceIDs)

	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesForDeployBillingByIDsRow, error) {
		return h.db.ListWorkspacesForDeployBillingByIDs(rc, workspaceIDs)
	}, restate.WithName("fetch workspace billing identities"))
	if err != nil {
		return nil, 0, fmt.Errorf("fetch workspace billing identities: %w", err)
	}

	workspacesByID := make(map[string]db.ListWorkspacesForDeployBillingByIDsRow, len(workspaces))
	for _, w := range workspaces {
		workspacesByID[w.ID] = w
	}

	tasks = make([]pushTask, 0, len(workspaceIDs))
	for _, id := range workspaceIDs {
		values := valuesByWorkspace[id]
		if !values.Positive() {
			continue
		}

		w, ok := workspacesByID[id]
		if !ok {
			continue
		}
		// A disabled workspace is still billed: usage already incurred is owed
		// regardless of the workspace's current state. The only blocker is a
		// missing Stripe customer, since there is nothing to map the usage onto.
		if !w.StripeCustomerID.Valid || w.StripeCustomerID.String == "" {
			logger.Info("workspace has deploy usage but no stripe customer; skipping",
				"workspace_id", id,
				"billing_period", period,
			)
			continue
		}

		tasks = append(tasks, pushTask{
			workspaceID: id,
			req: billingmeter.PushRequest{
				StripeCustomerID: w.StripeCustomerID.String,
				Values:           values,
				Timestamp:        eventTimestamp,
			},
		})
	}

	return tasks, len(valuesByWorkspace), nil
}
