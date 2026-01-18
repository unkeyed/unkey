package reconciler

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newDeleteSentinelRequest() *ctrlv1.DeleteSentinel {
	return &ctrlv1.DeleteSentinel{
		K8SName: "test-sentinel",
	}
}

func TestDeleteSentinel_SuccessfullyDeletesServiceAndDeployment(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	client := NewFakeClient(t, svc, dep)
	r := NewTestReconciler(client, nil)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.NoError(t, err)

	_, err = client.CoreV1().Services(NamespaceSentinel).Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "Service should be deleted")

	_, err = client.AppsV1().Deployments(NamespaceSentinel).Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "Deployment should be deleted")
}

func TestDeleteSentinel_DeletesServiceBeforeDeployment(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	client := NewFakeClient(t, svc, dep)
	deletes := AddDeleteTracker(client)
	r := NewTestReconciler(client, nil)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.NoError(t, err)

	require.Len(t, deletes.Actions, 2)
	require.Equal(t, "services", deletes.Actions[0], "Service should be deleted first")
	require.Equal(t, "deployments", deletes.Actions[1], "Deployment should be deleted second")
}

func TestDeleteSentinel_IgnoresNotFoundOnService(t *testing.T) {
	ctx := context.Background()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	client := NewFakeClient(t, dep)
	r := NewTestReconciler(client, nil)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.NoError(t, err, "should not error when Service doesn't exist")

	_, err = client.AppsV1().Deployments(NamespaceSentinel).Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "Deployment should still be deleted")
}

func TestDeleteSentinel_IgnoresNotFoundOnDeployment(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	client := NewFakeClient(t, svc)
	r := NewTestReconciler(client, nil)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.NoError(t, err, "should not error when Deployment doesn't exist")

	_, err = client.CoreV1().Services(NamespaceSentinel).Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "Service should still be deleted")
}

func TestDeleteSentinel_IgnoresNotFoundOnBoth(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	r := NewTestReconciler(client, nil)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.NoError(t, err, "should not error when neither Service nor Deployment exist")
}

func TestDeleteSentinel_PropagatesServiceError(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)

	forbiddenErr := apierrors.NewForbidden(
		schema.GroupResource{Group: "", Resource: "services"},
		"test-sentinel",
		fmt.Errorf("access denied"),
	)
	AddErrorReactor(client, "delete", "services", forbiddenErr)

	r := NewTestReconciler(client, nil)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.Error(t, err)
	require.True(t, apierrors.IsForbidden(err), "should propagate forbidden error from Service deletion")
}

func TestDeleteSentinel_PropagatesDeploymentError(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	client := NewFakeClient(t, svc)

	internalErr := apierrors.NewInternalError(fmt.Errorf("internal server error"))
	AddErrorReactor(client, "delete", "deployments", internalErr)

	r := NewTestReconciler(client, nil)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.Error(t, err)
	require.True(t, apierrors.IsInternalError(err), "should propagate internal server error from Deployment deletion")
}

func TestDeleteSentinel_CallsUpdateSentinelStateWithZeroReplicas(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: NamespaceSentinel,
		},
	}
	client := NewFakeClient(t, svc, dep)
	mockCluster := &MockClusterClient{}
	r := NewTestReconciler(client, mockCluster)

	req := newDeleteSentinelRequest()
	err := r.DeleteSentinel(ctx, req)
	require.NoError(t, err)

	require.Len(t, mockCluster.UpdateSentinelStateCalls, 1)
	call := mockCluster.UpdateSentinelStateCalls[0]

	require.Equal(t, "test-sentinel", call.GetK8SName())
	require.Equal(t, int32(0), call.GetAvailableReplicas())
}
