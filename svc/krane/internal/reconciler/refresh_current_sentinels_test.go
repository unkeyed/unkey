package reconciler

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestRefreshCurrentSentinels_ListsAllResources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dep1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1",
			Namespace: "ns-1",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_1"),
		},
	}
	dep2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-2",
			Namespace: "ns-2",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_2"),
		},
	}

	client := fake.NewSimpleClientset(dep1, dep2)

	var getDesiredCalls atomic.Int32
	mockCluster := &MockClusterClient{
		GetDesiredSentinelStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {
			getDesiredCalls.Add(1)
			return connect.NewResponse(&ctrlv1.SentinelState{}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return getDesiredCalls.Load() >= 2
	}, 2*time.Second, 50*time.Millisecond, "expected GetDesiredSentinelState to be called for each deployment")
}

func TestRefreshCurrentSentinels_HandlesEmptyList(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()

	var listCalled atomic.Bool
	client.PrependReactor("list", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		listCalled.Store(true)
		return false, nil, nil
	})

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return listCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected list to be called")

	require.Len(t, mockCluster.UpdateSentinelStateCalls, 0, "no updates should be made for empty list")
}

func TestRefreshCurrentSentinels_HandlesListError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()

	var listErrorCalled atomic.Bool
	client.PrependReactor("list", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		listErrorCalled.Store(true)
		return true, nil, fmt.Errorf("simulated list error")
	})

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return listErrorCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected list to be called with error")
}

func TestRefreshCurrentSentinels_PeriodicRefresh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()

	var listCallCount atomic.Int32
	client.PrependReactor("list", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		listCallCount.Add(1)
		return false, nil, nil
	})

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return listCallCount.Load() >= 1
	}, 2*time.Second, 50*time.Millisecond, "expected at least one list call (immediate execution)")
}

func TestRefreshCurrentSentinels_AppliesDesiredState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1",
			Namespace: "test-namespace",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_1"),
		},
	}

	client := fake.NewSimpleClientset(dep)

	var applyCalled atomic.Bool
	mockCluster := &MockClusterClient{
		GetDesiredSentinelStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {
			applyCalled.Store(true)
			return connect.NewResponse(&ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Apply{
					Apply: &ctrlv1.ApplySentinel{
						WorkspaceId:   "ws_123",
						ProjectId:     "prj_123",
						EnvironmentId: "env_123",
						SentinelId:    "sent_1",
						K8SNamespace:  "test-namespace",
						K8SName:       "dep-1",
						Image:         "unkey/sentinel:v1.0",
						Replicas:      1,
						CpuMillicores: 100,
						MemoryMib:     128,
					},
				},
			}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return applyCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected apply to be called")
}

func TestRefreshCurrentSentinels_DeletesDesiredState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1",
			Namespace: "test-namespace",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_1"),
		},
	}

	client := fake.NewSimpleClientset(dep)

	var deleteCalled atomic.Bool
	mockCluster := &MockClusterClient{
		GetDesiredSentinelStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {
			deleteCalled.Store(true)
			return connect.NewResponse(&ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Delete{
					Delete: &ctrlv1.DeleteSentinel{
						K8SNamespace: "test-namespace",
						K8SName:      "dep-1",
					},
				},
			}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return deleteCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected delete to be called")
}

func TestRefreshCurrentSentinels_HandlesGetDesiredStateError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dep1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-1",
			Namespace: "ns-1",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_1"),
		},
	}
	dep2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-2",
			Namespace: "ns-2",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_2"),
		},
	}

	client := fake.NewSimpleClientset(dep1, dep2)

	var getDesiredCalls atomic.Int32
	mockCluster := &MockClusterClient{
		GetDesiredSentinelStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {
			count := getDesiredCalls.Add(1)
			if count == 1 {
				return nil, fmt.Errorf("simulated control plane error")
			}
			return connect.NewResponse(&ctrlv1.SentinelState{}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return getDesiredCalls.Load() >= 2
	}, 2*time.Second, 50*time.Millisecond, "expected both deployments to be processed despite first error")
}

func TestRefreshCurrentSentinels_HandlesMissingSentinelID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	depWithoutID := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-no-id",
			Namespace: "ns-1",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel(),
		},
	}
	depWithID := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dep-with-id",
			Namespace: "ns-2",
			Labels: labels.New().
				ManagedByKrane().
				ComponentSentinel().
				SentinelID("sent_1"),
		},
	}

	client := fake.NewSimpleClientset(depWithoutID, depWithID)

	var getDesiredCalls atomic.Int32
	mockCluster := &MockClusterClient{
		GetDesiredSentinelStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {
			getDesiredCalls.Add(1)
			return connect.NewResponse(&ctrlv1.SentinelState{}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentSentinels(ctx)

	require.Eventually(t, func() bool {
		return getDesiredCalls.Load() >= 1
	}, 2*time.Second, 50*time.Millisecond, "expected only deployment with ID to be processed")

	require.Never(t, func() bool {
		return getDesiredCalls.Load() > 1
	}, 200*time.Millisecond, 20*time.Millisecond, "only one deployment should be processed (the one with ID)")
	require.Equal(t, int32(1), getDesiredCalls.Load(), "only one deployment should be processed (the one with ID)")
}
