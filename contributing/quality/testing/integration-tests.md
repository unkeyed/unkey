---
title: "Integration tests"
description: "Testing components with real dependencies"
notion:
  owners:
    - andreas
  tags:
    - Quality
    - Testing
---

## When to use integration tests

Use integration tests to cover behavior across service boundaries, real databases, caches, and storage. These tests catch failures that mocks miss.

Integration tests must document the cross-boundary guarantee they protect. Make it clear which production contract would be broken if the test failed, such as transaction rollback, idempotency, permission propagation, cache invalidation, or persistence after restart.

```go
// TestCreateKeyRollsBackAuditLogOnDatabaseFailure guarantees that key creation
// and audit logging remain atomic. Operators must not see an audit log for a
// key that was never committed.
func TestCreateKeyRollsBackAuditLogOnDatabaseFailure(t *testing.T) {
    // ...
}
```

## Container patterns

Use `pkg/testutil/containers` for shared test containers. Helpers lazily start
containers through `pkg/testutil/docker-compose.test.yaml` and reuse one Docker
Compose project per worktree, so tests only start the services they need.
`mise run test` removes the shared containers for that worktree after the test
suite exits.

Restate is the exception to container reuse. `containers.Restate` starts a
server per test and removes it afterwards, because Restate identifies services
by name and those names come from protobuf packages. Two tests registering
their own workers on one server would overwrite each other's routing and share
virtual object state.

A private Restate costs about a second to start. Since Go runs a package's
tests sequentially, at most one such container per test binary is alive at a
time.

To inspect a failure, set `UNKEY_TEST_KEEP_RESTATE=1`. The container of a
failed test is left running and its admin URL is logged, so the invocation
journal and state stay queryable:

```bash
UNKEY_TEST_KEEP_RESTATE=1 mise exec -- rask ./svc/ctrl/worker/cron
```

Containers left behind by a test process that was killed are removed by the
next run.

For full-suite runs, use `mise run test`. Direct `rask` does not run the
cleanup trap from the mise task.

```go
redisURL := containers.Redis(t)
```

## Test harness

API route tests use `svc/api/internal/testutil` for a service graph and seeded
data. This is an API-internal package, not a shared harness in `pkg/testutil`.
Tests outside `svc/api` must use their service's test setup and the shared
`pkg/testutil/containers` helpers rather than importing an API-internal package.

```go
h := testutil.NewHarness(t)
workspace := h.CreateWorkspace()
```

The default harness shares MySQL. Fixture rows live until the suite's container
teardown, not until the test ends. For database-wide queries or exact global
lists, use `testutil.NewHarness(t, testutil.HarnessConfig{IsolatedMySQL: true})`.
Unique IDs prevent collisions but do not hide sibling tests' rows.

For a project, app, environment, and their default settings, use
`h.CreateEnvironmentTopology(workspace.ID, mysqltype.EnvironmentKindProduction)`.
Pass the workspace and environment kind explicitly. This fixture creates no
credentials or billing entitlements; keep permission grants in the test.

Keep harness setup explicit when it affects the guarantee. Do not hide permissions, feature flags, tenants, or seeded records behind generic helpers unless the helper name or docstring explains the resulting system state.

Use fresh [test evidence](./index.md#fresh-reliability-and-named-witnesses) when
the result depends on external state. A cached package pass cannot establish
that a database migration, service version, or persisted state still works.

## Suite cost

Integration tests that start containers must use the shared container helpers.
Keep expensive setup explicit so readers can see why the test belongs in the
integration suite.

## Debugging failures

Use verbose output for failing tests:

```bash
mise exec -- rask -v ./pkg/vault/integration
```
