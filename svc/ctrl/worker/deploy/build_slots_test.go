package deploy_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

// buildSlotFixture creates a workspace + quota (max_concurrent_builds=1) +
// project + app + environment for build-slot tests, and exposes helpers
// to mint deployments, manipulate the queue, and call into the
// release-and-promote logic the deploy handler depends on.
type buildSlotFixture struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func newBuildSlotFixture(t *testing.T, h *harness.Harness) buildSlotFixture {
	t.Helper()

	// RequestsPerMonth > 0 triggers the quota row insert; max_concurrent_builds
	// keeps its schema default of 1, which is what every test below relies on.
	ws := h.Seed.CreateWorkspaceWithQuota(h.Ctx, seed.CreateWorkspaceWithQuotaRequest{
		RequestsPerMonth: 1000,
	})

	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID:               uid.New(uid.ProjectPrefix),
		WorkspaceID:      ws.ID,
		Name:             "test-project",
		Slug:             uid.New("slug"),
		DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: ws.ID,
		ProjectID:   project.ID,
		Name:        "default",
		Slug:        "default",
	})
	env := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      ws.ID,
		ProjectID:        project.ID,
		AppID:            app.ID,
		Slug:             "preview",
		Description:      "",
		SentinelConfig:   nil,
		DeleteProtection: false,
	})

	return buildSlotFixture{
		workspaceID:   ws.ID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: env.ID,
	}
}

func (f buildSlotFixture) createDeployment(t *testing.T, h *harness.Harness, status db.DeploymentsStatus) db.Deployment {
	t.Helper()
	return h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: f.environmentID,
		Status:        status,
	})
}

func (f buildSlotFixture) acquireSlot(t *testing.T, h *harness.Harness, dep db.Deployment) bool {
	t.Helper()
	rows, err := db.Query.TryAcquireBuildSlot(h.Ctx, h.DB.RW(), db.TryAcquireBuildSlotParams{
		DeploymentID:     dep.ID,
		WorkspaceID:      f.workspaceID,
		AcquiredAt:       time.Now().UnixMilli(),
		TerminalStatuses: db.TerminalDeploymentStatuses,
	})
	require.NoError(t, err)
	require.Contains(t, []int64{0, 1}, rows)
	return rows == 1
}

func (f buildSlotFixture) registerWaiter(t *testing.T, h *harness.Harness, dep db.Deployment, awakeableID string, isProduction bool, enqueuedAt int64) {
	t.Helper()
	require.NoError(t, db.Query.RegisterBuildSlotWaiter(h.Ctx, h.DB.RW(), db.RegisterBuildSlotWaiterParams{
		DeploymentID: dep.ID,
		WorkspaceID:  f.workspaceID,
		AwakeableID:  awakeableID,
		IsProduction: isProduction,
		EnqueuedAt:   enqueuedAt,
	}))
}

// release runs ReleaseAndPromoteBuildSlot in a real tx against the harness
// DB so the test exercises the same code path as the production handler
// (minus the restate.Run wrapper). Returns the promotion the loop produced,
// or nil when no waiter was promoted.
func (f buildSlotFixture) release(t *testing.T, h *harness.Harness, deploymentID string) *deploy.BuildSlotPromotion {
	t.Helper()
	var promoted *deploy.BuildSlotPromotion
	err := db.TxRetry(h.Ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		p, err := deploy.ReleaseAndPromoteBuildSlot(ctx, tx, f.workspaceID, deploymentID, time.Now().UnixMilli())
		promoted = p
		return err
	})
	require.NoError(t, err)
	return promoted
}

func (f buildSlotFixture) countWaiters(t *testing.T, h *harness.Harness) int {
	t.Helper()
	var n int
	err := h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM build_slot_waiters WHERE workspace_id = ?",
		f.workspaceID,
	).Scan(&n)
	require.NoError(t, err)
	return n
}

