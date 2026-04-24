# Probes that run in every target

Each probe in this folder can run against:

- the in-process dev harness (`go test -run TestDev ./svc/preflight/harness/...`)
- staging (via the CronJob deployed by the preflight Helm chart)
- prod (once the staging 72-hour green baseline is satisfied)

What makes a probe "common":

- It asserts through the ctrl API surface and/or harness-seedable
  state only.
- It does not require a real Depot build, Krane provisioning,
  frontline route, or sentinel-in-the-traffic-path.
- Its unit test runs under the harness in under a couple of minutes.

When a new probe wants to land here, check the list above. If any is
a `no`, it belongs under `realinfra/` instead.

New probes are registered via their package's `init()` and blank-
imported from `svc/preflight/registered.go`.
