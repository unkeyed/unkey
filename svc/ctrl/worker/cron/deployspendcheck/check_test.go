package deployspendcheck

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/x/mocks"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

func TestCheckWorkspaceSpendConfirmsDrainBeforeNotifying(t *testing.T) {
	for name, failure := range map[string]enforcementResult{
		"incomplete drain": {drained: false},
		"teardown error":   {err: restate.TerminalErrorf("teardown failed")},
	} {
		t.Run(name, func(t *testing.T) {
			f := newEnforcementTest(t)
			req := spendCheckRequest()
			result := make(chan error, 1)
			go func() {
				_, err := f.client.CheckWorkspaceSpend().Request(f.ctx, req)
				result <- err
			}()

			select {
			case operation := <-f.enforcement.started:
				require.Equal(t, "Teardown", operation)
			case <-f.ctx.Done():
				t.Fatal("teardown did not start")
			}
			require.True(t, f.db.suspended.Load(), "new deployments must be blocked before drain")
			require.Empty(t, f.email.Sent(), "compute is still running")
			select {
			case err := <-result:
				t.Fatalf("check returned before teardown completed: %v", err)
			default:
			}

			f.enforcement.results <- failure
			require.Error(t, <-result)
			require.True(t, f.db.suspended.Load())
			require.Empty(t, f.email.Sent())

			req.CurrentlySuspended = true
			req.SpendMicroCents = 250 * deploybilling.MicroCentsPerCent
			req.BudgetCents = 150
			f.enforcement.results <- enforcementResult{drained: true}
			resp, err := f.client.CheckWorkspaceSpend().Request(f.ctx, req)
			require.NoError(t, err)
			require.True(t, resp.GetSuspended())
			sent := f.email.Sent()
			require.Len(t, sent, 1, "the next successful tick must deliver the pending notification")
			require.Equal(t, "compute-budget-stopped", sent[0].TemplateID)
			require.Equal(t, "budget-stopped/ws_enforcement/"+req.GetPeriod()+"/1", sent[0].IdempotencyKey)
			require.Equal(t, "$2", sent[0].Variables["USAGE"])
			require.Equal(t, "$1", sent[0].Variables["BUDGET"])

			f.enforcement.results <- enforcementResult{drained: true}
			_, err = f.client.CheckWorkspaceSpend().Request(f.ctx, req)
			require.NoError(t, err)
			require.Len(t, f.email.Sent(), 1, "ordinary re-enforcement must not notify again")

			f.enforcement.results <- failure
			_, err = f.client.CheckWorkspaceSpend().Request(f.ctx, req)
			require.Error(t, err, "re-enforcement must also reject failed teardown")
			require.Len(t, f.email.Sent(), 1)

			req.Stop = false
			f.enforcement.results <- enforcementResult{}
			_, err = f.client.CheckWorkspaceSpend().Request(f.ctx, req)
			require.NoError(t, err)
			require.False(t, f.db.suspended.Load())

			req.Stop = true
			req.CurrentlySuspended = false
			f.enforcement.results <- enforcementResult{drained: true}
			_, err = f.client.CheckWorkspaceSpend().Request(f.ctx, req)
			require.NoError(t, err)
			sent = f.email.Sent()
			require.Len(t, sent, 2)
			require.Equal(t, "budget-stopped/ws_enforcement/"+req.GetPeriod()+"/2", sent[1].IdempotencyKey)
		})
	}
}

func TestCheckWorkspaceSpendClearsSuspensionOnlyAfterResume(t *testing.T) {
	for _, name := range []string{"budget removed", "stop disabled", "under budget"} {
		t.Run(name, func(t *testing.T) {
			f := newEnforcementTest(t)
			f.db.suspended.Store(true)
			req := spendCheckRequest()
			req.CurrentlySuspended = true
			switch name {
			case "budget removed":
				req.BudgetCents = 0
			case "stop disabled":
				req.Stop = false
			case "under budget":
				req.BudgetCents = 300
				resp, err := f.client.CheckWorkspaceSpend().Request(f.ctx, req)
				require.NoError(t, err)
				require.True(t, resp.GetSuspended(), "one under-budget tick must not resume")
			}

			f.enforcement.results <- enforcementResult{err: restate.TerminalErrorf("resume failed")}
			_, err := f.client.CheckWorkspaceSpend().Request(f.ctx, req)
			require.ErrorContains(t, err, "resume failed")
			require.True(t, f.db.suspended.Load(), "failed resume must leave the suspension flag set")
			require.Empty(t, f.email.Sent())

			f.enforcement.results <- enforcementResult{}
			resp, err := f.client.CheckWorkspaceSpend().Request(f.ctx, req)
			require.NoError(t, err)
			require.False(t, resp.GetSuspended())
			require.False(t, f.db.suspended.Load())
		})
	}
}

