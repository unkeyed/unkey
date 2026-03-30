package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/pkg/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type podInfo struct {
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

func (c *Collector) collect(ctx context.Context) error {
	now := time.Now()

	kranePods := c.buildKranePodLookup()

	// Build IP → pod name map for conntrack attribution
	podIPs := make(map[string]string, len(kranePods))
	for name, info := range kranePods {
		if info.podIP != "" {
			podIPs[info.podIP] = name
		}
	}

	podMetricsList, err := c.metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pod metrics: %w", err)
	}

	currentEgress, err := collectEgress(podIPs, c.internalCIDRs)
	if err != nil {
		logger.Error("conntrack failed, writing snapshots without network data", "error", err.Error())
		currentEgress = make(map[string]podEgress)
	}

	// Conntrack counters are cumulative — compute delta from previous tick
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
	var snapshotCount int
	for _, pm := range podMetricsList.Items {
		info, isKrane := kranePods[pm.Name]
		if !isKrane {
			continue
		}

		var cpuMilli int64
		var memBytes int64
		for _, container := range pm.Containers {
			cpuMilli += container.Usage.Cpu().MilliValue()
			memBytes += container.Usage.Memory().Value()
		}

		egress := egressDeltas[pm.Name]

		c.ch.Buffer(schema.ResourceSnapshot{
			Time:                 now.Unix(),
			WorkspaceID:          info.workspaceID,
			ProjectID:            info.projectID,
			AppID:                info.appID,
			EnvironmentID:        info.environmentID,
			ResourceType:         info.resourceType,
			ResourceID:           info.resourceID,
			InstanceID:           pm.Name,
			Region:               c.region,
			Platform:             c.platform,
			CPUMillicores:        int32(cpuMilli),
			MemoryBytes:          memBytes,
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

	metrics.KranePods.Set(float64(snapshotCount))

	logger.Info("collection tick",
		"node", c.nodeName,
		"metrics_server_pods", len(podMetricsList.Items),
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

		// Deployments use deployment.id, sentinels use sentinel.id
		resourceID := pod.Labels[LabelDeployment]
		if component == "sentinel" {
			resourceID = pod.Labels[LabelSentinel]
		}

		var startedAtMs int64
		if pod.Status.StartTime != nil {
			startedAtMs = pod.Status.StartTime.UnixMilli()
		}

		info := podInfo{
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

		kranePods[pod.Name] = info
	}

	return kranePods
}
