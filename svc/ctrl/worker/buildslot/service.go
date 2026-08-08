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

// InvocationLiveness answers whether Restate still tracks the given
// invocation IDs. It is the ground truth for slot audits: Virtual Object
// state lives independently of invocation lifecycles, so a deployment ID in
// active_slots proves nothing about its Deploy invocation still existing.
// Implemented by [pkg/restate/admin.Client] via sys_invocation introspection.
type InvocationLiveness interface {
	FindLiveInvocations(ctx context.Context, invocationIDs []string) (map[string]bool, error)
}

const (
	// MaxWaitDuration bounds how long a Deploy workflow may wait for a build
	// slot before giving up with a terminal error. The Deploy handler races
	// its slot awakeable against this timeout, so a waiter can never park
	// forever even if slot accounting goes wrong upstream.
	//
	// Sized for the worst honest case: a workspace at its concurrency limit
	// with a deep queue of preview builds ahead. Beyond this, failing loudly
	// beats silently waiting for days.
	MaxWaitDuration = 6 * time.Hour

	// slotLeaseDuration is how long a granted slot may be held before
	// ExpireSlot fires and audits it. It must comfortably exceed the longest
	// legitimate Deploy run after slot grant (build + rollout + readiness,
	// normally well under an hour). A deployment still non-terminal after
	// the lease is considered stuck and is force-failed.
	slotLeaseDuration = 4 * time.Hour

	// waiterExpiryDelay is when ExpireSlot audits a wait-list entry. It is
	// deliberately longer than MaxWaitDuration: a live waiter times itself
	// out and releases its entry first, so anything still in the wait list
	// at this point belongs to a dead invocation.
	waiterExpiryDelay = MaxWaitDuration + 15*time.Minute

	// expireRetryDelay re-arms an ExpireSlot check whose database read kept
	// failing. Losing the lease check would resurrect the permanent-leak
	// bug, so we retry later instead of giving up.
	expireRetryDelay = 10 * time.Minute

	// runMaxAttempts bounds restate.Run retries inside the virtual object.
	// The VO holds the per-workspace key lock while a handler runs, so an
	// unbounded retry here (e.g. a missing limits row returning ErrNoRows
	// forever) wedges every AcquireOrWait/Release for the workspace.
	runMaxAttempts uint = 5

	// defaultBuildLimit applies when the workspace has no limits row. One
	// concurrent build is the conservative floor; it keeps the queue moving
	// instead of freezing the whole workspace on a terminal error.
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
