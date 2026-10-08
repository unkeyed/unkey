# Heimdall

- Emit cumulative CPU and network counters, not interval deltas. Consumers use
  counter ranges, and duplicate lifecycle samples must not double-count usage.
  Memory and disk-used are gauges, not counters.
- Unknown restart counts cannot default to zero: that merges different container
  lifetimes into one billing series. `container_uid` includes the restart count.
  See [collector/collect.go](internal/collector/collect.go).
- `ErrNotAttached` means unmeasured, not zero. Network counters survive collector
  restarts in a pinned map; emitting zero before reading a retained counter can
  bill old traffic again. See [network_linux.go](internal/network/network_linux.go).
- An async attach completion must not restore stale state after detach or a new
  attach request. The request-generation check in `network_linux.go` guards this.
- The pinned BPF map survives deployments. Changes to its key/value layout or
  capacity need a compatible migration or a new pin generation, not just new Go
  bindings. Read the `bpfPinDir` rationale in `network_linux.go`.
- Checkpoint time uses a monotonic clock to avoid wall-clock corrections changing
  measured usage intervals. Preserve this property when changing clock injection.

Regenerate BPF changes with `mise run generate-bpf` from the repository root.
Unprivileged unit tests do not validate kernel loading, TCX attachment, or pinned
map reuse; those checks need a compatible privileged Linux environment.
