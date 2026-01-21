package deployment

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
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
	logger := logging.NewNoop()
	mockCluster := &testutil.MockClusterClient{}

	cfg := Config{
		ClientSet:     client,
		DynamicClient: dynamicClient,
		Logger:        logger,
		Cluster:       mockCluster,
		Region:        "us-east-1",
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
		Logger:        logging.NewNoop(),
		Cluster:       &testutil.MockClusterClient{},
		Region:        "us-east-1",
	}

	ctrl := New(cfg)

	require.NotNil(t, ctrl.cb, "circuit breaker should not be nil")
}

func TestNew_InitializesVersionCursorToZero(t *testing.T) {
	client := fake.NewSimpleClientset()
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	cfg := Config{
		ClientSet:     client,
		DynamicClient: dynamicClient,
		Logger:        logging.NewNoop(),
		Cluster:       &testutil.MockClusterClient{},
		Region:        "us-east-1",
	}

	ctrl := New(cfg)

	require.Equal(t, uint64(0), ctrl.versionLastSeen, "version cursor should start at 0")
}

func TestNew_CreatesDoneChannel(t *testing.T) {
	client := fake.NewSimpleClientset()
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	cfg := Config{
		ClientSet:     client,
		DynamicClient: dynamicClient,
		Logger:        logging.NewNoop(),
		Cluster:       &testutil.MockClusterClient{},
		Region:        "us-east-1",
	}

	ctrl := New(cfg)

	require.NotNil(t, ctrl.done, "done channel should not be nil")

	select {
	case <-ctrl.done:
		t.Fatal("done channel should not be closed initially")
	default:
	}
}

func TestStop_ClosesDoneChannel(t *testing.T) {
	client := fake.NewSimpleClientset()
	dynamicClient := fakedynamic.NewSimpleDynamicClient(runtime.NewScheme())
	cfg := Config{
		ClientSet:     client,
		DynamicClient: dynamicClient,
		Logger:        logging.NewNoop(),
		Cluster:       &testutil.MockClusterClient{},
		Region:        "us-east-1",
	}

	ctrl := New(cfg)

	err := ctrl.Stop()
	require.NoError(t, err)

	select {
	case <-ctrl.done:
	default:
		t.Fatal("done channel should be closed after Stop")
	}
}
