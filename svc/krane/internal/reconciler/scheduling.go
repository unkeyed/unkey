package reconciler

import (
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// topologyKeyZone is the standard Kubernetes topology key for availability zones
	topologyKeyZone = "topology.kubernetes.io/zone"
)

// deploymentTopologySpread returns topology spread constraints for customer deployment pods.
// Spreads pods evenly across availability zones with maxSkew of 1.
func deploymentTopologySpread(deploymentID string) []corev1.TopologySpreadConstraint {
	return []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: labels.New().DeploymentID(deploymentID),
			},
		},
	}
}

// sentinelTopologySpread returns topology spread constraints for sentinel pods.
// Spreads pods evenly across availability zones with maxSkew of 1.
func sentinelTopologySpread(sentinelID string) []corev1.TopologySpreadConstraint {
	return []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: labels.New().SentinelID(sentinelID),
			},
		},
	}
}

// deploymentAffinity returns affinity rules for customer deployment pods.
// Prefers scheduling in the same AZ as sentinels for the given environment
// to minimize cross-AZ latency between sentinel and customer code.
func deploymentAffinity(environmentID string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAffinity: &corev1.PodAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						TopologyKey: topologyKeyZone,
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: labels.New().
								EnvironmentID(environmentID).
								ComponentSentinel(),
						},
					},
				},
			},
		},
	}
}
