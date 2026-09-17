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
	probeService = "FlowControlProbe"

	// Every wait in the test is bounded by awaitBudget, and the longest chain
	// of waits before a Hold is released stays well under holdTimeout, which
	// in turn stays under Restate's one minute inactivity timeout
	awaitBudget = 10 * time.Second
	holdTimeout = 50 * time.Second

	// waitDuration stays well under Restate's one minute inactivity timeout,
	// so a Wait is never suspended and keeps its slot for the whole sleep
	waitDuration = 5 * time.Second
)

// TestFlowControl pins the Restate behaviour Deploy relies on when it calls
// Build in [restateadmin.BuildConcurrencyScope] with the workspace as limit
// key: a rule caps how many invocations per limit key run at once, cancelling
// the caller removes its queued callee, a callee that waits keeps its slot,
// and a rule written while callers wait lets them run
func TestFlowControl(t *testing.T) {
	ctx := context.Background()
	probe := &FlowControlProbe{
		mu:       sync.Mutex{},
		running:  map[string]map[string]struct{}{},
		started:  map[string]struct{}{},
		gates:    map[string]chan struct{}{},
		released: map[string]struct{}{},
		called:   map[string]struct{}{},
		parents:  map[string]error{},
	}
	cfg := containers.Restate(t, restate.Reflect(probe))
	admin := restateadmin.New(restateadmin.Config{BaseURL: cfg.AdminURL, APIKey: ""})

	err := admin.UpsertRules(ctx, []restateadmin.RuleUpsert{
		{Pattern: restateadmin.BuildConcurrencyScope + "/*", Concurrency: 1, Description: "test default"},
	})
	require.NoError(t, err)

	hold := func(t *testing.T, limitKey string, done chan<- error) {
		t.Helper()
		key := uid.New(uid.DeploymentPrefix)
		go func() {
			_, err := ingress.Workflow[holdRequest, string](cfg.IngressClient, probeService, key, "Hold", restate.WithScope(restateadmin.BuildConcurrencyScope)).
				Request(ctx, holdRequest{Scope: restateadmin.BuildConcurrencyScope, LimitKey: limitKey}, restate.WithLimitKey(limitKey))
			done <- err
		}()
	}

	t.Run("one invocation per limit key at a time", func(t *testing.T) {
		wsA := uid.New(uid.WorkspacePrefix)
		wsB := uid.New(uid.WorkspacePrefix)
		done := make(chan error, 4)
		for range 3 {
			hold(t, wsA, done)
		}
		hold(t, wsB, done)

		probe.awaitRunning(t, wsA, 1)
		probe.awaitRunning(t, wsB, 1)
		require.Never(t, func() bool { return probe.runningCount(wsA) > 1 }, 2*time.Second, 50*time.Millisecond,
			"a second %s invocation ran while the first was still running", wsA)

		probe.release(probe.nextRunning(t, wsB))
		for range 3 {
			key := probe.nextRunning(t, wsA)
			require.Never(t, func() bool { return probe.runningCount(wsA) > 1 }, 500*time.Millisecond, 50*time.Millisecond)
			probe.release(key)
		}

		for range 4 {
			require.NoError(t, <-done)
		}
	})

	t.Run("cancelling the caller removes its queued callee", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		first := uid.New(uid.DeploymentPrefix)
		queued := uid.New(uid.DeploymentPrefix)

		_, err := ingress.WorkflowSend[parentRequest](cfg.IngressClient, probeService, uid.New(uid.DeploymentPrefix), "Run").
			Send(ctx, parentRequest{ChildKey: first, LimitKey: ws})
		require.NoError(t, err)
		probe.awaitRunning(t, ws, 1)

		queuedParent, err := ingress.WorkflowSend[parentRequest](cfg.IngressClient, probeService, uid.New(uid.DeploymentPrefix), "Run").
			Send(ctx, parentRequest{ChildKey: queued, LimitKey: ws})
		require.NoError(t, err)
		require.Eventually(t, func() bool { return probe.hasCalled(queued) }, awaitBudget, 50*time.Millisecond,
			"the second caller never issued its call")
		require.Never(t, func() bool { return probe.hasStarted(queued) }, time.Second, 50*time.Millisecond,
			"the second callee must wait behind the first")

		require.NoError(t, admin.CancelInvocation(ctx, queuedParent.Id()))

		require.Eventually(t, func() bool {
			ok, _ := probe.parentResult(queued)
			return ok
		}, awaitBudget, 50*time.Millisecond, "the cancelled caller never returned")
		_, parentErr := probe.parentResult(queued)
		require.Error(t, parentErr)
		require.True(t, restate.IsTerminalError(parentErr), "cancel must surface as a terminal error, got %v", parentErr)

		probe.release(first)
		require.Eventually(t, func() bool {
			ok, parentErr := probe.parentResult(first)
			return ok && parentErr == nil
		}, awaitBudget, 50*time.Millisecond, "the first caller never completed")
		require.Never(t, func() bool { return probe.hasStarted(queued) }, time.Second, 50*time.Millisecond,
			"the cancelled callee ran once the first one finished")
	})

	t.Run("cancelling the caller does not interrupt a running callee", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		child := uid.New(uid.DeploymentPrefix)

		parent, err := ingress.WorkflowSend[parentRequest](cfg.IngressClient, probeService, uid.New(uid.DeploymentPrefix), "Run").
			Send(ctx, parentRequest{ChildKey: child, LimitKey: ws})
		require.NoError(t, err)
		probe.awaitRunning(t, ws, 1)

		require.NoError(t, admin.CancelInvocation(ctx, parent.Id()))

		require.Eventually(t, func() bool {
			ok, parentErr := probe.parentResult(child)
			return ok && parentErr != nil && restate.IsTerminalError(parentErr)
		}, awaitBudget, 50*time.Millisecond, "the cancelled caller must return a terminal error while its callee still runs")
		require.Never(t, func() bool { return probe.runningCount(ws) == 0 }, 2*time.Second, 50*time.Millisecond,
			"a callee that is already running keeps running until it finishes on its own")

		probe.release(child)
		probe.awaitRunning(t, ws, 0)
	})

	t.Run("a run handler can call its own shared handler in the scope", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		key := uid.New(uid.DeploymentPrefix)

		_, err := ingress.WorkflowSend[parentRequest](cfg.IngressClient, probeService, key, "Run").
			Send(ctx, parentRequest{ChildKey: "", LimitKey: ws})
		require.NoError(t, err)
		probe.awaitRunning(t, ws, 1)
		require.Equal(t, []string{key}, probe.runningKeys(ws), "the scoped callee runs under the caller's own key")

		probe.release(key)
		require.Eventually(t, func() bool {
			ok, parentErr := probe.parentResult(key)
			return ok && parentErr == nil
		}, awaitBudget, 50*time.Millisecond, "the caller never got its callee's result")
	})

	// A slot is held for as long as Restate is running the invocation, waits
	// included, and is freed only when Restate suspends it after about a minute
	// of no activity. Build holds its slot throughout because the depot build is
	// one blocking restate.Run
	t.Run("a callee that waits keeps its slot until Restate suspends it", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		waitKey := uid.New(uid.DeploymentPrefix)
		waitDone := make(chan error, 1)
		go func() {
			_, err := ingress.Workflow[holdRequest, string](cfg.IngressClient, probeService, waitKey, "Wait", restate.WithScope(restateadmin.BuildConcurrencyScope)).
				Request(ctx, holdRequest{Scope: restateadmin.BuildConcurrencyScope, LimitKey: ws}, restate.WithLimitKey(ws))
			waitDone <- err
		}()
		require.Eventually(t, func() bool { return probe.hasStarted(waitKey) }, awaitBudget, 50*time.Millisecond,
			"the waiting callee never started")

		done := make(chan error, 1)
		hold(t, ws, done)
		require.Never(t, func() bool { return probe.runningCount(ws) > 0 }, 2*time.Second, 50*time.Millisecond,
			"a callee ran for %s while another one was still waiting inside its handler", ws)

		probe.awaitRunning(t, ws, 1)
		require.NoError(t, <-waitDone)
		probe.release(probe.nextRunning(t, ws))
		require.NoError(t, <-done)
	})

	t.Run("a rule written while callers wait lets them run", func(t *testing.T) {
		ws := uid.New(uid.WorkspacePrefix)
		done := make(chan error, 3)
		for range 3 {
			hold(t, ws, done)
		}
		probe.awaitRunning(t, ws, 1)
		require.Never(t, func() bool { return probe.runningCount(ws) > 1 }, time.Second, 50*time.Millisecond)

		err := admin.UpsertRules(ctx, []restateadmin.RuleUpsert{
			{Pattern: restateadmin.BuildConcurrencyScope + "/" + ws, Concurrency: 3, Description: "test raise"},
		})
		require.NoError(t, err)
		probe.awaitRunning(t, ws, 3)

		rules, err := admin.ListRules(ctx)
		require.NoError(t, err)
		byPattern := make(map[string]uint32, len(rules))
		for _, rule := range rules {
			byPattern[rule.Pattern] = rule.Concurrency
		}
		require.Equal(t, uint32(1), byPattern[restateadmin.BuildConcurrencyScope+"/*"], "the upsert must not replace the wildcard rule")
		require.Equal(t, uint32(3), byPattern[restateadmin.BuildConcurrencyScope+"/"+ws])

		for _, key := range probe.runningKeys(ws) {
			probe.release(key)
		}
		for range 3 {
			require.NoError(t, <-done)
		}
	})
}

