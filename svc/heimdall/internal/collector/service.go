package collector

import (
	"context"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/heimdall/internal/checkpoint"
	"github.com/unkeyed/unkey/svc/heimdall/internal/metrics"
	"github.com/unkeyed/unkey/svc/heimdall/internal/network"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	LabelManagedBy  = "app.kubernetes.io/managed-by"
	LabelComponent  = "app.kubernetes.io/component"
	LabelWorkspace  = "unkey.com/workspace.id"
	LabelProject    = "unkey.com/project.id"
	LabelEnv        = "unkey.com/environment.id"
	LabelDeployment = "unkey.com/deployment.id"
	LabelSentinel   = "unkey.com/sentinel.id"
)

type Config struct {
	CH         *batch.BatchProcessor[schema.InstanceCheckpoint]
	PodLister  corelisters.PodLister
	CgroupRoot string
	// CgroupDriver controls pod-cgroup path shape. Must match kubelet's
	// cgroupDriver. Detect via Preflight and pass the result through.
	CgroupDriver CgroupDriver
	// KubeletRoot is the hostPath where /var/lib/kubelet is mounted. Used to
	// statfs ephemeral volume mount points for disk_used_bytes. Empty
	// disables used-bytes reporting (checkpoints carry 0).
	KubeletRoot string
	NodeName    string
	Region      string
	Platform    string
	// CRISocket is the containerd socket path. If empty, CRI watching is disabled
	// and the collector relies solely on the periodic loop (up to
	// CollectionInterval of lost CPU on each container exit).
	CRISocket string
	// Clock drives checkpoint timestamps. Must be monotonic (see
	// clock.NewMonotonic) — wall-clock backward jumps from NTP corrections
	// would produce negative intervals in the memory pair-integration.
	Clock clock.Clock
	// Network is the eBPF byte-counter reader. Must be non-nil. On non-Linux
	// platforms it's a no-op stub. On Linux it owns the loaded eBPF program
	// and the per-pod cgroup attach links.
	Network network.Reader
}

// Collector reads cgroup v2 counters for krane-managed pods and emits
// checkpoints to ClickHouse on a fixed cadence. Billing math is deferred
// to query time — collector output is raw counter values only.
type Collector struct {
	ch          *batch.BatchProcessor[schema.InstanceCheckpoint]
	podLister   corelisters.PodLister
	cgroup      *cgroupReader
	network     network.Reader
	kubeletRoot string
	nodeName    string
	region      string
	platform    string
	criSocket   string
	clk         clock.Clock
	mu          sync.Mutex
}

func New(cfg Config) *Collector {
	root := cfg.CgroupRoot
	if root == "" {
		root = "/sys/fs/cgroup"
	}

	clk := cfg.Clock
	if clk == nil {
		clk = clock.NewMonotonic()
	}

	return &Collector{
		ch:          cfg.CH,
		podLister:   cfg.PodLister,
		cgroup:      &cgroupReader{root: root, driver: cfg.CgroupDriver},
		network:     cfg.Network,
		kubeletRoot: cfg.KubeletRoot,
		nodeName:    cfg.NodeName,
		region:      cfg.Region,
		platform:    cfg.Platform,
		criSocket:   cfg.CRISocket,
		clk:         clk,
		mu:          sync.Mutex{},
	}
}

func (c *Collector) Run(ctx context.Context, interval time.Duration) error {
	stop := repeat.Every(interval, func() {
		c.collectWithMetrics(ctx)
	})
	defer stop()

	// Track the CRI goroutine so shutdown can wait for it before returning.
	// Without this wait, a late-arriving TaskExit event could fire its handler
	// while run.go's deferred checkpointBuffer.Close() was already running,
	// silently dropping the final exit checkpoint.
	var watcherWG sync.WaitGroup

	// CRI watcher is best-effort: if the socket isn't available, fall back
	// to periodic-only collection. Logging a warning preserves visibility.
	if c.criSocket != "" {
		watcher, err := newCRIWatcher(c.criSocket, c)
		if err != nil {
			logger.Warn("cri watcher disabled, falling back to periodic only", "error", err.Error())
		} else {
			defer func() {
				if cerr := watcher.Close(); cerr != nil {
					logger.Warn("cri watcher close failed", "error", cerr.Error())
				}
			}()
			watcherWG.Add(1)
			go func() {
				defer watcherWG.Done()
				if err := watcher.Run(ctx); err != nil && ctx.Err() == nil {
					logger.Error("cri watcher stopped", "error", err.Error())
				}
			}()
		}
	}

	<-ctx.Done()

	// Wait for CRI handler goroutine to finish before returning. The
	// periodic repeat.Every also needs to drain — `stop()` via defer blocks
	// until any in-flight tick completes. Combined, Run doesn't return until
	// all in-flight handlers have had a chance to Buffer() their rows.
	watcherWG.Wait()

	return ctx.Err()
}