// ---- Acquire-side tests: one query each, so they exercise behavior directly. ----

func TestBuildSlot_TryAcquire_BelowCapacityGrants(t *testing.T) {
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)
	dep := f.createDeployment(t, h, db.DeploymentsStatusBuilding)

	require.True(t, f.acquireSlot(t, h, dep), "first acquire below capacity must grant")
}

func TestBuildSlot_TryAcquire_AtCapacityRejects(t *testing.T) {
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	held := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, held))

	contender := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.False(t, f.acquireSlot(t, h, contender), "second acquire at capacity must not grant")
}

func TestBuildSlot_TryAcquire_IgnoresLeakedTerminalSlots(t *testing.T) {
	// The whole point of joining build_slots against deployments.status: a
	// leaked row (e.g. handler purged before compensation ran) must not
	// count against capacity, otherwise the workspace deadlocks at cap.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	leaked := f.createDeployment(t, h, db.DeploymentsStatusReady)
	_, err := h.DB.RW().ExecContext(h.Ctx,
		"INSERT INTO build_slots (deployment_id, workspace_id, acquired_at) VALUES (?, ?, ?)",
		leaked.ID, f.workspaceID, time.Now().UnixMilli(),
	)
	require.NoError(t, err)

	fresh := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, fresh), "leaked terminal slot must not count against capacity")
}

func TestBuildSlot_HoldsBuildSlot_ShortCircuitsReAcquireOnPreGrant(t *testing.T) {
	// Regression for the wake-up loop bug class in waitForBuildSlot.
	//
	// `build_slots` is the source of truth: any path that leaves our row
	// in the table means we own the slot. The loop checks HoldsBuildSlot
	// before TryAcquireBuildSlot precisely so that a pre-granted
	// deployment returns granted=true without re-running the INSERT —
	// because the raw INSERT would either deadlock (max=1: our own row
	// makes count == limit, 0 rows inserted, re-park forever) or
	// PK-violate (max>1: INSERT proceeds, duplicate deployment_id error).
	//
	// This test covers both wake-up paths (awakeable fired AND timeout
	// after holder crashed mid-handoff) because the loop fix treats them
	// identically — the HoldsBuildSlot check is the single source of
	// truth, run on every iteration.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h) // max_concurrent_builds defaults to 1

	promoted := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	_, err := h.DB.RW().ExecContext(h.Ctx,
		"INSERT INTO build_slots (deployment_id, workspace_id, acquired_at) VALUES (?, ?, ?)",
		promoted.ID, f.workspaceID, time.Now().UnixMilli(),
	)
	require.NoError(t, err)

	holds, err := db.Query.HoldsBuildSlot(h.Ctx, h.DB.RO(), promoted.ID)
	require.NoError(t, err)
	require.True(t, holds, "HoldsBuildSlot must return true for a deployment with an existing build_slots row")

	// Confirm the latent bug the short-circuit avoids: at unit capacity,
	// re-running TryAcquireBuildSlot for an already-pre-granted deployment
	// silently returns 0 rows (count includes our own row, count == limit).
	require.False(t, f.acquireSlot(t, h, promoted),
		"TryAcquireBuildSlot for an already pre-granted deployment must not grant at unit capacity; "+
			"waitForBuildSlot relies on HoldsBuildSlot to short-circuit before reaching this call")
}

func TestBuildSlot_HoldsBuildSlot_FalseWhenNoRow(t *testing.T) {
	// Negative case: HoldsBuildSlot must return false for a deployment
	// that has not been granted or pre-granted, so the loop falls through
	// to TryAcquireBuildSlot on the normal acquire path.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)
	dep := f.createDeployment(t, h, db.DeploymentsStatusBuilding)

	holds, err := db.Query.HoldsBuildSlot(h.Ctx, h.DB.RO(), dep.ID)
	require.NoError(t, err)
	require.False(t, holds, "HoldsBuildSlot must return false when no build_slots row exists for the deployment")
}

