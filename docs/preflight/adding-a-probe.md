# Adding a preflight probe

A probe asserts one property of the deploy pipeline and returns a typed
`core.Result`. Everything else (metrics, logging, tracing, artifact
upload, alert routing) is handled by the Runner. In practice that means
a typical probe is around 30 lines of Go with two imports: the shared
`wait` helpers and `core`.

This doc walks through writing one from scratch. If it feels harder than
the steps below, the abstraction is wrong: tell someone and we fix the
Runner, not the probe.

## Unit-testing a probe

Every probe gets a unit test that runs against the preflight harness.
The harness stands up a full environment (MySQL, ClickHouse, Restate,
Vault, in-process ctrl API, seeded workspace) in under 20 seconds and
tears everything down cleanly when the test ends.

```go
package probes_test

import (
    "context"
    "testing"

    "github.com/unkeyed/unkey/svc/preflight/harness"
    "github.com/unkeyed/unkey/svc/preflight/probes"
)

func TestMyProbe(t *testing.T) {
    h := harness.Start(t, harness.Config{
        SeedPreflightProject: true, // creates a preflight workspace+project+app+env
    })
    env := h.Env()

    res := (&probes.MyProbe{}).Run(context.Background(), env)
    if !res.OK {
        t.Fatalf("probe failed: %v", res.Err)
    }
}
```

Same `core.Env` shape the binary uses in staging/prod; only the
clients differ. That means the probe code does not branch on target.

## Worked example: probe a new metric collector

Setup: suppose we ship a new metric-collector service that scrapes
customer pods and writes billing rows into ClickHouse. We want preflight
to fail if what the collector records does not match what the test app
emitted.

### 1. Test app endpoint

Add one endpoint to `svc/preflight/testapp/main.go`:

```go
http.HandleFunc("POST /emit-metric", func(w http.ResponseWriter, r *http.Request) {
    name := r.URL.Query().Get("name")
    value, _ := strconv.ParseInt(r.URL.Query().Get("value"), 10, 64)
    // send via whatever SDK the collector expects
    collector.Emit(name, value)
})
```

### 2. Probe file

`svc/preflight/probes/metric_collector.go`:

```go
package probes

import (
    "context"
    "fmt"
    "math/rand"
    "strconv"
    "time"

    "github.com/unkeyed/unkey/svc/preflight/core"
    "github.com/unkeyed/unkey/svc/preflight/internal/wait"
)

type MetricCollector struct{}

func (MetricCollector) Name() string { return "metric_collector" }

func (MetricCollector) Run(ctx context.Context, env *core.Env) core.Result {
    value := rand.Int63()
    ts := time.Now().UnixMilli()

    if err := env.TestApp.Emit(ctx, "preflight_synth", value, ts); err != nil {
        return core.Result{OK: false, Err: fmt.Errorf("emit: %w", err)}
    }

    got, err := wait.Poll(ctx, 500*time.Millisecond, func(ctx context.Context) (int64, bool, error) {
        v, err := env.ClickHouse.QueryMetric(ctx, "preflight_synth", ts)
        return v, err == nil, nil
    })
    if err != nil {
        return core.Result{OK: false, Err: err}
    }

    return core.Result{
        OK:   got == value,
        Dims: map[string]string{"value": strconv.FormatInt(got, 10)},
    }
}

func init() { Register(MetricCollector{}) }
```

Notice what is **not** in this file: no Prometheus, no OTEL, no slog, no
S3, no alert fan-out. The Runner does all of that from the returned
`core.Result`.

### 3. Register it

`init()` above adds the probe to the global registry. Then add its name
to the suite you want it to run in (`svc/preflight/flow.go`):

```go
Probes: []string{
    // ... existing probes
    "metric_collector",
},
```

### 4. Regenerate the probe manifest

```
go run ./cmd/preflight/probemanifest --write
```

That updates `svc/preflight/probes/MANIFEST.txt`. CI
(`.github/workflows/preflight-manifest-check.yaml`) fails the PR if you
forget. The manifest is the sole handoff between this repo and the
infra repo, which reads it to verify every probe has a runbook.

### 5. Runbook (lives in the infra repo, required for oncall)

Runbooks live alongside the PrometheusRule yaml in infra because that
yaml carries the `runbook_url` annotation. Concretely:

```
infra/docs/runbooks/preflight/metric_collector.md
```

Template:

```markdown
# metric_collector

## What this probe proves
Emits a synthetic metric from the test app and confirms the collector
service wrote the correct value to ClickHouse within 30s.

## What a failure means
Either the collector is dropping events, the ClickHouse ingest path is
broken, or the emit endpoint on the test app regressed.

## Where to look first
- Collector service logs in the preflight run's region.
- ClickHouse `system.errors` for ingestion failures.
- Test app pod logs for the `POST /emit-metric` handler.

## Escalation
Slack #platform-oncall. Link the preflight run's artifact bundle.
```

Infra CI asserts that every name in `svc/preflight/probes/MANIFEST.txt`
has a matching file under `infra/docs/runbooks/preflight/`. The two
halves stay in sync by virtue of PR-level enforcement on both sides.

### 6. Ship in shadow mode first

In the Helm values for your environment:

```yaml
preflight:
  shadow:
    - metric_collector
```

The probe runs, emits metrics, logs as warnings, but does not count
toward the burn-rate alert. Graduate out of shadow once a 3-day baseline
is clean.

## Rules of thumb

- **One assertion per probe.** If your probe touches two unrelated
  invariants, split it. Shadow-mode graduation happens per probe.
- **Return context-deadline errors verbatim.** Do not wrap
  `context.DeadlineExceeded`; the Runner distinguishes it from assertion
  failures in the alert summary.
- **Artifacts are small.** Kubectl describes, JSON snapshots, the last
  50 lines of relevant logs. Not gigabytes.
- **Dims are labels, not descriptions.** Low cardinality. `{"protocol":
  "h2c"}` is fine; `{"deployment_id": "dep_xyz..."}` is not.
- **If you need a new shared client, put it on `*core.Env`.** Probes
  never open their own.
