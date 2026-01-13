package reconciler

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

type deleteDeploymentTestHarness struct {
	reconciler *Reconciler
	client     *fake.Clientset
	mock       *MockClusterClient
}

func newDeleteDeploymentTestHarness(t *testing.T, objects ...runtime.Object) *deleteDeploymentTestHarness {
	t.Helper()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	allObjects := append([]runtime.Object{namespace}, objects...)
	client := fake.NewSimpleClientset(allObjects...)

	mock := &MockClusterClient{}

	h := &deleteDeploymentTestHarness{
		client: client,
		mock:   mock,
		reconciler: &Reconciler{
			clientSet: client,
			cluster:   mock,
			cb:        circuitbreaker.New[any]("test"),
			logger:    logging.NewNoop(),
			region:    "test-region",
		},
	}

	return h
}

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
	h := newDeleteDeploymentTestHarness(t, rs)

	req := newDeleteDeploymentRequest()
	err := h.reconciler.DeleteDeployment(ctx, req)
	require.NoError(t, err)

	_, err = h.client.AppsV1().ReplicaSets("test-namespace").Get(ctx, "test-deployment", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "ReplicaSet should be deleted")
}

func TestDeleteDeployment_IgnoresNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	h := newDeleteDeploymentTestHarness(t)

	req := newDeleteDeploymentRequest()
	err := h.reconciler.DeleteDeployment(ctx, req)
	require.NoError(t, err, "should not error when ReplicaSet doesn't exist")
}

func TestDeleteDeployment_PropagatesForbiddenError(t *testing.T) {
	ctx := context.Background()
	h := newDeleteDeploymentTestHarness(t)

	h.client.PrependReactor("delete", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "apps", Resource: "replicasets"},
			"test-deployment",
			fmt.Errorf("access denied"),
		)
	})

	req := newDeleteDeploymentRequest()
	err := h.reconciler.DeleteDeployment(ctx, req)
	require.Error(t, err)
	require.True(t, apierrors.IsForbidden(err), "should propagate forbidden error")
}

func TestDeleteDeployment_PropagatesInternalServerError(t *testing.T) {
	ctx := context.Background()
	h := newDeleteDeploymentTestHarness(t)

	h.client.PrependReactor("delete", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewInternalError(fmt.Errorf("internal server error"))
	})

	req := newDeleteDeploymentRequest()
	err := h.reconciler.DeleteDeployment(ctx, req)
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
	h := newDeleteDeploymentTestHarness(t, rs)

	req := newDeleteDeploymentRequest()
	err := h.reconciler.DeleteDeployment(ctx, req)
	require.NoError(t, err)

	require.Len(t, h.mock.UpdateDeploymentStateCalls, 1)
	call := h.mock.UpdateDeploymentStateCalls[0]

	deleteChange, ok := call.GetChange().(*ctrlv1.UpdateDeploymentStateRequest_Delete_)
	require.True(t, ok, "Change should be a Delete type")
	require.Equal(t, "test-deployment", deleteChange.Delete.GetK8SName())
}
