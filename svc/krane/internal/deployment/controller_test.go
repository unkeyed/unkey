package deployment

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
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

func TestRunStopsLoopsAndPodEventHandler(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d, revision, pod := podEventFixture(t)
		var reports atomic.Int32
		client := fake.NewClientset(d, revision, pod)
		ctrl := New(Config{
			ClientSet: client,
			Cluster: &testutil.MockClusterClient{
				ReportDeploymentStatusFunc: func(context.Context, *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
					reports.Add(1)
					return &ctrlv1.ReportDeploymentStatusResponse{}, nil
				},
			},
			Fingerprints: cache.NewNoopCache[string, string](),
		})
		cachesCtx, stopCaches := context.WithCancel(t.Context())
		defer func() {
			stopCaches()
			ctrl.informers.Shutdown()
		}()
		require.NoError(t, ctrl.StartCaches(cachesCtx))

		ctx, cancel := context.WithCancel(cachesCtx)
		done := make(chan struct{})
		go func() {
			ctrl.Run(ctx)
			close(done)
		}()
		synctest.Wait()
		require.Positive(t, reports.Load(), "a new leader must report the cached pods")

		cancel()
		synctest.Wait()
		select {
		case <-done:
		default:
			t.Fatal("controller did not stop after cancellation")
		}

		reported := reports.Load()
		require.NoError(t, client.CoreV1().Pods(testNamespace).Delete(t.Context(), pod.Name, metav1.DeleteOptions{}))
		time.Sleep(2 * time.Minute)
		synctest.Wait()
		require.Equal(t, reported, reports.Load(), "a stopped controller must not report pod events")
	})
}
