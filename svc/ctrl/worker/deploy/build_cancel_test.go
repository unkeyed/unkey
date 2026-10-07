package deploy

import (
	"context"
	"sync"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	buildCancelProbeService = "BuildCancelProbe"

	// callerSuspendWait outlasts Restate's default one minute inactivity
	// timeout, so the caller is suspended before the test cancels it. Under
	// that boundary a cancel reaches the callee whatever its own timeout is,
	// and the test would pass without proving anything
	callerSuspendWait = 75 * time.Second
)

// TestCancelAbortsRunningBuild pins what [BuildKeepAliveWindow] buys: a
// suspended Build cannot be told it was cancelled, so without the window it
// would finish its image build and hold the workspace's build slot until it
// did. Fails if the window is removed or drops below the build's own bound.
func TestCancelAbortsRunningBuild(t *testing.T) {
	ctx := context.Background()
	probe := &BuildCancelProbe{
		mu:      sync.Mutex{},
		running: map[string]map[string]struct{}{},
		started: map[string]struct{}{},
		gates:   map[string]chan struct{}{},
	}
	// Only the callee carries the raised timeout, the way run.go binds it to
	// Build alone
	cfg := containers.Restate(t, restate.Reflect(probe).
		ConfigureHandler("Hold", restate.WithInactivityTimeout(10*time.Minute)))
	admin := restateadmin.New(restateadmin.Config{BaseURL: cfg.AdminURL, APIKey: ""})
	require.NoError(t, admin.UpsertRules(ctx, []restateadmin.RuleUpsert{
		{Pattern: restateadmin.BuildConcurrencyScope + "/*", Concurrency: 1, Description: "test default"},
	}))

	workspaceID := uid.New(uid.WorkspacePrefix)
	running := uid.New(uid.DeploymentPrefix)
	queued := uid.New(uid.DeploymentPrefix)

	caller, err := ingress.WorkflowSend[buildCancelRequest](cfg.IngressClient, buildCancelProbeService, uid.New(uid.DeploymentPrefix), "Run").
		Send(ctx, buildCancelRequest{CalleeKey: running, LimitKey: workspaceID})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return probe.runningCount(workspaceID) == 1 }, 20*time.Second, 100*time.Millisecond,
		"the first callee never started")

	_, err = ingress.WorkflowSend[buildCancelRequest](cfg.IngressClient, buildCancelProbeService, uid.New(uid.DeploymentPrefix), "Run").
		Send(ctx, buildCancelRequest{CalleeKey: queued, LimitKey: workspaceID})
	require.NoError(t, err)
	require.Never(t, func() bool { return probe.hasStarted(queued) }, 3*time.Second, 100*time.Millisecond,
		"the second callee must wait behind the first")

	time.Sleep(callerSuspendWait)
	require.NoError(t, admin.CancelInvocation(ctx, caller.Id()))

	require.Eventually(t, func() bool { return probe.hasStarted(queued) }, 30*time.Second, 200*time.Millisecond,
		"cancelling the caller did not free the slot, so a cancelled build would keep the workspace at its limit")

	probe.release(queued)
}

type buildCancelRequest struct {
	CalleeKey string
	LimitKey  string
}

// BuildCancelProbe is shaped like Deploy calling Build: Run requests Hold in
// the build scope, and Hold blocks inside a restate.Run the way the image
// build does
type BuildCancelProbe struct {
	mu      sync.Mutex
	running map[string]map[string]struct{}
	started map[string]struct{}
	gates   map[string]chan struct{}
}

func (p *BuildCancelProbe) Run(ctx restate.WorkflowContext, req buildCancelRequest) (string, error) {
	_, err := restate.Workflow[string](ctx, buildCancelProbeService, req.CalleeKey, "Hold", restate.WithScope(restateadmin.BuildConcurrencyScope)).
		RequestFuture(req, restate.WithLimitKey(req.LimitKey)).
		Response()
	if err != nil {
		return "", err
	}
	return req.CalleeKey, nil
}

func (p *BuildCancelProbe) Hold(ctx restate.WorkflowSharedContext, req buildCancelRequest) (string, error) {
	key := restate.Key(ctx)
	p.mu.Lock()
	if p.running[req.LimitKey] == nil {
		p.running[req.LimitKey] = map[string]struct{}{}
	}
	p.running[req.LimitKey][key] = struct{}{}
	p.started[key] = struct{}{}
	gate := p.gateLocked(key)
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.running[req.LimitKey], key)
		p.mu.Unlock()
	}()

	return restate.Run(ctx, func(runCtx restate.RunContext) (string, error) {
		select {
		case <-gate:
			return key, nil
		case <-runCtx.Done():
			return "", runCtx.Err()
		case <-time.After(4 * time.Minute):
			return "", restate.TerminalErrorf("hold %s was never released", key)
		}
	}, restate.WithName("hold"))
}

func (p *BuildCancelProbe) gateLocked(key string) chan struct{} {
	gate, ok := p.gates[key]
	if !ok {
		gate = make(chan struct{})
		p.gates[key] = gate
	}
	return gate
}

func (p *BuildCancelProbe) release(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	select {
	case <-p.gateLocked(key):
	default:
		close(p.gateLocked(key))
	}
}

func (p *BuildCancelProbe) runningCount(limitKey string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.running[limitKey])
}

func (p *BuildCancelProbe) hasStarted(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.started[key]
	return ok
}
