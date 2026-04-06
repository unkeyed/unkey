package collector

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/internal/checkpoint"
	"github.com/unkeyed/unkey/svc/heimdall/internal/metrics"
	"github.com/unkeyed/unkey/svc/heimdall/internal/network"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

// podInfo carries everything the collector needs to build a checkpoint for
// one container. It is populated from the informer cache at the start of each
// tick (we never block a cgroup read on an API call).
type podInfo struct {
	name          string
	uid           types.UID
	qosClass      corev1.PodQOSClass
	workspaceID   string
	projectID     string
	environmentID string
	resourceType  string
	resourceID    string
	restartCount  int32
	// restartCountKnown is false when Status.ContainerStatuses has not yet
	// caught up with the primary container. The collect loop skips pods
	// with restartCountKnown=false for this tick; the next tick retries.
	restartCountKnown      bool
	hostNetwork            bool
	podIP                  string          // primary IPv4 assigned by the CNI; used to find the pod's CNI netns at tc attach time
	phase                  corev1.PodPhase // skip network attach for non-Running pods (sandbox container is gone after Completion)
	cpuAllocatedMillicores int32
	memoryAllocatedBytes   int64
	diskAllocatedBytes     int64
	// Image identity from Status.ContainerStatuses[primary]. Recorded into
	// the checkpoint's JSON attributes for debugging — "this OOM correlates
	// with this exact image"  — without needing a join to live pod state.
	image   string
	imageID string
}

// collect runs one tick: list krane pods on this node, read their cgroup
// counters, and buffer one checkpoint per container. All billing math is
// deferred to query time.
func (c *Collector) collect(_ context.Context) error {
	tickStart := time.Now()
	now := c.clk.Now().UnixMilli()
	pods := c.buildKranePodLookup()

	var written int
	for _, info := range pods {
		// If Status.ContainerStatuses hasn't caught up with the primary
		// container yet, we don't know the real restart_count. Skip this
		// tick rather than stamp container_uid=pod_uid/0 and risk merging
		// a fresh incarnation's rows into a prior incarnation's series.
		// At worst we lose one tick of data (~5s) for a freshly-started
		// container, which is safe-direction (undercharge) and bounded.
		if !info.restartCountKnown {
			metrics.PeriodicSkips.WithLabelValues("restart_count_unknown").Inc()
			continue
		}

		// Skip the cgroup read entirely when both cpu and memory are
		// disabled — that's the only data it produces. Either alone still
		// requires the read; we just zero out the disabled column below.
		var reading cgroupReading
		if c.collectors.CPU || c.collectors.Memory {
			r, err := c.cgroup.read(info.uid, info.qosClass)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					metrics.CgroupReadErrors.Inc()
					logger.Error("cgroup read failed", "pod", info.name, "error", err.Error())
				}
				metrics.PeriodicSkips.WithLabelValues("cgroup_read_failed").Inc()
				continue
			}
			reading = r
		}

		containerUID := checkpoint.ContainerUID(string(info.uid), info.restartCount)

		var diskUsed, diskAllocated int64
		if c.collectors.Disk {
			diskAllocated = info.diskAllocatedBytes
			if c.kubeletRoot != "" && info.diskAllocatedBytes > 0 {
				diskUsed = readEphemeralUsedBytes(c.kubeletRoot, info.uid)
			}
		}

		var (
			cpuUsec       int64
			cpuAllocMilli int32
			memoryBytes   int64
			memoryAlloc   int64
		)
		if c.collectors.CPU {
			cpuUsec = reading.cpuUsageUsec
			cpuAllocMilli = info.cpuAllocatedMillicores
		}
		if c.collectors.Memory {
			memoryBytes = reading.memoryBytes
			memoryAlloc = info.memoryAllocatedBytes
		}

		net, netAttached := c.attachAndReadNetwork(info)
		c.ch.Buffer(schema.InstanceCheckpoint{
			NodeID:                     c.nodeName,
			WorkspaceID:                info.workspaceID,
			ProjectID:                  info.projectID,
			EnvironmentID:              info.environmentID,
			ResourceType:               info.resourceType,
			ResourceID:                 info.resourceID,
			PodUID:                     string(info.uid),
			InstanceID:                 info.name,
			ContainerUID:               containerUID,
			RestartCount:               uint32(info.restartCount),
			Ts:                         now,
			EventKind:                  checkpoint.EventPeriodic,
			CPUUsageUsec:               cpuUsec,
			MemoryBytes:                memoryBytes,
			CPUAllocatedMillicores:     cpuAllocMilli,
			MemoryAllocatedBytes:       memoryAlloc,
			DiskAllocatedBytes:         diskAllocated,
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
				Collectors:         c.enabledCollectorNames(),
			}.Marshal(),
		})
		metrics.CheckpointsWritten.Inc()
		written++
	}

	metrics.KranePods.Set(float64(len(pods)))
	if c.network != nil {
		// Sample the BPF map depth so operators can alert before LRU
		// eviction kicks in (the map has max_entries=16384; beyond that
		// the kernel silently drops traffic from the oldest veth). With
		// kubelet default max-pods of 110-250, this gives ~40x headroom.
		metrics.BPFMapEntries.Set(float64(c.network.MapEntries()))
	}

	logger.Info("collection tick",
		"node", c.nodeName,
		"krane_pods", len(pods),
		"checkpoints_written", written,
		"duration", time.Since(tickStart).String(),
	)
	return nil
}

