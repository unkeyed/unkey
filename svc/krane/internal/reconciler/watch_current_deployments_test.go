package reconciler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestWatchCurrentDeployments_SetupSucceeds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	mockCluster := &MockClusterClient{}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentDeployments(ctx)
	require.NoError(t, err)
}

func TestWatchCurrentDeployments_AddEventTriggersStateUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("replicasets", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentDeployments(ctx)
	require.NoError(t, err)

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicaset",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_123"),
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	fakeWatch.Add(rs)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateDeploymentStateCalls) >= 1
	}, 2*time.Second, 10*time.Millisecond)

	require.NotEmpty(t, mockCluster.UpdateDeploymentStateCalls)
}

func TestWatchCurrentDeployments_ModifyEventTriggersStateUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("replicasets", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentDeployments(ctx)
	require.NoError(t, err)

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicaset",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_123"),
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	fakeWatch.Modify(rs)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateDeploymentStateCalls) >= 1
	}, 2*time.Second, 10*time.Millisecond)
}

func TestWatchCurrentDeployments_DeleteEventTriggersStateUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("replicasets", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentDeployments(ctx)
	require.NoError(t, err)

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicaset",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_123"),
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	fakeWatch.Delete(rs)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateDeploymentStateCalls) >= 1
	}, 2*time.Second, 10*time.Millisecond)

	call := mockCluster.UpdateDeploymentStateCalls[0]
	deleteReq := call.GetDelete()
	require.NotNil(t, deleteReq)
	require.Equal(t, "test-replicaset", deleteReq.GetK8SName())
}

func TestWatchCurrentDeployments_ChannelClosure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("replicasets", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentDeployments(ctx)
	require.NoError(t, err)

	fakeWatch.Stop()

	time.Sleep(100 * time.Millisecond)
}

func TestWatchCurrentDeployments_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("replicasets", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentDeployments(ctx)
	require.NoError(t, err)

	cancel()

	time.Sleep(100 * time.Millisecond)
}

func TestWatchCurrentDeployments_WithPods(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
		},
	}

	client := fake.NewSimpleClientset(pod)
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("replicasets", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentDeployments(ctx)
	require.NoError(t, err)

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicaset",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_123"),
		},
		Spec: appsv1.ReplicaSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
		},
	}

	fakeWatch.Add(rs)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateDeploymentStateCalls) >= 1
	}, 2*time.Second, 10*time.Millisecond)

	call := mockCluster.UpdateDeploymentStateCalls[0]
	update := call.GetUpdate()
	require.NotNil(t, update)
	require.Equal(t, "test-replicaset", update.GetK8SName())
	require.Len(t, update.GetInstances(), 1)
}
