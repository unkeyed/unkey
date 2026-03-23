package deployschedule

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
)

// ReleaseBuildSlot frees a build slot and dispatches the next waiting queue.
// Called by DeployQueueService.OnDeployComplete when a deploy finishes.
func (s *Service) ReleaseBuildSlot(ctx restate.ObjectContext, req *hydrav1.ReleaseBuildSlotRequest) (*hydrav1.ReleaseBuildSlotResponse, error) {
	activeSlots, _ := restate.Get[map[string]activeSlot](ctx, stateActiveSlots)
	if activeSlots == nil {
		activeSlots = make(map[string]activeSlot)
	}

	delete(activeSlots, req.GetQueueKey())
	restate.Set(ctx, stateActiveSlots, activeSlots)

	logger.Info("build slot released",
		"queue_key", req.GetQueueKey(),
		"active", len(activeSlots),
	)

	// Wake the next waiting queue, if any.
	waitlist, _ := restate.Get[[]waitlistEntry](ctx, stateWaitlist)
	if len(waitlist) > 0 {
		next := waitlist[0]
		waitlist = waitlist[1:]
		restate.Set(ctx, stateWaitlist, waitlist)

		logger.Info("waking next queue from waitlist",
			"queue_key", next.QueueKey,
			"remaining_waitlist", len(waitlist),
		)

		hydrav1.NewDeployQueueServiceClient(ctx, next.QueueKey).ProcessNext().
			Send(&hydrav1.ProcessNextRequest{})
	}

	return &hydrav1.ReleaseBuildSlotResponse{}, nil
}