func (c *Collector) buildKranePodLookup() map[string]podInfo {
	pods := make(map[string]podInfo)
	allPods, err := c.podLister.List(labels.Everything())
	if err != nil {
		logger.Error("failed to list pods from cache", "error", err.Error())
		return pods
	}

	for _, pod := range allPods {
		if pod.Spec.NodeName != c.nodeName {
			continue
		}
		if !isBillablePod(pod) {
			continue
		}
		pods[string(pod.UID)] = buildPodInfo(pod)
	}

	return pods
}

// isBillablePod returns true if pod is a krane-managed deployment or a sentinel.
func isBillablePod(pod *corev1.Pod) bool {
	component := pod.Labels[LabelComponent]
	if component == "deployment" && pod.Labels[LabelManagedBy] != "krane" {
		return false
	}
	return component == "deployment" || component == "sentinel"
}

// buildPodInfo extracts the billing-relevant fields from a pod.
func buildPodInfo(pod *corev1.Pod) podInfo {
	component := pod.Labels[LabelComponent]
	resourceID := pod.Labels[LabelDeployment]
	if component == "sentinel" {
		resourceID = pod.Labels[LabelSentinel]
	}
	cpuMilli, memBytes := primaryContainerAllocation(pod)
	restartCount, restartCountKnown := primaryContainerRestartCount(pod)
	image, imageID := primaryContainerImage(pod)
	return podInfo{
		name:                   pod.Name,
		uid:                    pod.UID,
		qosClass:               pod.Status.QOSClass,
		workspaceID:            pod.Labels[LabelWorkspace],
		projectID:              pod.Labels[LabelProject],
		environmentID:          pod.Labels[LabelEnv],
		resourceType:           component,
		resourceID:             resourceID,
		restartCount:           restartCount,
		restartCountKnown:      restartCountKnown,
		hostNetwork:            pod.Spec.HostNetwork,
		podIP:                  pod.Status.PodIP,
		phase:                  pod.Status.Phase,
		cpuAllocatedMillicores: cpuMilli,
		memoryAllocatedBytes:   memBytes,
		diskAllocatedBytes:     ephemeralStorageBytes(pod),
		image:                  image,
		imageID:                imageID,
	}
}

// primaryContainerAllocation returns the CPU (millicores) and memory (bytes)
// limits declared on the pod's primary container. We prefer Limits over
// Requests because Limits is the true ceiling, which is what the customer is paying
// for and can actually use. Requests is a scheduling hint (often set lower
// for bin-packing, e.g., krane sets Requests = Limits / 4). For utilization
// dashboards ("how much of my capacity am I using"), the denominator that
// matters is the Limit, not the Request.
// If Limits is absent, fall back to Requests.
func primaryContainerAllocation(pod *corev1.Pod) (int32, int64) {
	if len(pod.Spec.Containers) == 0 {
		return 0, 0
	}

	c := pod.Spec.Containers[0]

	cpu := c.Resources.Limits.Cpu()
	if cpu == nil || cpu.IsZero() {
		cpu = c.Resources.Requests.Cpu()
	}

	mem := c.Resources.Limits.Memory()
	if mem == nil || mem.IsZero() {
		mem = c.Resources.Requests.Memory()
	}

	var cpuMilli int32
	if cpu != nil {
		cpuMilli = int32(cpu.MilliValue())
	}

	var memBytes int64
	if mem != nil {
		memBytes = mem.Value()
	}

	return cpuMilli, memBytes
}

// primaryContainerRestartCount returns the restart count of the pod's primary
// container (the one whose name appears first in Spec.Containers) and whether
// a status entry for that container was present. We look it up by name, not by
// status index, because Status.ContainerStatuses ordering is not part of the
// K8s API contract: sidecar restarts can reshuffle indices and silently change
// the container_uid we bill against, causing overcharge.
//
// Returns (0, false) when Status.ContainerStatuses is not yet populated for
// the primary container. Callers must treat that as "restart count unknown
// right now" and skip emitting a checkpoint rather than defaulting to 0. A
// naive fallback to 0 would, for a pod whose real restartCount is N>0, stamp
// some rows with container_uid=pod_uid/0 and others with pod_uid/N, splitting
// one container's span across two logical series.
func primaryContainerRestartCount(pod *corev1.Pod) (int32, bool) {
	name := primaryContainerName(pod)
	if name == "" {
		return 0, false
	}

	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == name {
			return pod.Status.ContainerStatuses[i].RestartCount, true
		}
	}

	return 0, false
}

