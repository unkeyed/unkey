package deployment

import corev1 "k8s.io/api/core/v1"

const (
	// runtimeClassGvisor specifies the gVisor sandbox RuntimeClass for running
	// untrusted user workloads with kernel-level isolation.
	runtimeClassGvisor = "gvisor"

	// fieldManagerKrane identifies krane as the server-side apply field manager,
	// so field ownership/conflict detection is tracked per manager.
	fieldManagerKrane = "krane"

	// CustomerNodeClass is the Karpenter nodepool name for untrusted customer
	// workloads. Nodes in this pool have additional isolation and monitoring.
	CustomerNodeClass = "untrusted"

	// resourceRequestFraction is the fraction of limits used for resource requests.
	// Requests determine scheduling; limits cap actual usage.
	resourceRequestFraction = 4 // requests = limits / 4

	// defaultCPUTargetUtilization is the average CPU utilization percentage (0-100)
	// used when no autoscaling policy is attached. When average pod CPU exceeds
	// this percentage of their requested CPU, the HPA adds replicas.
	defaultCPUTargetUtilization = 80

	// scaleDownStabilizationSeconds is how long the HPA waits after load drops
	// before removing pods. Prevents flapping when traffic is spiky.
	scaleDownStabilizationSeconds int32 = 60
)

// untrustedToleration allows deployment pods to be scheduled on nodes tainted
// for untrusted workloads. Without this toleration, pods would be rejected by
// the Karpenter-managed nodepool's NoSchedule taint.
var untrustedToleration = corev1.Toleration{
	Key:      "karpenter.sh/nodepool",
	Operator: corev1.TolerationOpEqual,
	Value:    CustomerNodeClass,
	Effect:   corev1.TaintEffectNoSchedule,
}
