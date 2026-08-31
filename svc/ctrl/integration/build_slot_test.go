//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/restatedev/sdk-go/server"
	restatetest "github.com/restatedev/sdk-go/testing"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/buildslot"
)

// restateImage pins the same Restate version and digest that production
// runs (dev/k8s/manifests/restate.yaml). The stale-slot reproduction
// depends on real kill and purge semantics, so the test must not float
// on :latest.
const restateImage = "docker.io/restatedev/restate:1.6.0@sha256:33f227db946864b5482340a8621e32ec5eaf464f4dc41d5deccfd3282bb930ae"

// lazyLiveness lets the test bind BuildSlotService before the Restate
// container exists. The admin port is only known after the container
// starts, and no handler runs before registration completes, so the
// late set is safe.
type lazyLiveness struct {
	mu     sync.Mutex
	client *restateadmin.Client
}

func (l *lazyLiveness) set(c *restateadmin.Client) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.client = c
}

func (l *lazyLiveness) FindLiveInvocations(ctx context.Context, ids []string) (map[string]bool, error) {
	l.mu.Lock()
	c := l.client
	l.mu.Unlock()
	if c == nil {
		return nil, fmt.Errorf("admin client not set")
	}
	return c.FindLiveInvocations(ctx, ids)
}

// slotGateRequest is the input for both SlotGate handlers.
type slotGateRequest struct {
	WorkspaceID  string `json:"workspace_id"`
	DeploymentID string `json:"deployment_id"`
	IsProduction bool   `json:"is_production"`
	WaitSeconds  int    `json:"wait_seconds"`
}

// SlotGate is a test stand-in for the Deploy workflow. It acquires a
// build slot exactly as svc/ctrl/worker/deploy/queue_gate.go does:
// create an awakeable, call AcquireOrWait, then wait for the resolve.
type SlotGate struct {
	db db.Database
}

// acquire mirrors Workflow.waitForBuildSlot with a short, test-owned
// timeout instead of the one-hour production bound.
func (g SlotGate) acquire(ctx restate.Context, req slotGateRequest) (bool, error) {
	awakeable := restate.Awakeable[bool](ctx)

	if _, err := hydrav1.NewBuildSlotServiceClient(ctx, req.WorkspaceID).AcquireOrWait().Request(&hydrav1.AcquireOrWaitRequest{
		DeploymentId: req.DeploymentID,
		AwakeableId:  awakeable.Id(),
		IsProduction: req.IsProduction,
	}); err != nil {
		return false, err
	}

	timeout := restate.After(ctx, time.Duration(req.WaitSeconds)*time.Second)
	winner, err := restate.WaitFirst(ctx, awakeable, timeout)
	if err != nil {
		return false, err
	}
	if winner != awakeable {
		return false, nil
	}
	return awakeable.Result()
}

// Acquire requests a slot and reports whether it was granted within the
// wait bound. The test calls it request-response for deployment B.
func (g SlotGate) Acquire(ctx restate.Context, req slotGateRequest) (bool, error) {
	return g.acquire(ctx, req)
}

// AcquireAndHold requests a slot, marks the deployment as building so
// the test can observe the grant, then parks forever. It simulates a
// Deploy invocation in mid-build. The test kills this invocation, which
// skips all compensations, so Release never runs and the slot entry
// goes stale. This is the production incident.
func (g SlotGate) AcquireAndHold(ctx restate.Context, req slotGateRequest) (restate.Void, error) {
	granted, err := g.acquire(ctx, req)
	if err != nil {
		return restate.Void{}, err
	}
	if !granted {
		return restate.Void{}, restate.TerminalErrorf("slot not granted within %d seconds", req.WaitSeconds)
	}

	if _, err := restate.Run(ctx, func(runCtx restate.RunContext) (restate.Void, error) {
		return restate.Void{}, g.db.UpdateDeploymentStatus(runCtx, db.UpdateDeploymentStatusParams{
			Status:    mysqltype.DeploymentsStatusBuilding,
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			ID:        req.DeploymentID,
		})
	}, restate.WithName("mark deployment building")); err != nil {
		return restate.Void{}, err
	}

	// Park on an awakeable that nothing resolves. The invocation stays
	// live in Restate until the test kills it.
	_, err = restate.Awakeable[bool](ctx).Result()
	return restate.Void{}, err
}

// terminateInvocation calls the Restate 1.6 admin API. Mode is "kill" or
// "purge". Kill ends the invocation without compensations. Purge removes
// the completed invocation row entirely.
func terminateInvocation(t *testing.T, adminURL, invocationID, mode string) {
	t.Helper()

	url := fmt.Sprintf("%s/invocations/%s?mode=%s", adminURL, invocationID, mode)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
}

