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

// minHostnameDomains is the number of nodes a scalable deployment must spread
// across, so a single node failure removes at most a third of its replicas.
const minHostnameDomains = 3

// deploymentTopologySpread returns the topology spread constraints for a
// deployment's pods.
//
// Deployments that can scale past one replica get a hard hostname constraint
// that allows at most ceil(maxReplicas/3) replicas on one node. A hard
// maxSkew=1 would force one node per replica, because skew is measured against
// full nodes that hold no replica too. That makes Karpenter add nodes even when
// existing nodes have room. A soft maxSkew=1 hostname constraint still prefers
// one replica per node when there is room.
func deploymentTopologySpread(deploymentID string, maxReplicas uint32) []corev1.TopologySpreadConstraint {
	deploymentSelector := &metav1.LabelSelector{
		MatchLabels: labels.New().DeploymentID(deploymentID),
	}
	fleetSelector := &metav1.LabelSelector{
		MatchLabels: labels.New().
			ManagedByKrane().
			ComponentDeployment(),
	}

	constraints := []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyHostname,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     deploymentSelector,
		},
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector:     fleetSelector,
		},
	}
	if maxReplicas > 1 {
		constraints = append(constraints, corev1.TopologySpreadConstraint{
			MaxSkew:           int32((maxReplicas + minHostnameDomains - 1) / minHostnameDomains),
			TopologyKey:       topologyKeyHostname,
			WhenUnsatisfiable: corev1.DoNotSchedule,
			LabelSelector:     deploymentSelector,
			// Without minDomains, a pool with fewer nodes permits all replicas
			// on those nodes.
			MinDomains:       new(int32(minHostnameDomains)),
			NodeTaintsPolicy: new(corev1.NodeInclusionPolicyHonor),
		})
	}

	return constraints
}
