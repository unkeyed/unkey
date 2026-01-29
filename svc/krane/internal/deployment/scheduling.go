package deployment

import (
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// topologyKeyZone is the standard Kubernetes label for availability zones,
	// used for spreading pods across zones for high availability.
	topologyKeyZone = "topology.kubernetes.io/zone"
)

// deploymentTopologySpread returns topology spread constraints that distribute
// deployment pods evenly across availability zones.
//
// The constraints use maxSkew=1 with WhenUnsatisfiable=ScheduleAnyway, meaning
// the scheduler prefers even distribution but won't block scheduling if zones
// are imbalanced. This ensures deployments remain schedulable even in degraded
// cluster states while still achieving zone redundancy under normal conditions.
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

// deploymentAffinity returns pod affinity rules that prefer co-locating deployment
// pods with their environment's sentinel pods in the same availability zone.
//
// This optimization reduces cross-AZ latency between sentinels and the user code
// they proxy to. The affinity is a soft preference (weight=100) rather than a hard
// requirement, so deployments can still schedule if no sentinel-local zones have
// capacity.
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
