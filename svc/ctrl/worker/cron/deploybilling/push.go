package deploybilling

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// pushConcurrency bounds how many workspaces we push to the provider at once.
// Pushing one by one is slow for large tenant counts; pushing all at once risks
// the provider's rate limits, so we fan out in batches of this size.
const pushConcurrency = 16

// pushTask is one eligible workspace push, resolved after filtering so the
// fan-out loop only deals with work that should actually happen.
type pushTask struct {
	workspaceID string
	req         billingmeter.PushRequest
}

// pushOutcome is the journaled result of one workspace's push. Failures are
// recorded as values, never returned as errors: a returned error would make
// Restate retry the invocation forever, and the virtual object key would
// then serialize every later tick behind the wedged one. One workspace with
// broken Stripe state (deleted customer, frozen test clock) must not block
// everyone else's billing push; the next tick re-pushes the same absolute
// totals anyway, so a failed tick self-heals as soon as the cause clears.
type pushOutcome struct {
	Meters int
	Failed bool
}

// pushAll fans the per-workspace pushes out in batches of pushConcurrency,
// awaiting each batch before starting the next. Returns the number of
// workspaces pushed, meter events pushed, and workspaces whose push failed.
func (h *Handler) pushAll(ctx restate.ObjectContext, tasks []pushTask) (workspacesPushed, metersPushed, workspacesFailed int, err error) {
	for start := 0; start < len(tasks); start += pushConcurrency {
		end := min(start+pushConcurrency, len(tasks))
		batch := tasks[start:end]

		futures := make([]restate.RunAsyncFuture[pushOutcome], len(batch))
		for i, task := range batch {
			futures[i] = restate.RunAsync(ctx, func(rc restate.RunContext) (pushOutcome, error) {
				n, pushErr := h.pusher.SetMonthToDate(rc, task.req)
				if pushErr != nil {
					// The pusher's own retries (transient errors) are already
					// exhausted by the time this returns; journal the failure
					// and move on.
					logger.Error("deploy billing push failed",
						"workspace_id", task.workspaceID,
						"stripe_customer_id", task.req.StripeCustomerID,
						"error", pushErr,
					)
					return pushOutcome{Meters: 0, Failed: true}, nil
				}
				// Shadow numbers, logged even when the noop pusher sent nothing.
				logger.Info("deploy billing push",
					"workspace_id", task.workspaceID,
					"stripe_customer_id", task.req.StripeCustomerID,
					"cpu_seconds", task.req.Values.CPUSeconds,
					"memory_gib_seconds", task.req.Values.MemoryGiBSeconds,
					"active_keys", task.req.Values.ActiveKeys,
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
				return 0, 0, 0, fmt.Errorf("push usage for workspace %s: %w", batch[i].workspaceID, resultErr)
			}
			if outcome.Failed {
				workspacesFailed++
				continue
			}
			if outcome.Meters > 0 {
				workspacesPushed++
				metersPushed += outcome.Meters
			}
		}
	}
	return workspacesPushed, metersPushed, workspacesFailed, nil
}
