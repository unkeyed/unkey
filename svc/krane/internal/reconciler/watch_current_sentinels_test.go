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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestWatchCurrentSentinels_SetupSucceeds(t *testing.T) {
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

	err := r.watchCurrentSentinels(ctx)
	require.NoError(t, err)
}

func TestWatchCurrentSentinels_AddEventTriggersStateUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("deployments", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentSentinels(ctx)
	require.NoError(t, err)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_123"),
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
		},
	}

	fakeWatch.Add(deployment)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateSentinelStateCalls) >= 1
	}, 2*time.Second, 10*time.Millisecond)

	call := mockCluster.UpdateSentinelStateCalls[0]
	require.Equal(t, "test-sentinel", call.GetK8SName())
	require.Equal(t, int32(3), call.GetAvailableReplicas())
}

func TestWatchCurrentSentinels_ModifyEventTriggersStateUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("deployments", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentSentinels(ctx)
	require.NoError(t, err)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_123"),
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 5,
		},
	}

	fakeWatch.Modify(deployment)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateSentinelStateCalls) >= 1
	}, 2*time.Second, 10*time.Millisecond)

	call := mockCluster.UpdateSentinelStateCalls[0]
	require.Equal(t, "test-sentinel", call.GetK8SName())
	require.Equal(t, int32(5), call.GetAvailableReplicas())
}

func TestWatchCurrentSentinels_DeleteEventTriggersStateUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("deployments", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentSentinels(ctx)
	require.NoError(t, err)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sentinel",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_123"),
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 3,
		},
	}

	fakeWatch.Delete(deployment)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateSentinelStateCalls) >= 1
	}, 2*time.Second, 10*time.Millisecond)

	call := mockCluster.UpdateSentinelStateCalls[0]
	require.Equal(t, "test-sentinel", call.GetK8SName())
	require.Equal(t, int32(0), call.GetAvailableReplicas())
}

func TestWatchCurrentSentinels_ChannelClosure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("deployments", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentSentinels(ctx)
	require.NoError(t, err)

	fakeWatch.Stop()
}

func TestWatchCurrentSentinels_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("deployments", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentSentinels(ctx)
	require.NoError(t, err)

	cancel()
}

func TestWatchCurrentSentinels_MultipleEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()
	fakeWatch := watch.NewFake()

	client.PrependWatchReactor("deployments", k8stesting.DefaultWatchReactor(fakeWatch, nil))

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.watchCurrentSentinels(ctx)
	require.NoError(t, err)

	deployment1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentinel-1",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_1"),
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}

	deployment2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sentinel-2",
			Namespace: "default",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_2"),
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 2,
		},
	}

	fakeWatch.Add(deployment1)
	fakeWatch.Add(deployment2)

	require.Eventually(t, func() bool {
		return len(mockCluster.UpdateSentinelStateCalls) >= 2
	}, 2*time.Second, 10*time.Millisecond)

	names := make(map[string]bool)
	for _, call := range mockCluster.UpdateSentinelStateCalls {
		names[call.GetK8SName()] = true
	}
	require.True(t, names["sentinel-1"])
	require.True(t, names["sentinel-2"])
}
