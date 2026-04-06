package collector

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

type podInfo struct {
	name            string
	uid             types.UID
	qosClass        corev1.PodQOSClass
	workspaceID     string
	projectID       string
	appID           string
	environmentID   string
	resourceType    string
	resourceID      string
	podIP           string
	startedAtMs     int64
	cpuRequestMilli int32
	cpuLimitMilli   int32
	memRequestBytes int64
	memLimitBytes   int64
}

func (c *Collector) collect(_ context.Context) error {
	now := time.Now()

	kranePods := c.buildKranePodLookup()

	// Build IP → pod name map for conntrack attribution
	podIPs := make(map[string]string, len(kranePods))
	for _, info := range kranePods {
		if info.podIP != "" {
			podIPs[info.podIP] = info.name
		}
	}

	// Conntrack for network egress
	currentEgress, err := collectEgress(podIPs, c.internalCIDRs)
	if err != nil {
		logger.Error("conntrack failed, writing snapshots without network data", "error", err.Error())
		currentEgress = make(map[string]podEgress)
	}

	egressDeltas := make(map[string]podEgress, len(currentEgress))
	for podName, current := range currentEgress {
		if prev, hasPrev := c.prevEgress[podName]; hasPrev {
			delta := podEgress{
				totalBytes:  current.totalBytes - prev.totalBytes,
				publicBytes: current.publicBytes - prev.publicBytes,
			}
			if delta.totalBytes < 0 {
				delta.totalBytes = 0
			}
			if delta.publicBytes < 0 {
				delta.publicBytes = 0
			}
			egressDeltas[podName] = delta
		}
	}
	c.prevEgress = currentEgress

	// Read cgroup stats and write snapshots
	var snapshotCount int
	seenUIDs := make(map[string]struct{}, len(kranePods))

	for uidStr, info := range kranePods {
		seenUIDs[uidStr] = struct{}{}

		prev := c.prevCPU[uidStr]
		resources, reading, err := c.cgroup.readPodResources(info.uid, info.qosClass, prev)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				metrics.CgroupReadErrors.Inc()
				logger.Error("cgroup read failed", "pod", info.name, "error", err.Error())
			}
			continue
		}
		c.prevCPU[uidStr] = &reading

		// First tick: no CPU delta yet, skip snapshot
		if prev == nil {
			continue
		}

		egress := egressDeltas[info.name]

		c.ch.Buffer(schema.ResourceSnapshot{
			Time:                 now.UnixMilli(),
			WorkspaceID:          info.workspaceID,
			ProjectID:            info.projectID,
			AppID:                info.appID,
			EnvironmentID:        info.environmentID,
			ResourceType:         info.resourceType,
			ResourceID:           info.resourceID,
			InstanceID:           info.name,
			Region:               c.region,
			Platform:             c.platform,
			CPUMillicores:        resources.cpuMillicores,
			MemoryBytes:          resources.memoryBytes,
			CPURequestMillicores: info.cpuRequestMilli,
			CPULimitMillicores:   info.cpuLimitMilli,
			MemoryRequestBytes:   info.memRequestBytes,
			MemoryLimitBytes:     info.memLimitBytes,
			NetworkEgressBytes:   egress.totalBytes,
			NetworkEgressPublic:  egress.publicBytes,
			StartedAt:            info.startedAtMs,
		})
		snapshotCount++
	}

	// Clean up prevCPU for pods that no longer exist
	for uid := range c.prevCPU {
		if _, ok := seenUIDs[uid]; !ok {
			delete(c.prevCPU, uid)
		}
	}

	metrics.KranePods.Set(float64(len(kranePods)))

	logger.Info("collection tick",
		"node", c.nodeName,
		"krane_pods", len(kranePods),
		"snapshots_written", snapshotCount,
		"conntrack_pods", len(currentEgress),
	)

	return nil
}

func (c *Collector) buildKranePodLookup() map[string]podInfo {
	kranePods := make(map[string]podInfo)
	allPods, err := c.podLister.List(labels.Everything())
	if err != nil {
		logger.Error("failed to list pods from cache", "error", err.Error())
		return kranePods
	}

	for _, pod := range allPods {
		if pod.Spec.NodeName != c.nodeName {
			continue
		}
		component := pod.Labels[LabelComponent]
		if component == "deployment" && pod.Labels[LabelManagedBy] != "krane" {
			continue
		}
		if component != "deployment" && component != "sentinel" {
			continue
		}

		resourceID := pod.Labels[LabelDeployment]
		if component == "sentinel" {
			resourceID = pod.Labels[LabelSentinel]
		}

		var startedAtMs int64
		if pod.Status.StartTime != nil {
			startedAtMs = pod.Status.StartTime.UnixMilli()
		}

		info := podInfo{
			name:            pod.Name,
			uid:             pod.UID,
			qosClass:        pod.Status.QOSClass,
			workspaceID:     pod.Labels[LabelWorkspace],
			projectID:       pod.Labels[LabelProject],
			appID:           pod.Labels[LabelApp],
			environmentID:   pod.Labels[LabelEnv],
			resourceType:    component,
			resourceID:      resourceID,
			podIP:           pod.Status.PodIP,
			startedAtMs:     startedAtMs,
			cpuRequestMilli: 0,
			cpuLimitMilli:   0,
			memRequestBytes: 0,
			memLimitBytes:   0,
		}

		if len(pod.Spec.Containers) > 0 {
			limits := pod.Spec.Containers[0].Resources.Limits
			requests := pod.Spec.Containers[0].Resources.Requests
			info.cpuRequestMilli = int32(requests.Cpu().MilliValue())
			info.cpuLimitMilli = int32(limits.Cpu().MilliValue())
			info.memRequestBytes = requests.Memory().Value()
			info.memLimitBytes = limits.Memory().Value()
		}

		kranePods[string(pod.UID)] = info
	}

	return kranePods
}

