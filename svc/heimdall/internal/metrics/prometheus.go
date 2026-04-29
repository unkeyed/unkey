package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CollectionTotal counts collection ticks by result.
	CollectionTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "collection_total",
			Help:      "Total number of collection ticks.",
		},
		[]string{"result"}, // "success", "error"
	)

	// CollectionDuration tracks how long each collection tick takes.
	CollectionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "collection_duration_seconds",
			Help:      "Duration of collection ticks in seconds.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		},
	)

	// KranePods tracks the current number of krane-managed pods seen on this node.
	KranePods = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "krane_pods",
			Help:      "Current number of krane-managed pods on this node.",
		},
	)

	// CgroupReadErrors counts cgroup file read failures.
	CgroupReadErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "cgroup_read_errors_total",
			Help:      "Total number of cgroup file read failures.",
		},
	)

	// LifecycleEmitted counts CRI lifecycle checkpoints successfully buffered.
	// kind is "start" or "stop".
	LifecycleEmitted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "lifecycle_checkpoints_emitted_total",
			Help:      "Total number of CRI lifecycle checkpoints written to the buffer.",
		},
		[]string{"kind"}, // "start", "stop"
	)

	// LifecycleDrops counts CRI lifecycle events we could not emit a
	// checkpoint for. kind is the event type; reason explains the drop so we
	// can distinguish informer races from cgroup-teardown races.
	LifecycleDrops = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "lifecycle_checkpoints_dropped_total",
			Help:      "Total CRI lifecycle events that did not produce a checkpoint. Each drop is a bounded undercharge.",
		},
		[]string{"kind", "reason"}, // kind: "start"|"stop"; reason: "pod_not_found"|"cgroup_read_failed"|"restart_count_unknown"
	)

	// PeriodicSkips counts pods the 5s periodic tick skipped and did NOT
	// write a checkpoint for. Mirrors LifecycleDrops for the periodic
	// loop so we can tell "restart count not yet populated by informer"
	// apart from "pod genuinely missing". Each skip is a bounded
	// undercharge of one tick (currently 5s) for the affected pod.
	PeriodicSkips = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "periodic_skips_total",
			Help:      "Pods skipped by the periodic collection tick (did not produce a checkpoint).",
		},
		[]string{"reason"}, // "restart_count_unknown", "cgroup_read_failed"
	)

	// NetworkAttachFailures counts per-pod network attach failures by
	// reason. Lets us distinguish benign cases (pod already terminated
	// and its sandbox container is gone from containerd) from real bugs
	// (netns open failed, TCX attach rejected by the kernel).
	NetworkAttachFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "network_attach_failures_total",
			Help:      "Per-pod network attach failures, labeled by reason.",
		},
		[]string{"reason"}, // "sandbox_not_found", "netns_open", "veth_lookup", "tcx_attach", "queue_full", "other"
	)

	// BPFMapEntries tracks how many per-pod counter entries are live in
	// the BPF LRU map (`pod_counters`, keyed by POD_KEY = pod netns cookie).
	// If this approaches max_entries (16384) we're about to start evicting,
	// which silently drops traffic from the oldest pods. Alert at 80%.
	BPFMapEntries = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "bpf_map_entries",
			Help:      "Current number of entries in the per-pod BPF byte counter map (keyed by POD_KEY).",
		},
	)

	// (Batch buffer depth is already exposed by pkg/buffer as
	// `unkey_buffer_size{name="instance_checkpoints"}`, no need to duplicate
	// it here. See pkg/buffer/buffer.go for that gauge.)

	// CheckpointsWritten counts rows successfully buffered (which is the
	// last step before the async flush to ClickHouse). This is the
	// primary "is the pipeline working?" signal: if it stops climbing,
	// we're dropping rows regardless of what else the dashboards say.
	CheckpointsWritten = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "checkpoints_written_total",
			Help:      "Total number of checkpoint rows buffered for write to ClickHouse (periodic + lifecycle paths combined).",
		},
	)

	// CollectionTicksSkipped counts ticks dropped because the previous
	// tick still held the collector mutex. Each skip is one lost
	// periodic sample (~5s) for every pod on this node; a non-zero rate
	// means collection is overrunning its interval.
	CollectionTicksSkipped = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "collection_ticks_skipped_total",
			Help:      "Ticks dropped because the previous tick was still running. Non-zero means collection is overrunning its interval.",
		},
	)

	// CRIEventsReceived counts raw containerd task events the CRI watcher
	// observed, before any filtering (sandbox container, missing labels,
	// unresolvable pod). Pair with LifecycleEmitted to spot filtering
	// regressions: if received > emitted by a large margin with no
	// corresponding LifecycleDrops, we're silently discarding events.
	CRIEventsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "cri_events_received_total",
			Help:      "Raw CRI task events received from containerd, pre-filtering.",
		},
		[]string{"kind"}, // "start", "exit"
	)

	// CRIReconnects counts containerd subscription reconnect attempts.
	// Spikes here mean containerd is restarting or the stream is
	// flaking, during which lifecycle checkpoints fall back to the
	// periodic loop (losing up to one interval of final CPU per exit).
	CRIReconnects = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "cri_reconnects_total",
			Help:      "Containerd event-stream reconnect attempts.",
		},
	)

	// DiskReadErrors counts failures reading ephemeral-volume statfs data.
	// readEphemeralUsedBytes silently returns 0 on any error (stat, statfs,
	// bind-mount detection), which is a bounded undercharge but invisible
	// without this counter.
	DiskReadErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "disk_read_errors_total",
			Help:      "Failures reading ephemeral-volume disk usage. Each failure undercharges one tick.",
		},
		[]string{"op"}, // "stat_root", "stat_mount", "statfs"
	)

	// NetworkReadErrors counts per-pod BPF map lookup failures. Read
	// errors currently degrade silently to zeroCounters in the collector;
	// a rising counter means the map or its attachments are in a bad
	// state and pods are being billed zero network bytes.
	NetworkReadErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "network_read_errors_total",
			Help:      "Per-pod network byte-counter lookups that failed. Each failure reports zero network bytes for that tick.",
		},
	)

	// InformerCacheSynced is 1 when the pod informer cache has completed
	// its initial list+watch sync and 0 before that. Collection during
	// the unsync window sees a partial pod set, which undercharges —
	// bounded (first ~second of startup) but worth alerting on if it
	// ever flips back to 0 at runtime (apiserver outage during watch).
	InformerCacheSynced = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "unkey",
			Subsystem: "heimdall",
			Name:      "informer_cache_synced",
			Help:      "1 when the pod informer cache is synced, 0 otherwise.",
		},
	)
)