type holdRequest struct {
	Scope    string
	LimitKey string
}

// parentRequest names the callee's workflow key; empty means the caller's own
// key, which is how Deploy calls Build
type parentRequest struct {
	ChildKey string
	LimitKey string
}

// FlowControlProbe is a Restate workflow whose Hold handler keeps running
// until the test releases it, whose Wait handler sleeps inside Restate, and
// whose run handler calls Hold the way Deploy calls Build
type FlowControlProbe struct {
	mu       sync.Mutex
	running  map[string]map[string]struct{}
	started  map[string]struct{}
	gates    map[string]chan struct{}
	released map[string]struct{}
	called   map[string]struct{}
	parents  map[string]error
}

func (p *FlowControlProbe) Run(ctx restate.WorkflowContext, req parentRequest) (string, error) {
	child := req.ChildKey
	if child == "" {
		child = restate.Key(ctx)
	}
	p.mu.Lock()
	p.called[child] = struct{}{}
	p.mu.Unlock()

	_, err := restate.Workflow[string](ctx, probeService, child, "Hold", restate.WithScope(restateadmin.BuildConcurrencyScope)).
		RequestFuture(holdRequest{Scope: restateadmin.BuildConcurrencyScope, LimitKey: req.LimitKey}, restate.WithLimitKey(req.LimitKey)).
		Response()

	p.mu.Lock()
	p.parents[child] = err
	p.mu.Unlock()

	if err != nil {
		return "", err
	}
	return child, nil
}

