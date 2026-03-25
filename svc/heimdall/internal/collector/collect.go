package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/pkg/metrics"
	"k8s.io/apimachinery/pkg/labels"
)

type cpuReading struct {
	nanoseconds uint64
	txBytes     uint64
	timestamp   time.Time
}

type podInfo struct {
	workspaceID     string
	projectID       string
	appID           string
	environmentID   string
	deploymentID    string
	cpuRequestMilli int32
	cpuLimitMilli   int32
	memRequestBytes int64
	memLimitBytes   int64
}

func (c *Collector) collect(ctx context.Context) error {
	summary, err := c.fetchSummary(ctx)
	if err != nil {
		return fmt.Errorf("fetching kubelet summary: %w", err)
	}

	kranePodInfo := c.buildKranePodLookup()

	var collected int
	seen := make(map[string]struct{})

	for _, pod := range summary.Pods {
		podName := pod.PodRef.Name

		if _, isKrane := kranePodInfo[podName]; !isKrane {
			continue
		}

		seen[podName] = struct{}{}

		if c.processSummaryPod(pod, kranePodInfo[podName]) {
			collected++
		}
	}

	// Clean up readings for pods that no longer exist
	for podName := range c.prevReadings {
		if _, ok := seen[podName]; !ok {
			delete(c.prevReadings, podName)
		}
	}

	metrics.KranePods.Set(float64(len(seen)))

	logger.Info("collection tick",
		"total_pods", len(summary.Pods),
		"krane_pods", len(seen),
		"samples_buffered", collected,
		"first_readings", len(seen)-collected,
	)

	return nil
}

// processPod finds a specific pod in the kubelet summary and processes it.
// Used by CollectPod for immediate single-pod collection on lifecycle events.
func (c *Collector) processPod(summary *kubeletSummary, podName string) {
	kranePodInfo := c.buildKranePodLookup()

	info, isKrane := kranePodInfo[podName]
	if !isKrane {
		return
	}

	for _, pod := range summary.Pods {
		if pod.PodRef.Name != podName {
			continue
		}
		c.processSummaryPod(pod, info)
		return
	}
}

// processSummaryPod extracts metrics from a kubelet summary pod, computes
// CPU rate from the previous reading, and buffers the sample.
// Returns true if a sample was buffered (false if this is a first reading).
func (c *Collector) processSummaryPod(pod kubeletPod, info podInfo) bool {
	podName := pod.PodRef.Name
	now := time.Now()

	var cpuNano uint64
	var memBytes int64
	for _, cont := range pod.Containers {
		if cont.CPU.UsageCoreNanoSeconds != nil {
			cpuNano += *cont.CPU.UsageCoreNanoSeconds
		}
		if cont.Memory.WorkingSetBytes != nil {
			memBytes += int64(*cont.Memory.WorkingSetBytes)
		}
	}

	var txBytes uint64
	if pod.Network != nil {
		for _, iface := range pod.Network.Interfaces {
			txBytes += iface.TxBytes
		}
	}

	prev, hasPrev := c.prevReadings[podName]
	c.prevReadings[podName] = cpuReading{
		nanoseconds: cpuNano,
		txBytes:     txBytes,
		timestamp:   now,
	}

	if !hasPrev {
		return false
	}

	elapsed := now.Sub(prev.timestamp)
	if elapsed <= 0 {
		return false
	}

	// CPU counter reset detection: if the current counter is lower than the
	// previous, the container restarted within this pod. Skip this sample —
	// the next tick will have a clean baseline. Without this, the uint64
	// underflow wraps to a huge number and produces a massive fake CPU spike.
	if cpuNano < prev.nanoseconds {
		logger.Info("cpu counter reset detected, skipping sample",
			"pod", podName,
			"prev_ns", prev.nanoseconds,
			"curr_ns", cpuNano,
		)
		return false
	}

	deltaCPU := cpuNano - prev.nanoseconds
	cpuMillicores := float64(deltaCPU) / float64(elapsed.Nanoseconds()) * 1000.0

	// Network counter reset: same idea, skip negative deltas
	var deltaTx int64
	if txBytes >= prev.txBytes {
		deltaTx = int64(txBytes - prev.txBytes)
	}

	c.ch.BufferContainerResource(schema.ContainerResource{
		Time:                  now.UnixMilli(),
		WorkspaceID:           info.workspaceID,
		ProjectID:             info.projectID,
		AppID:                 info.appID,
		EnvironmentID:         info.environmentID,
		DeploymentID:          info.deploymentID,
		InstanceID:            podName,
		Region:                c.region,
		Platform:              c.platform,
		CPUMillicores:         cpuMillicores,
		MemoryWorkingSetBytes: memBytes,
		CPURequestMillicores:  info.cpuRequestMilli,
		CPULimitMillicores:    info.cpuLimitMilli,
		MemoryRequestBytes:    info.memRequestBytes,
		MemoryLimitBytes:      info.memLimitBytes,
		NetworkTxBytes:        deltaTx,
		NetworkTxBytesPublic:  0, // TODO: Cilium Hubble integration
	})

	return true
}

// buildKranePodLookup creates a map of pod name -> labels + limits from the informer cache.
func (c *Collector) buildKranePodLookup() map[string]podInfo {
	kranePods := make(map[string]podInfo)
	allPods, err := c.podLister.List(labels.Everything())
	if err != nil {
		logger.Error("failed to list pods from cache", "error", err.Error())
		return kranePods
	}

	for _, pod := range allPods {
		if pod.Labels[LabelManagedBy] != "krane" || pod.Labels[LabelComponent] != "deployment" {
			continue
		}

		info := podInfo{
			workspaceID:   pod.Labels[LabelWorkspace],
			projectID:     pod.Labels[LabelProject],
			appID:         pod.Labels[LabelApp],
			environmentID: pod.Labels[LabelEnv],
			deploymentID:  pod.Labels[LabelDeployment],
		}

		if len(pod.Spec.Containers) > 0 {
			limits := pod.Spec.Containers[0].Resources.Limits
			requests := pod.Spec.Containers[0].Resources.Requests
			info.cpuRequestMilli = int32(requests.Cpu().MilliValue())
			info.cpuLimitMilli = int32(limits.Cpu().MilliValue())
			info.memRequestBytes = requests.Memory().Value()
			info.memLimitBytes = limits.Memory().Value()
		}

		kranePods[pod.Name] = info
	}

	return kranePods
}