// primaryContainerImage returns the image and image ID currently observed by
// kubelet for the primary container. Both can be empty if Status hasn't caught
// up yet (same race as primaryContainerRestartCount). Empty values are stamped
// onto the checkpoint as-is — they're observational, not part of billing math.
func primaryContainerImage(pod *corev1.Pod) (string, string) {
	name := primaryContainerName(pod)
	if name == "" {
		return "", ""
	}

	for i := range pod.Status.ContainerStatuses {
		if pod.Status.ContainerStatuses[i].Name == name {
			return pod.Status.ContainerStatuses[i].Image, pod.Status.ContainerStatuses[i].ImageID
		}
	}

	return "", ""
}

// primaryContainerName returns the name of the primary container. First in
// Spec.Containers (spec order is stable; only the declared order changes with
// new deployments, never transiently).
func primaryContainerName(pod *corev1.Pod) string {
	if len(pod.Spec.Containers) == 0 {
		return ""
	}

	return pod.Spec.Containers[0].Name
}

// zeroCounters is the all-zero value the fail-open paths below return.
// Declared once so we don't repeat the exhaustruct-shaped literal four
// times. Mirrors network.zeroCounters which serves the same role inside
// the network package.
var zeroCounters = network.Counters{
	EgressPublicBytes:   0,
	EgressPrivateBytes:  0,
	IngressPublicBytes:  0,
	IngressPrivateBytes: 0,
}

// attachAndReadNetwork lazily attaches the eBPF TCX counters on the pod's
// host-side veth on first observation, then reads the current snapshot.
// Host-network pods (heimdall itself, kube-proxy, sentinels) share the
// host net namespace and have no per-pod-veth traffic worth attributing,
// so they're skipped. Read failures fail open (return zero counters) so
// a flaky eBPF map can never take down the rest of the checkpoint.
//
// The bool return reports whether the eBPF read actually succeeded for this
// tick. The collector stamps it onto the checkpoint's attributes so query-
// time consumers can tell "real zero traffic" apart from "we couldn't read".
func (c *Collector) attachAndReadNetwork(info podInfo) (network.Counters, bool) {
	// Disabled-by-config short-circuit: never attach, never read. run.go
	// also passes nil network reader in this case so the BPF program
	// never loads — the c.network nil check below catches that path too,
	// but we short-circuit here for clarity.
	if !c.collectors.Network {
		return zeroCounters, false
	}
	if c.network == nil || info.hostNetwork || info.podIP == "" {
		return zeroCounters, false
	}

	// Completed/Failed pods keep showing up in the informer cache until
	// kubelet GC runs, but their sandbox containers are already gone, so any
	// Attach call would log a warn and churn retries. Skip them: if they
	// were previously attached, the final pre-exit map read already happened.
	if info.phase != corev1.PodRunning {
		return zeroCounters, false
	}

	if err := c.network.Attach(info.uid); err != nil {
		// Attach is async and idempotent; the only error it surfaces is
		// ErrAttachQueueFull, meaning a rollout storm pushed the queue
		// past capacity. Counter-side undercount for this tick is the
		// cost; next tick re-requests. Real attach failures (netns gone,
		// TCX rejected, etc.) are reported from the worker into
		// NetworkAttachFailures, not from here.
		metrics.NetworkAttachFailures.WithLabelValues("queue_full").Inc()
		return zeroCounters, false
	}

	counters, err := c.network.Read(info.uid)
	if err != nil {
		metrics.NetworkReadErrors.Inc()
		return zeroCounters, false
	}

	return counters, true
}

// ephemeralStorageBytes returns the allocated storage size (bytes) requested
// by the first ephemeral volume in the pod spec. Non-ephemeral pods return 0.
func ephemeralStorageBytes(pod *corev1.Pod) int64 {
	for _, vol := range pod.Spec.Volumes {
		if vol.Ephemeral == nil || vol.Ephemeral.VolumeClaimTemplate == nil {
			continue
		}

		q, ok := vol.Ephemeral.VolumeClaimTemplate.Spec.Resources.Requests[corev1.ResourceStorage]
		if !ok {
			continue
		}

		v, ok := q.AsInt64()
		if !ok || v <= 0 {
			continue
		}

		return v
	}

	return 0
}
