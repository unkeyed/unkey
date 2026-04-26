// Package collector reads cgroup v2 counters for each krane-managed pod
// on the node and emits one checkpoint per container per tick. Checkpoints
// carry raw kernel counter values (cpu.stat:usage_usec, memory.current) and
// the allocated PVC size. All billing math is deferred to ClickHouse —
// max(counter) - min(counter) over a window is monotone and replay-safe.
package collector
