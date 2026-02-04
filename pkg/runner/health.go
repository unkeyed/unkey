package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultCheckTimeout = 500 * time.Millisecond
)

// ReadinessCheck is a named health check function.
// It should return nil if healthy, or an error describing the failure.
// The context has a timeout; implementations should respect cancellation.
type ReadinessCheck func(ctx context.Context) error

// healthState holds the health-related state for the runner.
type healthState struct {
	started      atomic.Bool
	shuttingDown atomic.Bool

	mu           sync.RWMutex
	checks       map[string]ReadinessCheck
	checkTimeout time.Duration
}

func newHealthState() *healthState {
	return &healthState{
		started:      atomic.Bool{},
		shuttingDown: atomic.Bool{},
		mu:           sync.RWMutex{},
		checks:       make(map[string]ReadinessCheck),
		checkTimeout: defaultCheckTimeout,
	}
}

// checkResult holds the result of a single readiness check.
type checkResult struct {
	name string
	err  error
}

// healthResponse is the JSON response for health endpoints.
type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// RegisterHealth registers health check endpoints on the provided mux.
//
// Endpoints:
//   - GET <prefix>/live    - liveness probe (is the process alive?)
//   - GET <prefix>/ready   - readiness probe (can it receive traffic?)
//   - GET <prefix>/startup - startup probe (has initialization completed?)
//
// The default prefix is "/health". Pass a custom prefix as the second argument.
// Note: Uses Go 1.22+ ServeMux method patterns ("GET /path").
// The started state is automatically set when Runner.Wait() is called.
func (r *Runner) RegisterHealth(mux *http.ServeMux, prefix ...string) {
	base := "/health"
	if len(prefix) > 0 {
		base = strings.TrimRight(prefix[0], "/")
	}
	r.logger.Info("registering health endpoints", "base", base)
	mux.HandleFunc(fmt.Sprintf("GET %s/live", base), r.handleLive)
	mux.HandleFunc(fmt.Sprintf("GET %s/ready", base), r.handleReady)
	mux.HandleFunc(fmt.Sprintf("GET %s/startup", base), r.handleStartup)
}

// AddReadinessCheck registers a named readiness check.
// All checks run in parallel when /health/ready is called.
// Each check has a timeout (default 500ms, configurable via WithReadinessTimeout).
//
// Panics if name is empty or check is nil.
// If a check with the same name already exists, it is replaced.
func (r *Runner) AddReadinessCheck(name string, check ReadinessCheck) {
	if name == "" {
		panic("runner: readiness check name cannot be empty")
	}
	if check == nil {
		panic("runner: readiness check function cannot be nil")
	}
	r.health.mu.Lock()
	defer r.health.mu.Unlock()
	r.health.checks[name] = check
}

// WithReadinessTimeout sets the timeout for individual readiness checks.
// Default is 500ms. This should be less than the Kubernetes probe timeoutSeconds.
func WithReadinessTimeout(d time.Duration) RunOption {
	return func(c *RunConfig) {
		c.ReadinessTimeout = d
	}
}

// handleLive handles the liveness probe.
// Returns 200 if the process has started and is not stuck.
// Returns 503 if startup hasn't completed (blocked by startup probe).
func (r *Runner) handleLive(w http.ResponseWriter, _ *http.Request) {
	if !r.health.started.Load() {
		writeHealthResponse(w, http.StatusServiceUnavailable, healthResponse{Status: "not started", Checks: nil})
		return
	}
	writeHealthResponse(w, http.StatusOK, healthResponse{Status: "ok", Checks: nil})
}

// handleReady handles the readiness probe.
// Returns 200 if:
//   - Not shutting down
//   - All readiness checks pass
//
// Returns 503 otherwise.
func (r *Runner) handleReady(w http.ResponseWriter, req *http.Request) {
	if !r.health.started.Load() {
		writeHealthResponse(w, http.StatusServiceUnavailable, healthResponse{Status: "not started", Checks: nil})
		return
	}

	if r.health.shuttingDown.Load() {
		writeHealthResponse(w, http.StatusServiceUnavailable, healthResponse{Status: "shutting down", Checks: nil})
		return
	}

	r.health.mu.RLock()
	checks := make(map[string]ReadinessCheck, len(r.health.checks))
	for name, check := range r.health.checks {
		checks[name] = check
	}
	r.health.mu.RUnlock()

	if len(checks) == 0 {
		writeHealthResponse(w, http.StatusOK, healthResponse{Status: "ok", Checks: nil})
		return
	}

	results := r.runChecks(req.Context(), checks)

	response := healthResponse{
		Status: "ok",
		Checks: make(map[string]string, len(results)),
	}

	allPassed := true
	for _, result := range results {
		if result.err != nil {
			response.Checks[result.name] = result.err.Error()
			allPassed = false
		} else {
			response.Checks[result.name] = "ok"
		}
	}

	if !allPassed {
		response.Status = "fail"
		writeHealthResponse(w, http.StatusServiceUnavailable, response)
		return
	}

	writeHealthResponse(w, http.StatusOK, response)
}

// handleStartup handles the startup probe.
// Returns 200 if startup has completed, 503 otherwise.
func (r *Runner) handleStartup(w http.ResponseWriter, _ *http.Request) {
	if !r.health.started.Load() {
		writeHealthResponse(w, http.StatusServiceUnavailable, healthResponse{Status: "not started", Checks: nil})
		return
	}
	writeHealthResponse(w, http.StatusOK, healthResponse{Status: "ok", Checks: nil})
}

// runChecks executes all readiness checks in parallel with a timeout.
func (r *Runner) runChecks(ctx context.Context, checks map[string]ReadinessCheck) []checkResult {
	results := make([]checkResult, 0, len(checks))
	resultCh := make(chan checkResult, len(checks))
	timeout := r.health.checkTimeout

	for name, check := range checks {
		go func(name string, check ReadinessCheck) {
			checkCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			err := check(checkCtx)
			resultCh <- checkResult{name: name, err: err}
		}(name, check)
	}

	for range len(checks) {
		results = append(results, <-resultCh)
	}

	return results
}

func writeHealthResponse(w http.ResponseWriter, status int, response healthResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
