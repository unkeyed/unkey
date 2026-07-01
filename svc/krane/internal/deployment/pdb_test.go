package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestBuildPodDisruptionBudget(t *testing.T) {
	req := &ctrlv1.ApplyDeployment{
		WorkspaceId:   "ws_123",
		ProjectId:     "proj_123",
		AppId:         "app_123",
		EnvironmentId: "env_123",
		DeploymentId:  "dep_123",
		K8SName:       "dep-123",
		K8SNamespace:  "customer-ns",
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dep-123",
			UID:  types.UID("rs-uid-123"),
		},
	}

	pdb := buildPodDisruptionBudget(req, rs)

	// maxUnavailable must be the absolute integer 1, never a percentage: a
	// percentage rounds down and would compute 0 allowed disruptions for a
	// single-replica deployment, deadlocking node drains.
	require.NotNil(t, pdb.Spec.MaxUnavailable)
	require.Equal(t, intstr.Int, pdb.Spec.MaxUnavailable.Type)
	require.Equal(t, int32(1), pdb.Spec.MaxUnavailable.IntVal)
	require.Nil(t, pdb.Spec.MinAvailable, "minAvailable would deadlock single-replica drains")

	// Selects exactly the deployment's pods by deployment ID.
	require.Equal(t,
		map[string]string{labels.LabelKeyDeploymentID: "dep_123"},
		pdb.Spec.Selector.MatchLabels,
	)

	// Owned by the ReplicaSet so it is garbage-collected with the deployment.
	require.Len(t, pdb.OwnerReferences, 1)
	owner := pdb.OwnerReferences[0]
	require.Equal(t, "ReplicaSet", owner.Kind)
	require.Equal(t, rs.Name, owner.Name)
	require.Equal(t, rs.UID, owner.UID)
	require.NotNil(t, owner.Controller)
	require.True(t, *owner.Controller)

	require.Equal(t, req.GetK8SName(), pdb.Name)
	require.Equal(t, req.GetK8SNamespace(), pdb.Namespace)
	require.Equal(t, "krane", pdb.Labels[labels.LabelKeyManagedBy])
}
