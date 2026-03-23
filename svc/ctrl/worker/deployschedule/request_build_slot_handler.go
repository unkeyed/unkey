package deployschedule

import (
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// defaultMaxConcurrentBuilds is used when the workspace has no explicit limit.
const defaultMaxConcurrentBuilds = 3

// RequestBuildSlot checks if a build slot is available for the requesting queue.
// If a slot is available, it is immediately granted. If not, the queue key is
// added to the waitlist and the scheduler will call ProcessNext on it when a
// slot opens via ReleaseBuildSlot.
func (s *Service) RequestBuildSlot(ctx restate.ObjectContext, req *hydrav1.RequestBuildSlotRequest) (*hydrav1.RequestBuildSlotResponse, error) {
	activeSlots, _ := restate.Get[map[string]activeSlot](ctx, stateActiveSlots)
	if activeSlots == nil {
		activeSlots = make(map[string]activeSlot)
	}

	// Load the workspace's concurrency limit from the quotas table.
	maxBuilds, err := restate.Run(ctx, func(runCtx restate.RunContext) (uint32, error) {
		quota, findErr := db.Query.FindQuotaByWorkspaceID(runCtx, s.db.RO(), restate.Key(ctx))
		if findErr != nil {
			return defaultMaxConcurrentBuilds, findErr
		}
		if quota.MaxConcurrentBuilds <= 0 {
			return defaultMaxConcurrentBuilds, nil
		}
		return quota.MaxConcurrentBuilds, nil
	}, restate.WithName("load max concurrent builds"))
	if err != nil {
		// If we can't read the workspace, use the default and continue.
		maxBuilds = defaultMaxConcurrentBuilds
		logger.Error("failed to load workspace concurrency limit, using default",
			"workspace_id", restate.Key(ctx),
			"default", maxBuilds,
			"error", err,
		)
	}

	if len(activeSlots) < int(maxBuilds) {
		// Slot available — grant it.
		now, _ := restate.Run(ctx, func(_ restate.RunContext) (int64, error) {
			return time.Now().UnixMilli(), nil
		}, restate.WithName("get timestamp"))

		activeSlots[req.GetQueueKey()] = activeSlot{
			AcquiredAt: now,
		}
		restate.Set(ctx, stateActiveSlots, activeSlots)

		logger.Info("build slot granted",
			"queue_key", req.GetQueueKey(),
			"active", len(activeSlots),
			"max", maxBuilds,
		)

		return &hydrav1.RequestBuildSlotResponse{Granted: true}, nil
	}

	// No slot available — add to waitlist.
	waitlist, _ := restate.Get[[]waitlistEntry](ctx, stateWaitlist)

	entry := waitlistEntry{
		QueueKey:     req.GetQueueKey(),
		IsProduction: req.GetIsProduction(),
	}

	// Insert with priority: production entries go before preview entries.
	inserted := false
	for i, existing := range waitlist {
		if entry.IsProduction && !existing.IsProduction {
			waitlist = append(waitlist[:i+1], waitlist[i:]...)
			waitlist[i] = entry
			inserted = true
			break
		}
	}
	if !inserted {
		waitlist = append(waitlist, entry)
	}
	restate.Set(ctx, stateWaitlist, waitlist)

	logger.Info("build slot denied, added to waitlist",
		"queue_key", req.GetQueueKey(),
		"active", len(activeSlots),
		"max", maxBuilds,
		"waitlist_size", len(waitlist),
	)

	return &hydrav1.RequestBuildSlotResponse{Granted: false}, nil
}
