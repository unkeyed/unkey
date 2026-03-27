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
	deploymentID    string
	cpuRequestMilli int32
	cpuLimitMilli   int32
	memRequestBytes int64
	memLimitBytes   int64
}

func (c *Collector) collect(ctx context.Context) error {
	now := time.Now()

	// 1. Get pod metrics from Metrics Server
	podMetricsList, err := c.metricsClient.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pod metrics: %w", err)
	}

	// 2. Build krane pod lookup from informer cache
	kranePods := c.buildKranePodLookup()

	// 3. Match metrics to krane pods and write snapshots
	var snapshotCount int
	for _, pm := range podMetricsList.Items {
		info, isKrane := kranePods[pm.Name]
		if !isKrane {
			continue
		}

		// Sum CPU and memory across all containers
		var cpuMilli int64
		var memBytes int64
		for _, container := range pm.Containers {
			cpuMilli += container.Usage.Cpu().MilliValue()
			memBytes += container.Usage.Memory().Value()
		}

		c.ch.BufferResourceSnapshot(schema.ResourceSnapshot{
			Time:                 now.Unix(),
			WorkspaceID:          info.workspaceID,
			ProjectID:            info.projectID,
			AppID:                info.appID,
			EnvironmentID:        info.environmentID,
			DeploymentID:         info.deploymentID,
			InstanceID:           pm.Name,
			Region:               c.region,
			Platform:             c.platform,
			CPUMillicores:        int32(cpuMilli),
			MemoryBytes:          memBytes,
			CPURequestMillicores: info.cpuRequestMilli,
			CPULimitMillicores:   info.cpuLimitMilli,
			MemoryRequestBytes:   info.memRequestBytes,
			MemoryLimitBytes:     info.memLimitBytes,
			NetworkEgressBytes:   0, // TODO: Hubble integration
			NetworkEgressPublic:  0, // TODO: Hubble integration
		})
		snapshotCount++
	}

	metrics.KranePods.Set(float64(snapshotCount))

	logger.Info("collection tick",
		"metrics_server_pods", len(podMetricsList.Items),
		"snapshots_written", snapshotCount,
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