func TestSpendCheckWithholdsHeartbeatWhenTeardownFails(t *testing.T) {
	f := newEnforcementTest(t)
	client := hydrav1.NewCronServiceIngressClient(f.restate.IngressClient, time.Now().UTC().Format("2006-01"))
	for _, failure := range []enforcementResult{
		{drained: false},
		{err: restate.TerminalErrorf("teardown failed")},
	} {
		f.enforcement.results <- failure
		resp, err := client.RunDeploySpendCheck().Request(f.ctx, &hydrav1.RunDeploySpendCheckRequest{})
		require.NoError(t, err)
		require.Equal(t, int32(1), resp.GetWorkspacesDispatched())
		require.Zero(t, f.heartbeat.pings.Load())
		require.Empty(t, f.email.Sent())
	}

	f.enforcement.results <- enforcementResult{drained: true}
	_, err := client.RunDeploySpendCheck().Request(f.ctx, &hydrav1.RunDeploySpendCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), f.heartbeat.pings.Load())
	require.Len(t, f.email.Sent(), 1)
}

func TestAwaitEnforcementTimesOutWithoutCancellingChild(t *testing.T) {
	mockCtx := mocks.NewMockContext(t)
	response := mockCtx.EXPECT().MockObjectClient("enforcement", "workspace", "teardown").MockResponseFuture("request")
	timeout := mockCtx.EXPECT().MockAfter(10 * time.Minute)
	wait := mockCtx.EXPECT().MockWaitIter(response, timeout)
	wait.Next().Return(true)
	wait.Value().Return(timeout)
	wait.Err().Return(nil)

	ctx := restate.WithMockContext(mockCtx)
	result, err := awaitEnforcement(ctx, restate.Object[string](ctx, "enforcement", "workspace", "teardown").RequestFuture("request"))
	require.Empty(t, result)
	require.ErrorContains(t, err, "timed out waiting for spend-cap enforcement")
	require.True(t, restate.IsTerminalError(err))
	mockCtx.AssertNotCalled(t, "CancelInvocation")
	response.AssertNotCalled(t, "Response")
}

type enforcementTest struct {
	ctx         context.Context
	db          *spendCheckDB
	enforcement *controlledEnforcement
	email       *email.Capture
	heartbeat   *spendHeartbeat
	restate     containers.RestateConfig
	client      hydrav1.DeploySpendCheckServiceIngressClient
}

func newEnforcementTest(t *testing.T) *enforcementTest {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	database := &spendCheckDB{}
	sender := email.NewCapture()
	check, err := NewCheckHandler(CheckConfig{
		DB: database, Admins: spendAdmins{}, Email: sender, BillingBaseURL: "https://app.unkey.com",
	})
	require.NoError(t, err)
	heartbeat := &spendHeartbeat{}
	orchestrator, err := New(Config{DB: database, Usage: spendUsage{}, Heartbeat: heartbeat})
	require.NoError(t, err)
	enforcement := &controlledEnforcement{
		started: make(chan string, 16), results: make(chan enforcementResult, 1),
	}
	cfg := containers.Restate(t,
		hydrav1.NewDeploySpendCheckServiceServer(check).ConfigureHandler("CheckWorkspaceSpend", RetryPolicy()),
		hydrav1.NewDeployTeardownServiceServer(enforcement),
		hydrav1.NewCronServiceServer(&spendCron{handler: orchestrator}),
	)
	return &enforcementTest{
		ctx: ctx, db: database, enforcement: enforcement, email: sender, heartbeat: heartbeat, restate: cfg,
		client: hydrav1.NewDeploySpendCheckServiceIngressClient(cfg.IngressClient, "ws_enforcement"),
	}
}

