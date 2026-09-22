package deployment

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
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
