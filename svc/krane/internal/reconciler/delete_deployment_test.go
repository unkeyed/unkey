package reconciler

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newDeleteDeploymentRequest() *ctrlv1.DeleteDeployment {
	return &ctrlv1.DeleteDeployment{
		K8SNamespace: "test-namespace",
		K8SName:      "test-deployment",
	}
}

func TestDeleteDeployment_SuccessfullyDeletesReplicaSet(t *testing.T) {
	ctx := context.Background()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
	}
	client := NewFakeClient(t, rs)
	r := NewTestReconciler(client, nil)

	req := newDeleteDeploymentRequest()
	err := r.DeleteDeployment(ctx, req)
	require.NoError(t, err)

	_, err = client.AppsV1().ReplicaSets("test-namespace").Get(ctx, "test-deployment", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "ReplicaSet should be deleted")
}

func TestDeleteDeployment_IgnoresNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	r := NewTestReconciler(client, nil)

	req := newDeleteDeploymentRequest()
	err := r.DeleteDeployment(ctx, req)
	require.NoError(t, err, "should not error when ReplicaSet doesn't exist")
}

func TestDeleteDeployment_PropagatesForbiddenError(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)

	forbiddenErr := apierrors.NewForbidden(
		schema.GroupResource{Group: "apps", Resource: "replicasets"},
		"test-deployment",
		fmt.Errorf("access denied"),
	)
	AddErrorReactor(client, "delete", "replicasets", forbiddenErr)

	r := NewTestReconciler(client, nil)

	req := newDeleteDeploymentRequest()
	err := r.DeleteDeployment(ctx, req)
	require.Error(t, err)
	require.True(t, apierrors.IsForbidden(err), "should propagate forbidden error")
}

func TestDeleteDeployment_PropagatesInternalServerError(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)

	internalErr := apierrors.NewInternalError(fmt.Errorf("internal server error"))
	AddErrorReactor(client, "delete", "replicasets", internalErr)

	r := NewTestReconciler(client, nil)

	req := newDeleteDeploymentRequest()
	err := r.DeleteDeployment(ctx, req)
	require.Error(t, err)
	require.True(t, apierrors.IsInternalError(err), "should propagate internal server error")
}

func TestDeleteDeployment_CallsUpdateDeploymentStateWithDeleteChange(t *testing.T) {
	ctx := context.Background()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
	}
	client := NewFakeClient(t, rs)
	mockCluster := &MockClusterClient{}
	r := NewTestReconciler(client, mockCluster)

	req := newDeleteDeploymentRequest()
	err := r.DeleteDeployment(ctx, req)
	require.NoError(t, err)

	require.Len(t, mockCluster.UpdateDeploymentStateCalls, 1)
	call := mockCluster.UpdateDeploymentStateCalls[0]

	deleteChange, ok := call.GetChange().(*ctrlv1.UpdateDeploymentStateRequest_Delete_)
	require.True(t, ok, "Change should be a Delete type")
	require.Equal(t, "test-deployment", deleteChange.Delete.GetK8SName())
}
