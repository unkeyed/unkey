package reconciler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNew_CreatesReconcilerWithCorrectFields(t *testing.T) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)
	logger := logging.NewNoop()
	mockCluster := &MockClusterClient{}

	cfg := Config{
		ClientSet: client,
		Logger:    logger,
		Cluster:   mockCluster,
		ClusterID: "cluster-123",
		Region:    "us-east-1",
	}

	r := New(cfg)

	require.NotNil(t, r)
	require.Equal(t, client, r.clientSet)
	require.Equal(t, logger, r.logger)
	require.Equal(t, mockCluster, r.cluster)
	require.Equal(t, "us-east-1", r.region)
}

func TestNew_CreatesCircuitBreaker(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   &MockClusterClient{},
		ClusterID: "cluster-123",
		Region:    "us-east-1",
	}

	r := New(cfg)

	require.NotNil(t, r.cb, "circuit breaker should not be nil")
}

func TestNew_CreatesDoneChannel(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   &MockClusterClient{},
		ClusterID: "cluster-123",
		Region:    "us-east-1",
	}

	r := New(cfg)

	require.NotNil(t, r.done, "done channel should not be nil")

	select {
	case <-r.done:
		t.Fatal("done channel should not be closed initially")
	default:
	}
}

func TestStop_ClosesDoneChannel(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   &MockClusterClient{},
		ClusterID: "cluster-123",
		Region:    "us-east-1",
	}

	r := New(cfg)

	err := r.Stop()
	require.NoError(t, err)

	select {
	case <-r.done:
	default:
		t.Fatal("done channel should be closed after Stop")
	}
}

func TestStop_IsIdempotent(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   &MockClusterClient{},
		ClusterID: "cluster-123",
		Region:    "us-east-1",
	}

	r := New(cfg)

	err := r.Stop()
	require.NoError(t, err)

	require.Panics(t, func() {
		_ = r.Stop()
	}, "calling Stop twice should panic when closing already closed channel")
}

func TestStart_InitiatesGoroutines(t *testing.T) {
	client := fake.NewSimpleClientset()
	cfg := Config{
		ClientSet: client,
		Logger:    logging.NewNoop(),
		Cluster:   &MockClusterClient{},
		ClusterID: "cluster-123",
		Region:    "us-east-1",
	}

	r := New(cfg)
	ctx := context.Background()

	err := r.Start(ctx)
	require.NoError(t, err, "Start should return without error")

	err = r.Stop()
	require.NoError(t, err)
}