// Hold refuses a request whose scope or limit key did not reach it, so a
// protocol that drops either fails this test instead of passing it
func (p *FlowControlProbe) Hold(ctx restate.WorkflowSharedContext, req holdRequest) (string, error) {
	if ctx.Request().Scope != req.Scope || ctx.Request().LimitKey != req.LimitKey {
		return "", restate.TerminalErrorf("invoked with scope %q limit key %q, want %q %q",
			ctx.Request().Scope, ctx.Request().LimitKey, req.Scope, req.LimitKey)
	}

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

	select {
	case <-gate:
		return key, nil
	case <-time.After(holdTimeout):
		return "", restate.TerminalErrorf("hold %s was never released", key)
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (p *FlowControlProbe) Wait(ctx restate.WorkflowSharedContext, req holdRequest) (string, error) {
	if ctx.Request().Scope != req.Scope || ctx.Request().LimitKey != req.LimitKey {
		return "", restate.TerminalErrorf("invoked with scope %q limit key %q, want %q %q",
			ctx.Request().Scope, ctx.Request().LimitKey, req.Scope, req.LimitKey)
	}

	key := restate.Key(ctx)
	p.mu.Lock()
	p.started[key] = struct{}{}
	p.mu.Unlock()

	if err := restate.Sleep(ctx, waitDuration); err != nil {
		return "", err
	}
	return key, nil
}

func (p *FlowControlProbe) gateLocked(key string) chan struct{} {
	gate, ok := p.gates[key]
	if !ok {
		gate = make(chan struct{})
		p.gates[key] = gate
	}
	return gate
}

func (p *FlowControlProbe) release(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, done := p.released[key]; done {
		return
	}
	p.released[key] = struct{}{}
	close(p.gateLocked(key))
}

func (p *FlowControlProbe) runningKeys(limitKey string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	keys := make([]string, 0, len(p.running[limitKey]))
	for key := range p.running[limitKey] {
		keys = append(keys, key)
	}
	return keys
}

func (p *FlowControlProbe) runningCount(limitKey string) int {
	return len(p.runningKeys(limitKey))
}

func (p *FlowControlProbe) awaitRunning(t *testing.T, limitKey string, want int) {
	t.Helper()
	require.Eventually(t, func() bool { return p.runningCount(limitKey) == want }, awaitBudget, 50*time.Millisecond,
		"%d invocations should be running for %s, %d are", want, limitKey, p.runningCount(limitKey))
}

// nextRunning waits until exactly one invocation for limitKey is running and
// it is not one the test already released, whose handler may not have
// returned yet
func (p *FlowControlProbe) nextRunning(t *testing.T, limitKey string) string {
	t.Helper()
	var key string
	require.Eventually(t, func() bool {
		keys := p.runningKeys(limitKey)
		if len(keys) != 1 {
			return false
		}
		p.mu.Lock()
		_, done := p.released[keys[0]]
		p.mu.Unlock()
		key = keys[0]
		return !done
	}, awaitBudget, 50*time.Millisecond, "no unreleased invocation is running for %s", limitKey)
	return key
}

func (p *FlowControlProbe) hasStarted(key string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.started[key]
	return ok
}

func (p *FlowControlProbe) hasCalled(childKey string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.called[childKey]
	return ok
}

func (p *FlowControlProbe) parentResult(childKey string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	err, ok := p.parents[childKey]
	return ok, err
}
