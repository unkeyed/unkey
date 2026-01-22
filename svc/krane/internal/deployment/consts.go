package deployment

import corev1 "k8s.io/api/core/v1"

const (
	// DeploymentPort is the port user deployments listen on.
	DeploymentPort = 8080

	// runtimeClassGvisor specifies the gVisor sandbox for untrusted user workloads.
	runtimeClassGvisor = "gvisor"

	// fieldManagerKrane identifies krane as the server-side apply field manager.
	fieldManagerKrane = "krane"

	// CustomerNodeClass is the node class for untrusted customer workloads.
	CustomerNodeClass = "untrusted"
)

// untrustedToleration allows pods to be scheduled on untrusted nodes.
var untrustedToleration = corev1.Toleration{
	Key:      "karpenter.sh/nodepool",
	Operator: corev1.TolerationOpEqual,
	Value:    CustomerNodeClass,
	Effect:   corev1.TaintEffectNoSchedule,
}
