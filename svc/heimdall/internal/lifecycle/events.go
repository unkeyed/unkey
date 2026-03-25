package lifecycle

import (
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/heimdall/internal/collector"
	corev1 "k8s.io/api/core/v1"
)

func (t *Tracker) emitEvent(pod *corev1.Pod, event string) {
	labels := pod.Labels

	var cpuLimitMilli int32
	var memLimitBytes int64
	if containers := pod.Spec.Containers; len(containers) > 0 {
		if limits := containers[0].Resources.Limits; limits != nil {
			cpuLimitMilli = int32(limits.Cpu().MilliValue())
			memLimitBytes = limits.Memory().Value()
		}
	}

	// Report 1 per pod event — the billing service aggregates per deployment
	var replicas int32 = 1

	t.ch.BufferDeploymentLifecycleEvent(schema.DeploymentLifecycleEvent{
		Time:               time.Now().UnixMilli(),
		WorkspaceID:        labels[collector.LabelWorkspace],
		ProjectID:          labels[collector.LabelProject],
		AppID:              labels[collector.LabelApp],
		EnvironmentID:      labels[collector.LabelEnv],
		DeploymentID:       labels[collector.LabelDeployment],
		Region:             t.region,
		Platform:           t.platform,
		Event:              event,
		Replicas:           replicas,
		CPULimitMillicores: cpuLimitMilli,
		MemoryLimitBytes:   memLimitBytes,
	})

	logger.Info("lifecycle event",
		"event", event,
		"pod", pod.Name,
		"deployment_id", labels[collector.LabelDeployment],
	)
}

func isKraneManagedDeployment(pod *corev1.Pod) bool {
	return pod.Labels[collector.LabelManagedBy] == "krane" &&
		pod.Labels[collector.LabelComponent] == "deployment"
}