// TestBuildSlot_ReclaimsSlotOfKilledInvocation reproduces the production
// incident: active_slots holds a deployment whose Deploy invocation no
// longer exists in Restate, and every later deployment queues behind it.
//
// Flow, against real Restate 1.6 and real MySQL:
//
//  1. The workspace limit is one concurrent build.
//  2. Deployment A takes the slot and parks, like a running build.
//  3. The test records A's invocation ID on the deployment row, then
//     kills the invocation. Kill skips compensations, so Release never
//     runs. The slot entry is now stale.
//  4. Deployment B asks for a slot. The acquire-time audit must detect
//     the dead invocation, reclaim the slot, and grant B immediately.
//  5. The killed invocation is purged and the liveness query must still
//     report it dead.
func TestBuildSlot_ReclaimsSlotOfKilledInvocation(t *testing.T) {
	h := New(t)
	ctx := h.Context()
	workspaceID := h.Resources().UserWorkspace.ID

	// Limit the workspace to one concurrent build so one stale slot
	// consumes all capacity, as in the incident.
	require.NoError(t, h.DB.UpsertLimit(ctx, db.UpsertLimitParams{
		WorkspaceID:                           workspaceID,
		ApiBillableOperationsCountMaxPerMonth: 1_000_000,
		ApiRequestsCountMaxPerMinute:          sql.NullInt32{}, //nolint:exhaustruct
		LogsRetentionDaysMax:                  7,
		LogsAuditRetentionDaysMax:             7,
		TeamEnabled:                           false,
		CpuCoresMax:                           10,
		CpuCoresMaxPerInstance:                2,
		MemoryMibMax:                          20_480,
		MemoryMibMaxPerInstance:               4_096,
		StorageMibMax:                         51_200,
		StorageMibMaxPerInstance:              10_240,
		BuildsConcurrentMax:                   1,
		CustomDomainsMax:                      0,
		AutoscalingReplicasMax:                0,
	}))

	liveness := &lazyLiveness{} //nolint:exhaustruct
	slotService := buildslot.New(buildslot.Config{
		DB:           h.DB,
		RestateAdmin: liveness,
	})

	tEnv := restatetest.StartWithOptions(t,
		server.NewRestate().
			Bind(hydrav1.NewBuildSlotServiceServer(slotService)).
			Bind(restate.Reflect(SlotGate{db: h.DB})),
		restatetest.WithRestateImage(restateImage),
	)

	adminURL := fmt.Sprintf("http://localhost:%d", tEnv.AdminPort())
	adminClient := restateadmin.New(restateadmin.Config{BaseURL: adminURL, APIKey: ""})
	liveness.set(adminClient)

	depA := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})

	// Deployment A takes the only slot and parks, like a live build.
	holder, err := ingress.ServiceSend[slotGateRequest](tEnv.Ingress(), "SlotGate", "AcquireAndHold").
		Send(ctx, slotGateRequest{
			WorkspaceID:  workspaceID,
			DeploymentID: depA.ID,
			IsProduction: false,
			WaitSeconds:  60,
		})
	require.NoError(t, err)
	holderInvocationID := holder.Id()
	require.NotEmpty(t, holderInvocationID)

	// The holder sets the status to building after the grant. Wait for it
	// so the kill below cannot race the acquire.
	require.Eventually(t, func() bool {
		dep, findErr := h.DB.FindDeploymentById(ctx, depA.ID)
		return findErr == nil && dep.Status == mysqltype.DeploymentsStatusBuilding
	}, 30*time.Second, 200*time.Millisecond, "deployment A never acquired the slot")

	// Record the invocation on the deployment row, as the Deploy workflow
	// does in production. The audit reads this column.
	require.NoError(t, h.DB.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
		InvocationID: sql.NullString{String: holderInvocationID, Valid: true},
		UpdatedAt:    sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:           depA.ID,
	}))

	// Liveness query against real Restate: the holder is live, an unknown
	// but well-formed ID is not, and the mixed query does not error.
	live, err := adminClient.FindLiveInvocations(ctx, []string{holderInvocationID, "inv_doesnotexist123"})
	require.NoError(t, err)
	require.True(t, live[holderInvocationID], "running invocation must report live")
	require.False(t, live["inv_doesnotexist123"], "unknown invocation must report dead")

	// Kill A's invocation. Kill skips compensations: Release never runs
	// and active_slots still holds deployment A. This is the incident
	// state.
	terminateInvocation(t, adminURL, holderInvocationID, "kill")

	// A killed invocation keeps its sys_invocation row with status
	// 'completed' until purge. The status filter must classify it dead.
	require.Eventually(t, func() bool {
		liveAfterKill, findErr := adminClient.FindLiveInvocations(ctx, []string{holderInvocationID})
		return findErr == nil && !liveAfterKill[holderInvocationID]
	}, 30*time.Second, 200*time.Millisecond, "killed invocation still reports live")

	// Deployment B asks for a slot. The workspace is at capacity with a
	// stale entry. The acquire-time audit must reclaim A's slot and
	// grant B within the wait bound.
	depB := h.CreateDeployment(ctx, CreateDeploymentRequest{
		Region:       "us-east-1",
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	})

	granted, err := ingress.Service[slotGateRequest, bool](tEnv.Ingress(), "SlotGate", "Acquire").
		Request(ctx, slotGateRequest{
			WorkspaceID:  workspaceID,
			DeploymentID: depB.ID,
			IsProduction: false,
			WaitSeconds:  60,
		})
	require.NoError(t, err)
	require.True(t, granted, "deployment B must get the slot after the stale entry is reclaimed")

	// The audit force-fails the reclaimed deployment. Without this write
	// the row stays at building forever and the dashboard shows a phantom
	// build.
	depAAfter, err := h.DB.FindDeploymentById(ctx, depA.ID)
	require.NoError(t, err)
	require.Equal(t, mysqltype.DeploymentsStatusFailed, depAAfter.Status,
		"reclaimed deployment must be marked failed")

	// Purge removes the completed invocation row. The liveness query must
	// still report it dead, not error.
	terminateInvocation(t, adminURL, holderInvocationID, "purge")
	require.Eventually(t, func() bool {
		liveAfterPurge, findErr := adminClient.FindLiveInvocations(ctx, []string{holderInvocationID})
		return findErr == nil && !liveAfterPurge[holderInvocationID]
	}, 30*time.Second, 200*time.Millisecond, "purged invocation must report dead")
}
