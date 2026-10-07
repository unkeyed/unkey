# Krane

- Ctrl supplies desired state; Kubernetes supplies observed state. Reporting a
  desired state back as observed can falsely mark a deployment ready.
- A stream checkpoint is safe only after all preceding events have applied.
  Stop on an apply/delete failure rather than acknowledging past it. Resume
  tokens are opaque CDC checkpoints, not numeric resource versions. See
  [watcher.go](internal/watcher/watcher.go).
- Reconciliation must tolerate repeated application. Full sync and incremental
  delivery overlap, and reconnects can replay changes. A broken stream must not
  disable the full-sync repair path.
- Labels are shared with routing, metering, and cleanup, not just selectors local
  to Krane. Use [pkg/labels](pkg/labels) and trace consumers before changing them.
- Environment-variable ciphertext uses the environment ID as its Vault keyring.
  If ciphertext exists but decryption is unavailable, fail the apply rather than
  deploy without secrets. See [secrets.go](internal/deployment/secrets.go).
