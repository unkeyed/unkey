package sentinel

import corev1 "k8s.io/api/core/v1"

const (
	// NamespaceSentinel is the Kubernetes namespace where sentinel pods run.
	NamespaceSentinel = "sentinel"

	// SentinelPort is the port sentinel pods listen on.
	SentinelPort = 8040

	// SentinelNodeClass is the node class for sentinel workloads.
	SentinelNodeClass = "sentinel"

	// fieldManagerKrane identifies krane as the server-side apply field manager.
	fieldManagerKrane = "krane"

	// topologyKeyZone is the standard Kubernetes topology key for availability zones
	topologyKeyZone = "topology.kubernetes.io/zone"
)

// sentinelToleration allows sentinel pods to be scheduled on sentinel nodes.
var sentinelToleration = corev1.Toleration{
	Key:      "node-class",
	Operator: corev1.TolerationOpEqual,
	Value:    SentinelNodeClass,
	Effect:   corev1.TaintEffectNoSchedule,
}
