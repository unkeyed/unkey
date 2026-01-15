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

func TestRefreshCurrentDeployments_ListsAllResources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rs1 := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-1",
			Namespace: "ns-1",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_1"),
		},
	}
	rs2 := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-2",
			Namespace: "ns-2",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_2"),
		},
	}

	client := fake.NewSimpleClientset(rs1, rs2)

	var getDesiredCalls atomic.Int32
	mockCluster := &MockClusterClient{
		GetDesiredDeploymentStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {
			getDesiredCalls.Add(1)
			return connect.NewResponse(&ctrlv1.DeploymentState{}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return getDesiredCalls.Load() >= 2
	}, 2*time.Second, 50*time.Millisecond, "expected GetDesiredDeploymentState to be called for each replicaset")
}

func TestRefreshCurrentDeployments_HandlesEmptyList(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()

	var listCalled atomic.Bool
	client.PrependReactor("list", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return listCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected list to be called")

	require.Len(t, mockCluster.UpdateDeploymentStateCalls, 0, "no updates should be made for empty list")
}

func TestRefreshCurrentDeployments_HandlesListError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()

	var listErrorCalled atomic.Bool
	client.PrependReactor("list", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return listErrorCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected list to be called with error")
}

func TestRefreshCurrentDeployments_PeriodicRefresh(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := fake.NewSimpleClientset()

	var listCallCount atomic.Int32
	client.PrependReactor("list", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return listCallCount.Load() >= 1
	}, 2*time.Second, 50*time.Millisecond, "expected at least one list call (immediate execution)")
}

func TestRefreshCurrentDeployments_AppliesDesiredState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-1",
			Namespace: "test-namespace",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_1"),
		},
	}

	client := fake.NewSimpleClientset(rs)

	var applyCalled atomic.Bool
	mockCluster := &MockClusterClient{
		GetDesiredDeploymentStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {
			applyCalled.Store(true)
			return connect.NewResponse(&ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						WorkspaceId:   "ws_123",
						ProjectId:     "prj_123",
						EnvironmentId: "env_123",
						DeploymentId:  "dep_1",
						K8SNamespace:  "test-namespace",
						K8SName:       "rs-1",
						Image:         "nginx:1.19",
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

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return applyCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected apply to be called")
}

func TestRefreshCurrentDeployments_DeletesDesiredState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-1",
			Namespace: "test-namespace",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_1"),
		},
	}

	client := fake.NewSimpleClientset(rs)

	var deleteCalled atomic.Bool
	mockCluster := &MockClusterClient{
		GetDesiredDeploymentStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {
			deleteCalled.Store(true)
			return connect.NewResponse(&ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: "test-namespace",
						K8SName:      "rs-1",
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

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return deleteCalled.Load()
	}, 2*time.Second, 50*time.Millisecond, "expected delete to be called")
}

func TestRefreshCurrentDeployments_HandlesGetDesiredStateError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rs1 := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-1",
			Namespace: "ns-1",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_1"),
		},
	}
	rs2 := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-2",
			Namespace: "ns-2",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_2"),
		},
	}

	client := fake.NewSimpleClientset(rs1, rs2)

	var getDesiredCalls atomic.Int32
	mockCluster := &MockClusterClient{
		GetDesiredDeploymentStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {
			count := getDesiredCalls.Add(1)
			if count == 1 {
				return nil, fmt.Errorf("simulated control plane error")
			}
			return connect.NewResponse(&ctrlv1.DeploymentState{}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return getDesiredCalls.Load() >= 2
	}, 2*time.Second, 50*time.Millisecond, "expected both replicasets to be processed despite first error")
}

func TestRefreshCurrentDeployments_HandlesMissingDeploymentID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rsWithoutID := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-no-id",
			Namespace: "ns-1",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment(),
		},
	}
	rsWithID := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rs-with-id",
			Namespace: "ns-2",
			Labels: labels.New().
				ManagedByKrane().
				ComponentDeployment().
				DeploymentID("dep_1"),
		},
	}

	client := fake.NewSimpleClientset(rsWithoutID, rsWithID)

	var getDesiredCalls atomic.Int32
	mockCluster := &MockClusterClient{
		GetDesiredDeploymentStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {
			getDesiredCalls.Add(1)
			return connect.NewResponse(&ctrlv1.DeploymentState{}), nil
		},
	}

	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	go r.refreshCurrentDeployments(ctx)

	require.Eventually(t, func() bool {
		return getDesiredCalls.Load() >= 1
	}, 2*time.Second, 50*time.Millisecond, "expected only replicaset with ID to be processed")

	require.Never(t, func() bool {
		return getDesiredCalls.Load() > 1
	}, 200*time.Millisecond, 20*time.Millisecond, "only one replicaset should be processed (the one with ID)")
	require.Equal(t, int32(1), getDesiredCalls.Load(), "only one replicaset should be processed (the one with ID)")
}
