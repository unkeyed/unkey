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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

var ciliumGVR = schema.GroupVersionResource{
	Group:    "cilium.io",
	Version:  "v2",
	Resource: "ciliumnetworkpolicies",
}

// newFakeDynamicClient creates a fake dynamic client for testing CiliumNetworkPolicy operations
func newFakeDynamicClient() *dynamicfake.FakeDynamicClient {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		ciliumGVR: "CiliumNetworkPolicyList",
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
}

// newFakeDynamicClientWithPatchReactor creates a dynamic client that properly handles Apply (patch) operations
func newFakeDynamicClientWithPatchReactor(onPatch func(action k8stesting.Action)) *dynamicfake.FakeDynamicClient {
	client := newFakeDynamicClient()
	client.PrependReactor("patch", "ciliumnetworkpolicies", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		if onPatch != nil {
			onPatch(action)
		}
		patchAction := action.(k8stesting.PatchAction)
		// Return a valid unstructured object for Apply
		obj := &unstructured.Unstructured{}
		obj.SetAPIVersion("cilium.io/v2")
		obj.SetKind("CiliumNetworkPolicy")
		obj.SetName(patchAction.GetName())
		obj.SetNamespace(patchAction.GetNamespace())
		return true, obj, nil
	})
	return client
}

func TestEnsureNamespaceExists_CreatesNamespaceIfMissing(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	dynamicClient := newFakeDynamicClientWithPatchReactor(nil)

	var createdNamespace *corev1.Namespace
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		createdNamespace = createAction.GetObject().(*corev1.Namespace)
		return false, createdNamespace, nil
	})

	r := &Reconciler{
		clientSet:     client,
		dynamicClient: dynamicClient,
		cluster:       &MockClusterClient{},
		cb:            circuitbreaker.New[any]("test"),
		logger:        logging.NewNoop(),
		region:        "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "new-namespace", "ws-123", "env-456")
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
	dynamicClient := newFakeDynamicClientWithPatchReactor(nil)

	var createCount int
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createCount++
		return true, nil, errors.NewAlreadyExists(schema.GroupResource{Group: "", Resource: "namespaces"}, "existing-namespace")
	})

	r := &Reconciler{
		clientSet:     client,
		dynamicClient: dynamicClient,
		cluster:       &MockClusterClient{},
		cb:            circuitbreaker.New[any]("test"),
		logger:        logging.NewNoop(),
		region:        "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "existing-namespace", "ws-123", "env-456")
	require.NoError(t, err, "should not error when namespace already exists")
	require.Equal(t, 1, createCount, "should attempt creation exactly once")

	err = r.ensureNamespaceExists(ctx, "existing-namespace", "ws-123", "env-456")
	require.NoError(t, err, "should remain idempotent on repeated calls")
}

func TestEnsureNamespaceExists_HandlesCreationError(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	dynamicClient := newFakeDynamicClientWithPatchReactor(nil)

	expectedErr := fmt.Errorf("k8s API unavailable")
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, expectedErr
	})

	r := &Reconciler{
		clientSet:     client,
		dynamicClient: dynamicClient,
		cluster:       &MockClusterClient{},
		cb:            circuitbreaker.New[any]("test"),
		logger:        logging.NewNoop(),
		region:        "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "test-namespace", "ws-123", "env-456")
	require.Error(t, err)
	require.Equal(t, expectedErr, err)
}

func TestEnsureNamespaceExists_CorrectNamespaceMetadata(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	dynamicClient := newFakeDynamicClientWithPatchReactor(nil)

	var createdNamespace *corev1.Namespace
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		createAction := action.(k8stesting.CreateAction)
		createdNamespace = createAction.GetObject().(*corev1.Namespace)
		return false, createdNamespace, nil
	})

	r := &Reconciler{
		clientSet:     client,
		dynamicClient: dynamicClient,
		cluster:       &MockClusterClient{},
		cb:            circuitbreaker.New[any]("test"),
		logger:        logging.NewNoop(),
		region:        "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "my-namespace", "ws-123", "env-456")
	require.NoError(t, err)

	require.NotNil(t, createdNamespace)
	require.Equal(t, "my-namespace", createdNamespace.Name)
}

func TestEnsureNamespaceExists_CreatesCiliumPolicyForCustomerNamespace(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	var createdPolicy *unstructured.Unstructured
	dynamicClient := newFakeDynamicClientWithPatchReactor(func(action k8stesting.Action) {
		patchAction := action.(k8stesting.PatchAction)
		createdPolicy = &unstructured.Unstructured{}
		createdPolicy.SetName(patchAction.GetName())
		createdPolicy.SetNamespace(patchAction.GetNamespace())
	})

	r := &Reconciler{
		clientSet:     client,
		dynamicClient: dynamicClient,
		cluster:       &MockClusterClient{},
		cb:            circuitbreaker.New[any]("test"),
		logger:        logging.NewNoop(),
		region:        "test-region",
	}

	err := r.ensureNamespaceExists(ctx, "customer-namespace", "ws-123", "env-456")
	require.NoError(t, err)
	require.NotNil(t, createdPolicy, "CiliumNetworkPolicy should be created")
	require.Equal(t, "allow-sentinel-ingress", createdPolicy.GetName())
	require.Equal(t, "customer-namespace", createdPolicy.GetNamespace())
}

func TestEnsureNamespaceExists_SkipsCiliumPolicyForSentinelNamespace(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	var policyCreated bool
	dynamicClient := newFakeDynamicClientWithPatchReactor(func(action k8stesting.Action) {
		policyCreated = true
	})

	r := &Reconciler{
		clientSet:     client,
		dynamicClient: dynamicClient,
		cluster:       &MockClusterClient{},
		cb:            circuitbreaker.New[any]("test"),
		logger:        logging.NewNoop(),
		region:        "test-region",
	}

	// Sentinel namespace should NOT create a CiliumNetworkPolicy
	err := r.ensureNamespaceExists(ctx, NamespaceSentinel, "", "")
	require.NoError(t, err)
	require.False(t, policyCreated, "CiliumNetworkPolicy should NOT be created for sentinel namespace")
}

func TestApplyCiliumPolicyForNamespace_CorrectPolicySpec(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()

	var appliedPatch []byte
	dynamicClient := newFakeDynamicClientWithPatchReactor(func(action k8stesting.Action) {
		patchAction := action.(k8stesting.PatchAction)
		appliedPatch = patchAction.GetPatch()
	})

	r := &Reconciler{
		clientSet:     client,
		dynamicClient: dynamicClient,
		cluster:       &MockClusterClient{},
		cb:            circuitbreaker.New[any]("test"),
		logger:        logging.NewNoop(),
		region:        "test-region",
	}

	err := r.applyCiliumPolicyForNamespace(ctx, "test-namespace", "ws-test", "env-test")
	require.NoError(t, err)
	require.NotEmpty(t, appliedPatch, "patch should be applied")

	// Verify the patch contains the expected values
	patchStr := string(appliedPatch)
	require.Contains(t, patchStr, "allow-sentinel-ingress")
	require.Contains(t, patchStr, "ws-test")
	require.Contains(t, patchStr, "env-test")
	require.Contains(t, patchStr, "8080")
}
