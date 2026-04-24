# Probes that require real staging / prod infrastructure

Each probe in this folder depends on something the dev harness
cannot provide:

- a real GitHub App that can push commits
- a real Depot build
- Krane provisioning of a real pod
- a real frontline route
- sentinel serving real traffic

These are staging-only and (after the 72-hour green baseline)
prod-enabled. They are excluded from `DevSuite` in
`svc/preflight/flow.go`, so `go test -run TestDev ./svc/preflight/harness/...`
does not try to run them.

When a probe's real-infra dependency is eventually mockable in the
harness (e.g. we add a fake Depot + Krane shim), it can graduate to
`common/` by moving its package and including it in `DevSuite`.

New probes are registered via their package's `init()` and blank-
imported from `svc/preflight/registered.go`.
