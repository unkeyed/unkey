package reconciler

import corev1 "k8s.io/api/core/v1"

var untrustedToleration = corev1.Toleration{
	Key:      "node-class",
	Operator: corev1.TolerationOpEqual,
	Value:    NamespaceUntrusted,
	Effect:   corev1.TaintEffectNoSchedule,
}

var sentinelToleration = corev1.Toleration{
	Key:      "node-class",
	Operator: corev1.TolerationOpEqual,
	Value:    NamespaceSentinel,
	Effect:   corev1.TaintEffectNoSchedule,
}