func TestBuildSlot_TryAcquire_OnPreGrantedDeploymentWithSpareCapacity(t *testing.T) {
	// Sibling regression documenting the second failure mode of the
	// wake-up loop bug: when max_concurrent_builds > 1 and the workspace
	// has spare capacity at the moment a pre-granted deployment would
	// re-acquire, TryAcquireBuildSlot's INSERT proceeds (count < limit
	// is true) and hits a duplicate-key error on deployment_id — because
	// our pre-granted row is already there.
	//
	// The HoldsBuildSlot check in waitForBuildSlot's loop avoids this
	// path entirely; this test pins the underlying SQL behavior so a
	// future change to remove the EXISTS guard can't silently regress.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)
	_, err := h.DB.RW().ExecContext(h.Ctx,
		"UPDATE quota SET max_concurrent_builds = 2 WHERE workspace_id = ?",
		f.workspaceID,
	)
	require.NoError(t, err)

	promoted := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	_, err = h.DB.RW().ExecContext(h.Ctx,
		"INSERT INTO build_slots (deployment_id, workspace_id, acquired_at) VALUES (?, ?, ?)",
		promoted.ID, f.workspaceID, time.Now().UnixMilli(),
	)
	require.NoError(t, err)

	// max=2, current count=1 (us), so count < limit is true and the
	// INSERT proceeds — straight into a duplicate-key error.
	_, err = db.Query.TryAcquireBuildSlot(h.Ctx, h.DB.RW(), db.TryAcquireBuildSlotParams{
		DeploymentID:     promoted.ID,
		WorkspaceID:      f.workspaceID,
		AcquiredAt:       time.Now().UnixMilli(),
		TerminalStatuses: db.TerminalDeploymentStatuses,
	})
	require.Error(t, err,
		"TryAcquireBuildSlot for an already pre-granted deployment must surface a duplicate-key error "+
			"when capacity allows the INSERT to proceed; waitForBuildSlot's HoldsBuildSlot guard avoids this path")
}

func TestBuildSlot_RegisterWaiter_UpdatesAwakeableOnReentry(t *testing.T) {
	// A Deploy retry creates a fresh awakeable; the waiter row must be
	// updated in-place so the next Release wakes the live handler.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)
	dep := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	now := time.Now().UnixMilli()

	f.registerWaiter(t, h, dep, "awk-original", false, now)
	f.registerWaiter(t, h, dep, "awk-replacement", false, now+1)

	require.Equal(t, 1, f.countWaiters(t, h), "re-register must not duplicate the row")
	next, err := db.Query.PickNextBuildSlotWaiter(h.Ctx, h.DB.RW(), f.workspaceID)
	require.NoError(t, err)
	require.Equal(t, "awk-replacement", next.AwakeableID)
}

// ---- Release-side tests: target ReleaseAndPromoteBuildSlot's composed behavior. ----

func TestBuildSlot_Release_NoWaitersReturnsEmpty(t *testing.T) {
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)
	holder := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, holder))

	prom := f.release(t, h, holder.ID)
	require.Nil(t, prom, "release with empty queue must not promote")

	// Slot is gone so a fresh acquire succeeds.
	fresh := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, fresh))
}

func TestBuildSlot_Release_PromotesProductionAheadOfPreview(t *testing.T) {
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	holder := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, holder))

	preview := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	prod := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	now := time.Now().UnixMilli()
	f.registerWaiter(t, h, preview, "awk-preview", false, now)
	f.registerWaiter(t, h, prod, "awk-prod", true, now+1000) // enqueued strictly later

	prom := f.release(t, h, holder.ID)
	require.NotNil(t, prom)
	require.Equal(t, "awk-prod", prom.AwakeableID, "production waiter must beat preview")
	require.Equal(t, prod.ID, prom.DeploymentID)

	// Promoted waiter is removed from the queue; preview waiter still parked.
	require.Equal(t, 1, f.countWaiters(t, h))
}

