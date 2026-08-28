# Control Plane Worker Guide

The ctrl worker (`svc/ctrl/worker/`) executes durable workflows and periodic background tasks via Restate. It handles deployments, builds, certificate management, and scheduled cron jobs.

## Architecture

```
Restate Runtime ←→ Worker (HTTP server on port 9080)
                      │
                      ├── Deploy Service (deployment workflows)
                      ├── Cron Service (periodic background tasks)
                      ├── Certificate Service (TLS cert management)
                      └── GitHub Webhook Service (push-triggered deploys)
```

The worker registers itself with Restate's admin API on startup. Restate invokes handlers via HTTP with journaling for exactly-once semantics.

## Configuration (`svc/ctrl/worker/config.go`)

Loaded from TOML via `config.Load[worker.Config]("/path/to/config.toml")`. Key sections:

```toml
[restate]
admin_url = "http://restate:9070"
http_port = 9080
register_as = "http://worker:9080"

[depot]
api_url = "https://api.depot.dev"
project_region = "us-east-1"
project_prefix = "builds-local"

[registry]
repository = "registry.depot.dev/..."
username = "x-token"
password = "..."

[billing]
stripe_secret_key = ""  # Empty disables billing push

[heartbeat]
quota_check_url = ""
key_refill_url = ""
deploy_billing_push_url = ""
# ...
```

## Startup Flow (`svc/ctrl/worker/run.go`)

1. Parse config, validate
2. Connect to MySQL, ClickHouse, Vault, Restate
3. Construct service instances (Deploy, Cron, Cert, GitHub)
4. Register Restate services
5. Start HTTP server
6. Self-register with Restate admin

## Cron Service (`svc/ctrl/worker/cron/`)

All scheduled tasks are handlers on a single `hydra.v1.CronService` Restate virtual object. The service shim delegates each proto-generated `RunX` method to the corresponding handler.

### Structure

```
svc/ctrl/worker/cron/
├── cron.go                 # Service struct, New(), delegation methods
├── auditlogcleanup/        # Audit log outbox cleanup
├── auditlogexport/         # MySQL outbox → ClickHouse drain
├── deploybilling/          # Hourly Deploy billing push
├── keylastusedsync/        # Key last-used timestamp sync
├── keyrefill/              # Key credit refill
├── quotacheck/             # Workspace quota enforcement
└── ratelimitcleanup/       # Expired rate limit counter cleanup
```

### Adding a New Cron Task

1. Create a subpackage under `svc/ctrl/worker/cron/` with a `Handler` struct
2. Implement `Handle(ctx restate.ObjectContext, req *proto.RunXRequest) (*proto.RunXResponse, error)`
3. Add the proto message to `svc/ctrl/proto/hydra/v1/cron.proto`
4. Run `mise run generate` to regenerate protobuf
5. Add a field on `cron.Service` for the new handler
6. Wire it in `cron.New()` (add to Config, construct, assign)
7. Add a one-line delegating `RunX` method on Service
8. Add a heartbeat URL field to `HeartbeatConfig` and `cron.Heartbeats`
9. Register the schedule in `dev/k8s/charts/restate-cronjobs/values.yaml`
10. Run `mise run bazel` to sync BUILD files

### Handler Pattern

```go
package mytask

import (
    restate "github.com/restatedev/sdk-go"
    hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
    "github.com/unkeyed/unkey/pkg/healthcheck"
)

type Config struct {
    DB        db.Database
    Heartbeat healthcheck.Heartbeat
}

type Handler struct {
    db        db.Database
    heartbeat healthcheck.Heartbeat
}

func New(cfg Config) (*Handler, error) {
    // Validate config with assert.All(...)
    return &Handler{db: cfg.DB, heartbeat: cfg.Heartbeat}, nil
}

func (h *Handler) Handle(
    ctx restate.ObjectContext,
    req *hydrav1.RunMyTaskRequest,
) (*hydrav1.RunMyTaskResponse, error) {
    // The object key (ctx.Key()) is typically "YYYY-MM" for monthly tasks
    // or a fixed string for fixed-interval tasks.

    // Do work...

    // Ping heartbeat on success
    h.heartbeat.Ping(ctx)

    return &hydrav1.RunMyTaskResponse{}, nil
}
```