func (c *Collector) collectWithMetrics(ctx context.Context) {
	start := time.Now()

	if !c.mu.TryLock() {
		metrics.CollectionTicksSkipped.Inc()
		logger.Info("skipping collection, previous tick still running")
		return
	}
	defer c.mu.Unlock()

	err := c.collect(ctx)

	metrics.CollectionDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.CollectionTotal.WithLabelValues("error").Inc()
		logger.Error("collection failed", "error", err.Error())
	} else {
		metrics.CollectionTotal.WithLabelValues("success").Inc()
	}
}

// OnStart is called by the CRI watcher when a container task starts. We emit
// a baseline checkpoint so max-min can capture full-lifetime CPU even for
// containers that only produce one other checkpoint (exit). Without it, a
// container that lives entirely within one periodic interval would have
// max(cpu_usage_usec) == min(...) → billing = 0.
func (c *Collector) OnStart(_ context.Context, podUID string) {
	c.emitLifecycleCheckpoint(podUID, checkpoint.EventStart)
}

// OnExit is called by the CRI watcher when a container task exits. We read
// the cgroup one final time before kubelet garbage collects it, closing the
// undercharge window that would otherwise be up to one checkpoint_interval.
func (c *Collector) OnExit(_ context.Context, podUID string) {
	c.emitLifecycleCheckpoint(podUID, checkpoint.EventStop)
}

// emitLifecycleCheckpoint writes one checkpoint tied to a container lifecycle
// event (start or stop). Shared because both read the same cgroup fields —
// only the event_kind differs.
func (c *Collector) emitLifecycleCheckpoint(podUID, kind string) {
	pod := c.findPodByUID(podUID)
	if pod == nil {
		metrics.LifecycleDrops.WithLabelValues(kind, "pod_not_found").Inc()
		return // not a krane pod, not on this node, or informer cache stale
	}

	reading, err := c.cgroup.read(pod.UID, pod.Status.QOSClass)
	if err != nil {
		// cgroup not yet ready (start, kubelet still mounting) or already
		// gone (exit, cgroup torn down). Either way, accept the undercharge,
		// the next periodic tick or the last one is our best data.
		metrics.LifecycleDrops.WithLabelValues(kind, "cgroup_read_failed").Inc()
		return
	}

	info := buildPodInfo(pod)

	// Same rule as the periodic loop: if Status.ContainerStatuses hasn't
	// populated the primary container yet, we don't know the real
	// restart_count, so we can't emit a correctly-keyed checkpoint. Skip
	// and accept the undercharge. The next start / stop event or the
	// next periodic tick will catch it.
	if !info.restartCountKnown {
		metrics.LifecycleDrops.WithLabelValues(kind, "restart_count_unknown").Inc()
		return
	}

	containerUID := checkpoint.ContainerUID(string(pod.UID), info.restartCount)
	now := c.clk.Now().UnixMilli()

	var diskUsed int64
	if c.kubeletRoot != "" && info.diskAllocatedBytes > 0 {
		diskUsed = readEphemeralUsedBytes(c.kubeletRoot, pod.UID)
	}

	net, netAttached := c.attachAndReadNetwork(info)
	c.ch.Buffer(schema.InstanceCheckpoint{
		NodeID:                     c.nodeName,
		WorkspaceID:                info.workspaceID,
		ProjectID:                  info.projectID,
		EnvironmentID:              info.environmentID,
		ResourceType:               info.resourceType,
		ResourceID:                 info.resourceID,
		PodUID:                     string(pod.UID),
		InstanceID:                 info.name,
		ContainerUID:               containerUID,
		RestartCount:               uint32(info.restartCount),
		Ts:                         now,
		EventKind:                  kind,
		CPUUsageUsec:               reading.cpuUsageUsec,
		MemoryBytes:                reading.memoryBytes,
		CPUAllocatedMillicores:     info.cpuAllocatedMillicores,
		MemoryAllocatedBytes:       info.memoryAllocatedBytes,
		DiskAllocatedBytes:         info.diskAllocatedBytes,
		DiskUsedBytes:              diskUsed,
		NetworkEgressPublicBytes:   net.EgressPublicBytes,
		NetworkEgressPrivateBytes:  net.EgressPrivateBytes,
		NetworkIngressPublicBytes:  net.IngressPublicBytes,
		NetworkIngressPrivateBytes: net.IngressPrivateBytes,
		Region:                     c.region,
		Platform:                   c.platform,
		Attributes: schema.InstanceCheckpointAttributes{
			Image:              info.image,
			ImageID:            info.imageID,
			QOSClass:           string(info.qosClass),
			EBPFProgramVersion: buildinfo.Revision,
			EBPFPinDir:         network.PinDir(),
			NetworkAttached:    netAttached,
		}.Marshal(),
	})
	metrics.LifecycleEmitted.WithLabelValues(kind).Inc()
	metrics.CheckpointsWritten.Inc()

	// On final exit, drop the network attachments and map entry. The kernel
	// auto-detaches when the cgroup is reaped, but doing it eagerly keeps
	// the BPF map sparse and prevents stale entries from outliving their
	// cgroups for the LRU eviction cycle.
	if kind == checkpoint.EventStop && c.network != nil {
		c.network.Detach(pod.UID)
	}
}

