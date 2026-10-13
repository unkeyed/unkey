# Kitchensink

This is a diagnostic workload, not a normal product API. It intentionally echoes
environment variables and headers, logs caller input, delays responses, and
returns requested status codes. Do not "fix" those probes by adding authentication,
redaction, or normalization as incidental cleanup. Do not give it real secrets
or sensitive traffic.

The [README](README.md) deliberately constrains probes to the standard library,
stateless request handling, and no cross-probe imports. Keep them independently
readable as integration examples.

Docker builds need repository-root context, not `svc/kitchensink/`. The
`build-local-image` task also pushes to the registry selected by the active
Kubernetes context; it is not a build-only verification command.