func spendCheckRequest() *hydrav1.CheckWorkspaceSpendRequest {
	return &hydrav1.CheckWorkspaceSpendRequest{
		Period: time.Now().UTC().Format("2006-01"), BudgetCents: 100, Stop: true,
		OrgId: "org_test", WorkspaceName: "test", WorkspaceSlug: "test",
		SpendMicroCents: 200 * deploybilling.MicroCentsPerCent,
	}
}

type spendCheckDB struct {
	db.Database
	suspended atomic.Bool
}

func (d *spendCheckDB) SetWorkspaceDeploySpendSuspended(_ context.Context, req db.SetWorkspaceDeploySpendSuspendedParams) error {
	d.suspended.Store(req.Suspended)
	return nil
}

func (*spendCheckDB) FindWorkspaceDeployEntitlement(context.Context, string) (db.FindWorkspaceDeployEntitlementRow, error) {
	return db.FindWorkspaceDeployEntitlementRow{Plan: sql.NullString{String: "pro", Valid: true}}, nil
}

func (d *spendCheckDB) ListWorkspacesWithDeployBudget(context.Context) ([]db.ListWorkspacesWithDeployBudgetRow, error) {
	return []db.ListWorkspacesWithDeployBudgetRow{{
		ID: "ws_enforcement", Name: "test", Slug: "test", OrgID: "org_test",
		SpendBudgetCents: sql.NullInt64{Int64: 100, Valid: true},
		SpendBudgetStop:  sql.NullBool{Bool: true, Valid: true},
		SpendSuspended:   sql.NullBool{Bool: d.suspended.Load(), Valid: true},
	}}, nil
}

type enforcementResult struct {
	drained bool
	err     error
}

type controlledEnforcement struct {
	hydrav1.UnimplementedDeployTeardownServiceServer
	started chan string
	results chan enforcementResult
}

func (c *controlledEnforcement) Teardown(ctx restate.ObjectContext, _ *hydrav1.TeardownRequest) (*hydrav1.TeardownResponse, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (*hydrav1.TeardownResponse, error) {
		c.started <- "Teardown"
		select {
		case result := <-c.results:
			return &hydrav1.TeardownResponse{Drained: result.drained}, result.err
		case <-rc.Done():
			return nil, rc.Err()
		}
	})
}

func (c *controlledEnforcement) Resume(ctx restate.ObjectContext, _ *hydrav1.ResumeRequest) (*hydrav1.ResumeResponse, error) {
	return restate.Run(ctx, func(rc restate.RunContext) (*hydrav1.ResumeResponse, error) {
		c.started <- "Resume"
		select {
		case result := <-c.results:
			return &hydrav1.ResumeResponse{}, result.err
		case <-rc.Done():
			return nil, rc.Err()
		}
	})
}

type spendAdmins struct{}

func (spendAdmins) AdminEmails(context.Context, string) ([]string, error) {
	return []string{"admin@example.com"}, nil
}

type spendHeartbeat struct{ pings atomic.Int32 }

func (h *spendHeartbeat) Ping(context.Context) error {
	h.pings.Add(1)
	return nil
}

type spendUsage struct{}

func (spendUsage) GetInstanceMeterUsage(context.Context, clickhouse.GetInstanceMeterUsageRequest) ([]clickhouse.InstanceMeterUsage, error) {
	return []clickhouse.InstanceMeterUsage{{WorkspaceID: "ws_enforcement", ResourceID: "cpu", CPUSeconds: 1_000_000}}, nil
}

func (spendUsage) GetActiveKeysUsage(context.Context, clickhouse.GetActiveKeysUsageRequest) ([]clickhouse.ActiveKeysUsage, error) {
	return nil, nil
}

type spendCron struct {
	hydrav1.UnimplementedCronServiceServer
	handler *Handler
}

func (c *spendCron) RunDeploySpendCheck(ctx restate.ObjectContext, req *hydrav1.RunDeploySpendCheckRequest) (*hydrav1.RunDeploySpendCheckResponse, error) {
	return c.handler.Handle(ctx, req)
}
