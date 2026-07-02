package deploybilling

import (
	"fmt"
	"sort"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

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
// service, fire-and-forget for the hourly tick or awaited for the close.
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
			Month:       p.Start().UnixMilli(),
		})
	}, restate.WithName("get active keys"))
	if err != nil {
		return nil, 0, fmt.Errorf("get active keys: %w", err)
	}

	valuesByWorkspace := AggregateUsage(rows)
	MergeActiveKeys(valuesByWorkspace, keyRows)
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
		return db.Query.ListWorkspacesForDeployBillingByIDs(rc, h.db.RO(), workspaceIDs)
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
