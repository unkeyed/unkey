package deploybilling

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// pushConcurrency caps parallel Stripe pushes per batch.
const pushConcurrency = 16

type pushTask struct {
	workspaceID string
	req         billingmeter.PushRequest
}

// pushOutcome journals one workspace push. Failures are values, not errors.
// A returned error would wedge the VO and block every later tick for that month.
// One bad Stripe customer must not stall everyone; the next tick re-sends totals.
type pushOutcome struct {
	Meters int
	Failed bool
}

// pushAll pushes tasks in batches. Returns failed workspace IDs so callers
// can leave those invoices in draft instead of finalizing an undercount.
func (h *Handler) pushAll(ctx restate.ObjectContext, tasks []pushTask) (workspacesPushed, metersPushed int, failedWorkspaceIDs []string, err error) {
	for start := 0; start < len(tasks); start += pushConcurrency {
		end := min(start+pushConcurrency, len(tasks))
		batch := tasks[start:end]

		futures := make([]restate.RunAsyncFuture[pushOutcome], len(batch))
		for i, task := range batch {
			futures[i] = restate.RunAsync(ctx, func(rc restate.RunContext) (pushOutcome, error) {
				n, pushErr := h.pusher.Push(rc, task.req)
				if pushErr != nil {
					// Pusher retries are exhausted. Journal failure and continue.
					logger.Error("deploy billing push failed",
						"workspace_id", task.workspaceID,
						"stripe_customer_id", task.req.StripeCustomerID,
						"error", pushErr,
					)
					return pushOutcome{Meters: 0, Failed: true}, nil
				}
				// Logged even when the noop pusher sent nothing.
				logger.Info("deploy billing push",
					"workspace_id", task.workspaceID,
					"stripe_customer_id", task.req.StripeCustomerID,
					"cpu_seconds", task.req.Values.CPUSeconds,
					"memory_gib_seconds", task.req.Values.MemoryGiBSeconds,
					"egress_gib", task.req.Values.EgressGiB,
					"disk_gib_seconds", task.req.Values.DiskGiBSeconds,
					"meters_pushed", n,
				)
				return pushOutcome{Meters: n, Failed: false}, nil
			}, restate.WithName("push "+task.workspaceID))
		}

		for i, fut := range futures {
			outcome, resultErr := fut.Result()
			if resultErr != nil {
				return 0, 0, nil, fmt.Errorf("push usage for workspace %s: %w", batch[i].workspaceID, resultErr)
			}
			if outcome.Failed {
				failedWorkspaceIDs = append(failedWorkspaceIDs, batch[i].workspaceID)
				continue
			}
			if outcome.Meters > 0 {
				workspacesPushed++
				metersPushed += outcome.Meters
			}
		}
	}
	return workspacesPushed, metersPushed, failedWorkspaceIDs, nil
}
