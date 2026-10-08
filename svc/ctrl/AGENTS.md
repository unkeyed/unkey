# Control plane

- [internal/auth/auth.go](internal/auth/auth.go) validates a shared service bearer
  token, not an end user's resource permissions. Do not treat it as a replacement
  for authorization at the public API boundary.
- Deployment admission runs in [worker/deploy/create.go](worker/deploy/create.go).
  `Deploy` and `Build` are ingress-private in [worker/run.go](worker/run.go)
  because direct calls would bypass admission checks. Preserve that boundary
  when changing deployment entry points.
- The deployment stream reloads the latest state when a CDC event arrives; it
  does not replay historical desired state. Send each checkpoint only after its
  preceding deployment events. Krane uses that ordering to resume safely. See
  [rpc_watch_deployment_changes.go](services/cluster/rpc_watch_deployment_changes.go).
