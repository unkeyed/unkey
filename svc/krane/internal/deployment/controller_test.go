package deployment

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestNew_CreatesControllerWithCorrectFields(t *testing.T) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	mockCluster := &testutil.MockClusterClient{}

	cfg := Config{
		ClientSet:     client,
		DynamicClient: dynamicClient,
		Cluster:       mockCluster,
		Region:        "us-east-1",
		Fingerprints:  cache.NewNoopCache[string, string](),
	}

	ctrl := New(cfg)

	require.NotNil(t, ctrl)
	require.Equal(t, client, ctrl.clientSet)
	require.Equal(t, dynamicClient, ctrl.dynamicClient)
	require.Equal(t, mockCluster, ctrl.cluster)
	require.Equal(t, "us-east-1", ctrl.region)
}

func TestNew_CreatesOwnCircuitBreaker(t *testing.T) {
	client := fake.NewSimpleClientset()
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	cfg := Config{
		ClientSet:     client,
		DynamicClient: dynamicClient,
		Cluster:       &testutil.MockClusterClient{},
		Region:        "us-east-1",
		Fingerprints:  cache.NewNoopCache[string, string](),
	}

	ctrl := New(cfg)

	require.NotNil(t, ctrl.cb, "circuit breaker should not be nil")
}

func TestRunStopsWatchAndResyncLoops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := fake.NewClientset()
		var lists atomic.Int32
		client.PrependReactor("list", "replicasets", func(ktesting.Action) (bool, runtime.Object, error) {
			lists.Add(1)
			return false, nil, nil
		})

		w := &stubbornWatch{result: make(chan watch.Event)}
		client.PrependWatchReactor("pods", func(ktesting.Action) (bool, watch.Interface, error) {
			return true, w, nil
		})

		ctrl := New(Config{ClientSet: client})
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		done := make(chan struct{})
		go func() {
			ctrl.Run(ctx)
			close(done)
		}()
		synctest.Wait()
		require.Equal(t, int32(2), lists.Load())

		cancel()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("controller did not stop after cancellation")
		}
		require.True(t, w.stopped.Load())

		time.Sleep(2 * time.Minute)
		require.Equal(t, int32(2), lists.Load())
	})
}

func TestActualStateResyncRepairsRecoveryWithoutWatchEvents(t *testing.T) {
	fingerprints, err := cache.New(cache.Config[string, string]{
		Fresh: time.Hour, Stale: time.Hour, MaxSize: 10, Resource: "resync-test", Clock: clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(fingerprints.Close)
	synctest.Test(t, func(t *testing.T) {
		selector := labels.New().ManagedByKrane().ComponentDeployment()
		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "test", Labels: selector},
			Spec:       appsv1.ReplicaSetSpec{Selector: &metav1.LabelSelector{MatchLabels: selector}},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "test", Labels: selector},
			Status: corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{{
				Name: "deployment", RestartCount: 3,
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
			}}},
		}
		client := fake.NewClientset(rs, pod)
		cluster := &testutil.MockClusterClient{}
		var persisted *ctrlv1.ContainerObservation
		cluster.ReportDeploymentStatusFunc = func(_ context.Context, request *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
			if len(cluster.ReportDeploymentStatusCalls) == 2 {
				return nil, errors.New("injected transport failure")
			}
			persisted = request.GetUpdate().GetInstances()[0].GetContainerObservation()
			return &ctrlv1.ReportDeploymentStatusResponse{}, nil
		}
		ctrl := New(Config{ClientSet: client, Cluster: cluster, Fingerprints: fingerprints})
		ctx, cancel := context.WithCancel(t.Context())
		t.Cleanup(cancel)
		done := make(chan struct{})
		go func() {
			ctrl.runActualStateResyncLoop(ctx)
			close(done)
		}()
		synctest.Wait()
		require.Equal(t, "CrashLoopBackOff", persisted.GetWaiting().GetReason())

		pod.Status.ContainerStatuses[0].State = corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
		_, err := client.CoreV1().Pods("test").UpdateStatus(ctx, pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		time.Sleep(30 * time.Second)
		synctest.Wait()
		require.Equal(t, "CrashLoopBackOff", persisted.GetWaiting().GetReason())
		time.Sleep(30 * time.Second)
		synctest.Wait()
		require.NotNil(t, persisted)
		require.Nil(t, persisted.Waiting)
		require.Equal(t, int32(3), persisted.RestartCount)
		time.Sleep(30 * time.Second)
		synctest.Wait()
		require.Len(t, cluster.ReportDeploymentStatusCalls, 4)
		changed, err := ctrl.reportReplicaSet(ctx, rs, false)
		require.NoError(t, err)
		require.False(t, changed)
		require.Len(t, cluster.ReportDeploymentStatusCalls, 4)
		pod.Status.ContainerStatuses[0].State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ErrImagePull"}}
		_, err = client.CoreV1().Pods("test").UpdateStatus(ctx, pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		changed, err = ctrl.reportReplicaSet(ctx, rs, false)
		require.NoError(t, err)
		require.True(t, changed)
		require.Equal(t, "ErrImagePull", persisted.GetWaiting().GetReason())
		require.Empty(t, cluster.ReportInstanceEventsCalls)
		cancel()
		<-done
	})
}
