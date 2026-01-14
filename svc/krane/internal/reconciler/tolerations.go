package reconciler

import corev1 "k8s.io/api/core/v1"

var untrustedTolerations = []corev1.Toleration{
	{
		Key:      "node-class",
		Operator: corev1.TolerationOpEqual,
		Value:    "untrusted",
		Effect:   corev1.TaintEffectNoSchedule,
	},
}

var sentinelTolerations = []corev1.Toleration{
	{
		Key:      "node-class",
		Operator: corev1.TolerationOpEqual,
		Value:    "sentinel",
		Effect:   corev1.TaintEffectNoSchedule,
	},
}
