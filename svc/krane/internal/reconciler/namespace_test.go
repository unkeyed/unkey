package reconciler

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestEnsureNamespaceExists_CreatesNamespaceIfMissing(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	var createdNamespace *corev1.Namespace
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		createdNamespace = createAction.GetObject().(*corev1.Namespace)
		return false, createdNamespace, nil
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "new-namespace")
	require.NoError(t, err)
	require.NotNil(t, createdNamespace, "namespace should be created")
	require.Equal(t, "new-namespace", createdNamespace.Name)
}

func TestEnsureNamespaceExists_IdempotentWhenNamespaceExists(t *testing.T) {
	ctx := context.Background()

	existingNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "existing-namespace",
		},
	}
	client := fake.NewSimpleClientset(existingNamespace)

	var createCount int
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createCount++
		return true, nil, errors.NewAlreadyExists(schema.GroupResource{Group: "", Resource: "namespaces"}, "existing-namespace")
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "existing-namespace")
	require.NoError(t, err, "should not error when namespace already exists")
	require.Equal(t, 1, createCount, "should attempt creation exactly once")

	err = r.ensureNamespaceExists(ctx, "existing-namespace")
	require.NoError(t, err, "should remain idempotent on repeated calls")
}

func TestEnsureNamespaceExists_HandlesCreationError(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	expectedErr := fmt.Errorf("k8s API unavailable")
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, expectedErr
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "test-namespace")
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

func TestEnsureNamespaceExists_CorrectNamespaceMetadata(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	var createdNamespace *corev1.Namespace
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		createdNamespace = createAction.GetObject().(*corev1.Namespace)
		return false, createdNamespace, nil
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "my-namespace")
	require.NoError(t, err)

	require.NotNil(t, createdNamespace)
	require.Equal(t, "my-namespace", createdNamespace.Name)
}