### Key Patterns

- **Object key = serialization scope**: Tasks keyed by `"YYYY-MM"` serialize within a month but allow concurrent months. Fixed-interval tasks use a constant key.
- **Idempotency**: Design handlers to be safe on retry/replay. Use absolute values (not deltas), upserts, and deduplication.
- **Heartbeat on success**: Every handler pings its healthcheck heartbeat after successful completion. Use `healthcheck.NewNoop()` when monitoring is not configured.
- **No-op when unconfigured**: Handlers degrade gracefully when optional dependencies (ClickHouse, Stripe) are absent.

## Deploy Service (`svc/ctrl/worker/deploy/`)

Handles the deployment lifecycle: build → push → deploy → promote.

### Key Files

- `deploy_handler.go`: Main deployment orchestration workflow
- `build.go`: Shared build scaffold (Depot project resolution, GitHub token, env var decryption, retry)
- `railpack.go`: Automatic (Dockerfile-less) builds via Railpack
- `service.go`: Service struct and dependencies
- `scale_down_idle_preview_deployments.go`: Idle preview cleanup

### Build Flow

```
CreateDeployment request
  → Determine build method (Dockerfile vs Railpack)
  → Resolve Depot project (create if needed)
  → Obtain GitHub token (or allow unauthenticated in dev)
  → Decrypt environment variables
  → Execute build (Docker or Railpack two-phase)
  → Push image to registry
  → Update deployment status
  → Notify Krane for StatefulSet creation
  → Update frontline routes (promote)
```

### Railpack (Automatic Builds)

When `app_build_settings.dockerfile` is NULL:

1. **Plan phase**: Generate a Dockerfile that runs `railpack prepare` inside the Railpack builder image against a git context URL. Exports only the plan JSON.
2. **Build phase**: Feed the plan to the Railpack BuildKit frontend with the same git context. Pushes the final image.

Repository content never touches the worker. Both phases use git context URLs.

### Durable Steps

Use `restate.Run()` for side effects that should be journaled:

```go
result, err := restate.Run(ctx, func(ctx restate.RunContext) (MyResult, error) {
    // This runs at-most-once. If the worker crashes after completion,
    // Restate replays the result from the journal.
    return doSomething()
})
```

## Virtual Object Keys

Restate virtual objects serialize operations by key. Choose keys to prevent conflicting mutations:

| Resource | Key Pattern | Purpose |
|----------|-------------|---------|
| Deployment | `deployment_id` | One build/deploy at a time per deployment |
| Project | `project_id` | Serialize project-level operations |
| Domain | `domain_name` | One cert operation per domain |
| Billing period | `"YYYY-MM"` | Serialize monthly billing pushes |
| Workspace | `workspace_id` | Serialize per-workspace operations |

## Error Handling

- Use `pkg/fault` for all errors with context
- Terminal errors (bad config, invalid input): return error directly (Restate won't retry)
- Transient errors (network, 5xx): wrap with retry hint so Restate retries automatically
- Use `assert.All()` for config validation at construction time

## Testing

- Integration tests in `svc/ctrl/integration/` with a full harness
- Unit tests colocated with handlers (e.g., `billing_test.go`)
- Use `pkg/billingperiod` for period parsing in tests
- Mock external services (Stripe, Depot) for unit tests

## Scheduling

Cron schedules are configured in `dev/k8s/charts/restate-cronjobs/values.yaml`. Each entry specifies:
- Service name and handler
- Cron expression
- Object key (static or template)

The Restate cron job controller invokes the handler at the specified interval with the configured key.
