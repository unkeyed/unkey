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

	// topologyKeyHostname is the standard Kubernetes label for individual nodes,
	// used for spreading pods across nodes so a single node failure can't take
	// out all of a deployment's replicas.
	topologyKeyHostname = "kubernetes.io/hostname"
)

func deploymentTopologySpread(deploymentID string, maxReplicas uint32) []corev1.TopologySpreadConstraint {
	deploymentSelector := &metav1.LabelSelector{
		MatchLabels: labels.New().DeploymentID(deploymentID),
	}
	fleetSelector := &metav1.LabelSelector{
		MatchLabels: labels.New().
			ManagedByKrane().
			ComponentDeployment(),
	}
	hostname := corev1.TopologySpreadConstraint{
		MaxSkew:           1,
		TopologyKey:       topologyKeyHostname,
		WhenUnsatisfiable: corev1.ScheduleAnyway,
		LabelSelector:     deploymentSelector,
	}
	if maxReplicas > 1 {
		hostname.WhenUnsatisfiable = corev1.DoNotSchedule
		hostname.NodeTaintsPolicy = new(corev1.NodeInclusionPolicyHonor)
		// Without minDomains, a one-node pool permits all replicas on that node.
		hostname.MinDomains = new(int32(2))
	}

	return []corev1.TopologySpreadConstraint{
		hostname,
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     fleetSelector,
		},
	}
}
