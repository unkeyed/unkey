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

type deleteSentinelTestHarness struct {
	reconciler    *Reconciler
	client        *fake.Clientset
	mock          *MockClusterClient
	deleteActions []string
}

func newDeleteSentinelTestHarness(t *testing.T, objects ...runtime.Object) *deleteSentinelTestHarness {
	t.Helper()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	allObjects := append([]runtime.Object{namespace}, objects...)
	client := fake.NewSimpleClientset(allObjects...)

	mock := &MockClusterClient{}

	h := &deleteSentinelTestHarness{
		client:        client,
		mock:          mock,
		deleteActions: []string{},
		reconciler: &Reconciler{
			clientSet: client,
			cluster:   mock,
			cb:        circuitbreaker.New[any]("test"),
			logger:    logging.NewNoop(),
			region:    "test-region",
		},
	}

	client.PrependReactor("delete", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(k8stesting.DeleteAction)
		h.deleteActions = append(h.deleteActions, deleteAction.GetResource().Resource)
		return false, nil, nil
	})

	return h
}

func newDeleteSentinelRequest() *ctrlv1.DeleteSentinel {
	return &ctrlv1.DeleteSentinel{
		K8SNamespace: "test-namespace",
		K8SName:      "test-sentinel",
	}
}

func TestDeleteSentinel_SuccessfullyDeletesServiceAndReplicaSet(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	h := newDeleteSentinelTestHarness(t, svc, rs)

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.NoError(t, err)

	_, err = h.client.CoreV1().Services("test-namespace").Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "Service should be deleted")

	_, err = h.client.AppsV1().ReplicaSets("test-namespace").Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "ReplicaSet should be deleted")
}

func TestDeleteSentinel_DeletesServiceBeforeReplicaSet(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	h := newDeleteSentinelTestHarness(t, svc, rs)

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.NoError(t, err)

	require.Len(t, h.deleteActions, 2)
	require.Equal(t, "services", h.deleteActions[0], "Service should be deleted first")
	require.Equal(t, "replicasets", h.deleteActions[1], "ReplicaSet should be deleted second")
}

func TestDeleteSentinel_IgnoresNotFoundOnService(t *testing.T) {
	ctx := context.Background()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	h := newDeleteSentinelTestHarness(t, rs)

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.NoError(t, err, "should not error when Service doesn't exist")

	_, err = h.client.AppsV1().ReplicaSets("test-namespace").Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "ReplicaSet should still be deleted")
}

func TestDeleteSentinel_IgnoresNotFoundOnReplicaSet(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	h := newDeleteSentinelTestHarness(t, svc)

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.NoError(t, err, "should not error when ReplicaSet doesn't exist")

	_, err = h.client.CoreV1().Services("test-namespace").Get(ctx, "test-sentinel", metav1.GetOptions{})
	require.True(t, apierrors.IsNotFound(err), "Service should still be deleted")
}

func TestDeleteSentinel_IgnoresNotFoundOnBoth(t *testing.T) {
	ctx := context.Background()
	h := newDeleteSentinelTestHarness(t)

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.NoError(t, err, "should not error when neither Service nor ReplicaSet exist")
}

func TestDeleteSentinel_PropagatesServiceError(t *testing.T) {
	ctx := context.Background()
	h := newDeleteSentinelTestHarness(t)

	h.client.PrependReactor("delete", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewForbidden(
			schema.GroupResource{Group: "", Resource: "services"},
			"test-sentinel",
			fmt.Errorf("access denied"),
		)
	})

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.Error(t, err)
	require.True(t, apierrors.IsForbidden(err), "should propagate forbidden error from Service deletion")
}

func TestDeleteSentinel_PropagatesReplicaSetError(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	h := newDeleteSentinelTestHarness(t, svc)

	h.client.PrependReactor("delete", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, apierrors.NewInternalError(fmt.Errorf("internal server error"))
	})

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.Error(t, err)
	require.True(t, apierrors.IsInternalError(err), "should propagate internal server error from ReplicaSet deletion")
}

func TestDeleteSentinel_CallsUpdateSentinelStateWithZeroReplicas(t *testing.T) {
	ctx := context.Background()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "test-namespace",
		},
	}
	h := newDeleteSentinelTestHarness(t, svc, rs)

	req := newDeleteSentinelRequest()
	err := h.reconciler.DeleteSentinel(ctx, req)
	require.NoError(t, err)

	require.Len(t, h.mock.UpdateSentinelStateCalls, 1)
	call := h.mock.UpdateSentinelStateCalls[0]

	require.Equal(t, "test-sentinel", call.GetK8SName())
	require.Equal(t, int32(0), call.GetAvailableReplicas())
}
