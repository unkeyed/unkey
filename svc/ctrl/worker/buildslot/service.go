// Package buildslot implements the BuildSlotService Restate virtual object,
// which caps how many deployments in a workspace can be actively building at
// the same time.
//
// The virtual object is keyed by workspace_id so that all AcquireOrWait and
// Release calls for a given workspace are serialized, making slot management
// race-free even when multiple (app, environment)-keyed Deploy workflows ask
// for slots concurrently.
//
// State is push-based via Restate awakeables:
//   - `active_slots` holds the set of deployment IDs currently building
//   - `prod_wait_list` is a FIFO of production waiters
//   - `preview_wait_list` is a FIFO of non-production waiters
//
// Both wait lists store {deployment_id, awakeable_id} entries. Production
// deployments respect the workspace's max_concurrent_builds quota — they
// don't bypass the cap — but Release drains prod_wait_list before
// preview_wait_list so a hot-fix priority-queues ahead of preview builds.
//
// This avoids the journal bloat of a poll loop in the Deploy handler: a
// waiting handler is parked on a single Restate wait operation until
// explicitly woken.
package buildslot

import (
	"context"
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// InvocationLiveness reports which of the given invocation IDs still
// execute in Restate. Slot audits use it because Virtual Object state
// outlives invocations: a deployment ID in active_slots does not prove
// that its Deploy invocation still runs. Implemented by
// [pkg/restate/admin.Client] through sys_invocation introspection.
type InvocationLiveness interface {
	FindLiveInvocations(ctx context.Context, invocationIDs []string) (map[string]bool, error)
}

const (
	// MaxWaitDuration is the maximum time a Deploy workflow waits for a
	// build slot. The Deploy handler races its awakeable against this
	// timeout. A waiter cannot stay parked longer, even when the slot
	// accounting is wrong.
	//
	// One hour covers a queue of normal builds at the concurrency limit.
	// The acquire-time audit reclaims dead slots on demand, so a longer
	// wait means a deep queue or a defect. A visible failure after one
	// hour is better than a silent multi-hour wait; the user can deploy
	// again.
	MaxWaitDuration = 1 * time.Hour

	// slotLeaseDuration is the lease check interval for a granted slot.
	// ExpireSlot fires after each interval and checks the deployment. A
	// live Deploy invocation renews the lease (see maxSlotLeaseRenewals),
	// so a real long build keeps its slot. A dead invocation loses the
	// slot at the next check.
	slotLeaseDuration = 1 * time.Hour

	// maxSlotLeaseRenewals caps lease renewals for a live invocation. A
	// slot can be held for at most
	// slotLeaseDuration * (maxSlotLeaseRenewals + 1), 8 hours in total.
	// After that the deployment is force-failed even when its invocation
	// is still live: at that point it is hung, not building.
	maxSlotLeaseRenewals uint32 = 7

	// waiterExpiryDelay is when ExpireSlot audits a wait-list entry. It is
	// longer than MaxWaitDuration on purpose: a live waiter times out and
	// removes its own entry first. An entry that remains belongs to a dead
	// invocation.
	waiterExpiryDelay = MaxWaitDuration + 15*time.Minute

	// expireRetryDelay re-arms an ExpireSlot check after its database read
	// failed. A dropped lease check would allow permanent leaks again, so
	// retry later instead of giving up.
	expireRetryDelay = 10 * time.Minute

	// runMaxAttempts bounds restate.Run retries inside the virtual object.
	// The VO holds the per-workspace key lock while a handler runs. An
	// unbounded retry (for example, on a missing limits row) blocks every
	// AcquireOrWait and Release for the workspace.
	runMaxAttempts uint = 5

	// defaultBuildLimit applies when the workspace has no limits row. One
	// concurrent build keeps the queue moving instead of failing the
	// workspace.
	defaultBuildLimit uint32 = 1
)

// Service implements the BuildSlotService Restate virtual object.
// Key: workspace_id.
type Service struct {
	hydrav1.UnimplementedBuildSlotServiceServer
	db           db.Database
	restateAdmin InvocationLiveness
}

var _ hydrav1.BuildSlotServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database
	// RestateAdmin is used to verify that deployments occupying slots still
	// have live Restate invocations behind them.
	RestateAdmin InvocationLiveness
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedBuildSlotServiceServer: hydrav1.UnimplementedBuildSlotServiceServer{},
		db:                                  cfg.DB,
		restateAdmin:                        cfg.RestateAdmin,
	}
}
