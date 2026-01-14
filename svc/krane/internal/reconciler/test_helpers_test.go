package reconciler

import (
	"encoding/json"
	"testing"

	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// -----------------------------------------------------------------------------
// Fake Client Setup
// -----------------------------------------------------------------------------

// NewFakeClient creates a fake Kubernetes client pre-seeded with a "test-namespace"
// namespace and any additional objects passed as arguments.
func NewFakeClient(t *testing.T, objects ...runtime.Object) *fake.Clientset {
	t.Helper()
	allObjects := append([]runtime.Object{newTestNamespace()}, objects...)
	return fake.NewSimpleClientset(allObjects...)
}

// NewFakeClientWithoutNamespace creates a fake Kubernetes client with only the
// provided objects. Use this when testing namespace creation behavior.
func NewFakeClientWithoutNamespace(t *testing.T, objects ...runtime.Object) *fake.Clientset {
	t.Helper()
	return fake.NewSimpleClientset(objects...)
}

func newTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
}

// -----------------------------------------------------------------------------
// Reconciler Setup
// -----------------------------------------------------------------------------

// NewTestReconciler creates a Reconciler with the provided fake client and mock
// control plane client. If controlPlane is nil, a new MockClusterClient is used.
func NewTestReconciler(client *fake.Clientset, controlPlane *MockClusterClient) *Reconciler {
	if controlPlane == nil {
		controlPlane = &MockClusterClient{}
	}
	return &Reconciler{
		clientSet: client,
		cluster:   controlPlane,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}
}

// -----------------------------------------------------------------------------
// Reactor Utilities - Add behaviors to capture or intercept K8s operations
// -----------------------------------------------------------------------------

// ReplicaSetCapture holds a captured ReplicaSet from a patch operation.
type ReplicaSetCapture struct {
	Applied *appsv1.ReplicaSet
}

// AddReplicaSetPatchReactor adds a reactor that captures server-side apply patches
// for ReplicaSets. Returns a capture struct to access the applied resource.
func AddReplicaSetPatchReactor(client *fake.Clientset) *ReplicaSetCapture {
	capture := &ReplicaSetCapture{}
	client.PrependReactor("patch", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var rs appsv1.ReplicaSet
		if err := json.Unmarshal(patchAction.GetPatch(), &rs); err != nil {
			return true, nil, err
		}

		capture.Applied = &rs
		rs.Namespace = patchAction.GetNamespace()
		return true, &rs, nil
	})
	return capture
}

// DeploymentCapture holds a captured Deployment from a patch operation.
type DeploymentCapture struct {
	Applied *appsv1.Deployment
}

// AddDeploymentPatchReactor adds a reactor that captures server-side apply patches
// for Deployments. Returns a capture struct to access the applied resource.
func AddDeploymentPatchReactor(client *fake.Clientset) *DeploymentCapture {
	capture := &DeploymentCapture{}
	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}

		capture.Applied = &dep
		dep.Namespace = patchAction.GetNamespace()
		dep.UID = "test-uid-12345"
		return true, &dep, nil
	})
	return capture
}

// ServiceCapture holds a captured Service from a patch operation.
type ServiceCapture struct {
	Applied *corev1.Service
}

// AddServicePatchReactor adds a reactor that captures server-side apply patches
// for Services. Returns a capture struct to access the applied resource.
func AddServicePatchReactor(client *fake.Clientset) *ServiceCapture {
	capture := &ServiceCapture{}
	client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var svc corev1.Service
		if err := json.Unmarshal(patchAction.GetPatch(), &svc); err != nil {
			return true, nil, err
		}

		capture.Applied = &svc
		svc.Namespace = patchAction.GetNamespace()
		return true, &svc, nil
	})
	return capture
}

// DeleteCapture tracks delete operations in order.
type DeleteCapture struct {
	Actions []string
}

// AddDeleteTracker adds a reactor that tracks all delete operations.
// Returns a capture struct with the ordered list of deleted resource types.
func AddDeleteTracker(client *fake.Clientset) *DeleteCapture {
	capture := &DeleteCapture{Actions: []string{}}
	client.PrependReactor("delete", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(k8stesting.DeleteAction)
		capture.Actions = append(capture.Actions, deleteAction.GetResource().Resource)
		return false, nil, nil
	})
	return capture
}

// NamespaceCreateCapture tracks namespace creation.
type NamespaceCreateCapture struct {
	Created bool
}

// AddNamespaceCreateTracker adds a reactor that tracks namespace creation.
func AddNamespaceCreateTracker(client *fake.Clientset) *NamespaceCreateCapture {
	capture := &NamespaceCreateCapture{}
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		capture.Created = true
		createAction := action.(k8stesting.CreateAction)
		return false, createAction.GetObject(), nil
	})
	return capture
}

// AddErrorReactor adds a reactor that returns an error for the specified verb and resource.
func AddErrorReactor(client *fake.Clientset, verb, resource string, err error) {
	client.PrependReactor(verb, resource, func(action k8stesting.Action) (handled bool, ret runtime.Object, retErr error) {
		return true, nil, err
	})
}

// AddPatchErrorReactor adds a reactor that returns an error only for SSA patch operations
// on the specified resource.
func AddPatchErrorReactor(client *fake.Clientset, resource string, err error) {
	client.PrependReactor("patch", resource, func(action k8stesting.Action) (handled bool, ret runtime.Object, retErr error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}
		return true, nil, err
	})
}

// -----------------------------------------------------------------------------
// TestHarness - Convenience struct for tests that need everything
// -----------------------------------------------------------------------------

// TestHarness provides a pre-configured Reconciler with fake Kubernetes and control
// plane clients for unit testing. Use the composable utilities above for more
// fine-grained control over test setup.
type TestHarness struct {
	Reconciler   *Reconciler
	Client       *fake.Clientset
	ControlPlane *MockClusterClient

	ReplicaSets *ReplicaSetCapture
	Deployments *DeploymentCapture
	Services    *ServiceCapture
	Deletes     *DeleteCapture
}

// NewTestHarness creates a TestHarness with all capture reactors pre-configured.
// For simpler tests or more control, use the composable utilities directly.
func NewTestHarness(t *testing.T, objects ...runtime.Object) *TestHarness {
	t.Helper()

	client := NewFakeClient(t, objects...)
	controlPlane := &MockClusterClient{}

	h := &TestHarness{
		Client:       client,
		ControlPlane: controlPlane,
		ReplicaSets:  AddReplicaSetPatchReactor(client),
		Deployments:  AddDeploymentPatchReactor(client),
		Services:     AddServicePatchReactor(client),
		Deletes:      AddDeleteTracker(client),
	}

	h.Reconciler = NewTestReconciler(client, controlPlane)

	return h
}