// HandlePodUpdate is called by the pod informer on every Status change. It
// detects container lifecycle transitions (Running ↔ Terminated) on the
// primary container and fires the corresponding lifecycle checkpoint.
//
// This runs redundantly with the CRI watcher: CRI gives us ms-precise
// signals; the informer catches cases where containerd drops the event
// (containerd#3177) or where CRI is disabled. Duplicate checkpoints from
// both paths are fine — they carry different ts (and hence different
// idempotency keys), and max-min billing absorbs extra readings.
func (c *Collector) HandlePodUpdate(oldPod, newPod *corev1.Pod) {
	if newPod.Spec.NodeName != c.nodeName {
		return
	}
	if !isBillablePod(newPod) {
		return
	}

	name := primaryContainerName(newPod)
	if name == "" {
		return
	}

	oldStatus := findContainerStatus(oldPod, name)
	newStatus := findContainerStatus(newPod, name)
	if newStatus == nil {
		return // kubelet hasn't populated status yet
	}

	switch {
	case runningTransition(oldStatus, newStatus):
		c.OnStart(context.Background(), string(newPod.UID))
	case terminatedTransition(oldStatus, newStatus):
		c.OnExit(context.Background(), string(newPod.UID))
	}
}

// findContainerStatus returns the status entry for the named container, or
// nil if the pod/status is missing. Safe on nil pods (first informer add).
func findContainerStatus(pod *corev1.Pod, name string) *corev1.ContainerStatus {
	if pod == nil {
		return nil
	}

	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == name {
			return &pod.Status.ContainerStatuses[i]
		}
	}

	return nil
}

// runningTransition reports whether the container just entered the Running
// state. Covers initial admission (old == nil) and Waiting → Running.
func runningTransition(old, new *corev1.ContainerStatus) bool {
	if new == nil || new.State.Running == nil {
		return false
	}
	if old == nil || old.State.Running == nil {
		return true
	}
	// Same state; no transition. Restart (where old.RestartCount <
	// new.RestartCount and old.State.Running != nil) is intentionally
	// ignored here — CRI is the reliable path for restart detection
	// because the pod cgroup counter resets on the new container, and
	// firing a checkpoint against the new container_uid with its fresh
	// counter is still correct billing.
	return false
}

// terminatedTransition reports whether the container just entered the
// Terminated state. Covers Running → Terminated (normal exit) and any
// other transition into Terminated that wasn't already terminated.
func terminatedTransition(old, new *corev1.ContainerStatus) bool {
	if new == nil || new.State.Terminated == nil {
		return false
	}
	if old == nil || old.State.Terminated == nil {
		return true
	}
	// Already terminated; no transition.
	return false
}

// findPodByUID locates a krane/sentinel pod on this node by UID. Returns nil
// if the UID doesn't match a billable pod.
func (c *Collector) findPodByUID(uid string) *corev1.Pod {
	all, err := c.podLister.List(labels.Everything())
	if err != nil {
		return nil
	}

	target := types.UID(uid)
	for _, pod := range all {
		if pod.UID != target {
			continue
		}
		if pod.Spec.NodeName != c.nodeName {
			return nil
		}
		if !isBillablePod(pod) {
			return nil
		}

		return pod
	}

	return nil
}