func TestBuildSlot_Release_FIFOWithinPriority(t *testing.T) {
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	holder := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, holder))

	first := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	second := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	now := time.Now().UnixMilli()
	f.registerWaiter(t, h, first, "awk-first", true, now)
	f.registerWaiter(t, h, second, "awk-second", true, now+1000)

	prom := f.release(t, h, holder.ID)
	require.NotNil(t, prom)
	require.Equal(t, first.ID, prom.DeploymentID, "oldest waiter within priority must win")
}

func TestBuildSlot_Release_SkipsTerminalWaitersAndPromotesLiveOne(t *testing.T) {
	// Head waiter has gone terminal (was cancelled mid-wait but its row
	// wasn't cleaned up). Release must drop the dead row and promote the
	// next live waiter in a single transaction.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	holder := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, holder))

	dead := f.createDeployment(t, h, db.DeploymentsStatusCancelled)
	alive := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	now := time.Now().UnixMilli()
	// dead is enqueued first AND production, so it's the head waiter.
	f.registerWaiter(t, h, dead, "awk-dead", true, now)
	f.registerWaiter(t, h, alive, "awk-alive", false, now+1)

	prom := f.release(t, h, holder.ID)
	require.NotNil(t, prom)
	require.Equal(t, alive.ID, prom.DeploymentID, "dead head waiter must be skipped")
	require.Equal(t, "awk-alive", prom.AwakeableID)

	// Both waiter rows are gone — the dead one was dropped, the live one
	// was promoted out.
	require.Equal(t, 0, f.countWaiters(t, h))
}

func TestBuildSlot_Release_AllWaitersTerminalDropsAllAndReturnsEmpty(t *testing.T) {
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	holder := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, holder))

	dead1 := f.createDeployment(t, h, db.DeploymentsStatusCancelled)
	dead2 := f.createDeployment(t, h, db.DeploymentsStatusFailed)
	now := time.Now().UnixMilli()
	f.registerWaiter(t, h, dead1, "awk-1", true, now)
	f.registerWaiter(t, h, dead2, "awk-2", false, now+1)

	prom := f.release(t, h, holder.ID)
	require.Nil(t, prom, "no live waiter, no promotion")
	require.Equal(t, 0, f.countWaiters(t, h), "all dead waiters must be cleaned up")
}

func TestBuildSlot_Release_DropsReleaserOwnWaiterRow(t *testing.T) {
	// When a deployment is cancelled mid-wait its compensation calls
	// releaseBuildSlot. It never held a slot, but it did register itself
	// as a waiter — that row must go so a later release doesn't promote
	// the now-dead handler.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	cancelled := f.createDeployment(t, h, db.DeploymentsStatusCancelled)
	f.registerWaiter(t, h, cancelled, "awk-dead", false, time.Now().UnixMilli())
	require.Equal(t, 1, f.countWaiters(t, h))

	prom := f.release(t, h, cancelled.ID)
	require.Nil(t, prom)
	require.Equal(t, 0, f.countWaiters(t, h), "releaser's own waiter row must be dropped")
}

func TestBuildSlot_Release_IsIdempotent(t *testing.T) {
	// Compensation may fire alongside the happy-path release. The second
	// call must be a no-op, not a duplicate promotion.
	h := harness.New(t)
	f := newBuildSlotFixture(t, h)

	holder := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	require.True(t, f.acquireSlot(t, h, holder))

	waiter := f.createDeployment(t, h, db.DeploymentsStatusBuilding)
	f.registerWaiter(t, h, waiter, "awk-once", false, time.Now().UnixMilli())

	first := f.release(t, h, holder.ID)
	require.NotNil(t, first)
	require.Equal(t, waiter.ID, first.DeploymentID)

	second := f.release(t, h, holder.ID)
	require.Nil(t, second, "second release must not re-promote")
}
