package preflight

// Suite is a named collection of probes invoked in a well-defined
// order. Multiple suites share the probe registry; suite composition
// lives here rather than in the registry so that the solo / mesh /
// stateful split stays loose.
//
// Ordering matters. Two principles, in tension:
//
//  1. Cheap prereqs first. ClickHouse / webhook / RPC checks each
//     finish in seconds and isolate which layer is broken when the
//     real customer flow fails downstream. If they fail on their own,
//     the run aborts cheaply with a clear cause.
//
//  2. Real customer flow before its dependents. git_push (the actual
//     "did a customer push and serve traffic" check) MUST come before
//     request_logs, because request_logs reads the sentinel rows
//     git_push generated. Reversed, request_logs is guaranteed to
//     timeout looking for a row that hasn't been written.
type Suite struct {
	Name   string
	Probes []string // probe names, in run order
}

// DefaultSolo is the single-app-deploy suite that covers tier-1
// through tier-4 items in the plan. This is what the binary runs in
// staging and prod; it includes probes that require real
// infrastructure.
func DefaultSolo() Suite {
	return Suite{
		Name: "solo",
		Probes: []string{
			// --- Cheap prereq checks (~15s combined). Fail fast on
			// obvious infrastructure breakage; isolate which layer
			// regressed when the real flow fails downstream.

			// ClickHouse reachability. Cheapest of all; if CH is
			// down, request_logs (later) is guaranteed to fail and
			// we want a clean signal.
			"clickhouse_connectivity",

			// Webhook ingress + signature verification.
			"github_webhook",

			// Control-plane RPC surface (CreateDeployment + GetDeployment).
			"create_deployment",

			// --- The real customer flow. This is the one that
			// matters most: when it's green, every leg of the
			// pipeline works.
			"git_push",

			// --- Probes that depend on git_push having produced a
			// live deployment. Must run after.

			// Asserts Krane-injected UNKEY_* env vars match the
			// deployment row. Runs before request_logs so a broken
			// pod fails loud here rather than as a missing log row.
			"krane_contract",
			// Reads sentinel_requests_raw_v1 for the row git_push
			// caused sentinel to write.
			"request_logs",

			// --- State-mutating probes. Run last so earlier probes
			// operate on stable pre-mutation state.

			// CreateDeployment + DeployService.Rollback round-trip
			// against the env-sticky hostname. Leaves the environment
			// on whichever deployment was live at probe start, plus
			// one superseded row the nightly GC cleans up.
			"rollback",
		},
	}
}

// DevSuite is for environments that lack the real staging
// infrastructure: no GitHub App, no Depot, no Krane, no real
// frontline pushing requests through sentinel. TestDev uses this
// (and seeds a sentinel row separately so request_logs would work);
// developers running the binary against minikube without wiring the
// dev GitHub App also use this.
//
// Excludes:
//   - git_push (needs the real GitHub App + ngrok + Depot)
//   - request_logs (needs traffic from git_push; without it, the
//     probe times out reading a non-existent row)
//
// TestDev opts request_logs back in by appending it to the suite
// after seeding; see svc/preflight/harness/dev_test.go.
func DevSuite() Suite {
	return Suite{
		Name: "solo-dev",
		Probes: []string{
			"noop",
			"clickhouse_connectivity",
			"github_webhook",
			"create_deployment",
		},
	}
}
