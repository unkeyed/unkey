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

// pushAll fans the per-workspace pushes out in batches of pushConcurrency,
// awaiting each batch before starting the next. Returns the number of
// workspaces and meter events pushed.
func (h *Handler) pushAll(ctx restate.ObjectContext, tasks []pushTask) (workspacesPushed, metersPushed int, err error) {
	for start := 0; start < len(tasks); start += pushConcurrency {
		end := min(start+pushConcurrency, len(tasks))
		batch := tasks[start:end]

		futures := make([]restate.RunAsyncFuture[int], len(batch))
		for i, task := range batch {
			futures[i] = restate.RunAsync(ctx, func(rc restate.RunContext) (int, error) {
				n, err := h.pusher.Push(rc, task.req)
				if err != nil {
					return 0, err
				}
				// Shadow numbers, logged even when the noop pusher sent nothing.
				logger.Info("deploy billing push",
					"workspace_id", task.workspaceID,
					"stripe_customer_id", task.req.StripeCustomerID,
					"cpu_seconds", task.req.Values.CPUSeconds,
					"memory_gib_seconds", task.req.Values.MemoryGiBSeconds,
					"egress_gib", task.req.Values.EgressGiB,
					"disk_gib_seconds", task.req.Values.DiskGiBSeconds,
					"meters_pushed", n,
				)
				return n, nil
			}, restate.WithName("push "+task.workspaceID))
		}

		for i, fut := range futures {
			n, pushErr := fut.Result()
			if pushErr != nil {
				return 0, 0, fmt.Errorf("push usage for workspace %s: %w", batch[i].workspaceID, pushErr)
			}
			if n > 0 {
				workspacesPushed++
				metersPushed += n
			}
		}
	}
	return workspacesPushed, metersPushed, nil
}
