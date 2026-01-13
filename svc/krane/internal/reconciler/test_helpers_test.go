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

// TestHarness provides a pre-configured [Reconciler] with fake Kubernetes and control
// plane clients for unit testing. The harness captures resources applied during tests
// so assertions can verify the correct Kubernetes objects were created. It also tracks
// delete operations in order, which is useful for verifying cleanup sequences.
type TestHarness struct {
	Reconciler        *Reconciler
	Client            *fake.Clientset
	ControlPlane      *MockClusterClient
	AppliedReplicaSet *appsv1.ReplicaSet
	AppliedDeployment *appsv1.Deployment
	AppliedService    *corev1.Service
	DeleteActions     []string
}

// NewTestHarness creates a [TestHarness] with a fake Kubernetes client pre-seeded
// with a "test-namespace" namespace and any additional objects passed as arguments.
// The harness sets up reactors to capture apply operations, so tests can inspect
// AppliedReplicaSet, AppliedDeployment, and AppliedService after calling reconciler methods.
func NewTestHarness(t *testing.T, objects ...runtime.Object) *TestHarness {
	t.Helper()

	// Always include the test namespace
	allObjects := append([]runtime.Object{newTestNamespace()}, objects...)
	client := fake.NewSimpleClientset(allObjects...)
	controlPlane := &MockClusterClient{}

	h := &TestHarness{
		Client:        client,
		ControlPlane:  controlPlane,
		DeleteActions: []string{},
	}

	// Set up reactors to capture applied resources
	h.addReplicaSetReactor()
	h.addDeploymentReactor()
	h.addServiceReactor()
	h.addDeleteTracker()

	h.Reconciler = &Reconciler{
		clientSet: client,
		cluster:   controlPlane,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	return h
}

func (h *TestHarness) addReplicaSetReactor() {
	h.Client.PrependReactor("patch", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var rs appsv1.ReplicaSet
		if err := json.Unmarshal(patchAction.GetPatch(), &rs); err != nil {
			return true, nil, err
		}

		h.AppliedReplicaSet = &rs
		rs.Namespace = patchAction.GetNamespace()
		return true, &rs, nil
	})
}

func (h *TestHarness) addDeploymentReactor() {
	h.Client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}

		h.AppliedDeployment = &dep
		dep.Namespace = patchAction.GetNamespace()
		dep.UID = "test-uid-12345"
		return true, &dep, nil
	})
}

func (h *TestHarness) addServiceReactor() {
	h.Client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var svc corev1.Service
		if err := json.Unmarshal(patchAction.GetPatch(), &svc); err != nil {
			return true, nil, err
		}

		h.AppliedService = &svc
		svc.Namespace = patchAction.GetNamespace()
		return true, &svc, nil
	})
}

func (h *TestHarness) addDeleteTracker() {
	h.Client.PrependReactor("delete", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		deleteAction := action.(k8stesting.DeleteAction)
		h.DeleteActions = append(h.DeleteActions, deleteAction.GetResource().Resource)
		return false, nil, nil
	})
}

// -----------------------------------------------------------------------------
// Test Data Builders
// -----------------------------------------------------------------------------

func newTestNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
}

// addReplicaSetPatchReactor is a legacy helper retained for tests not yet migrated
// to [TestHarness]. Prefer NewTestHarness for new tests.
func addReplicaSetPatchReactor(client *fake.Clientset, captureFunc func(*appsv1.ReplicaSet)) {
	client.PrependReactor("patch", "replicasets", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var rs appsv1.ReplicaSet
		if err := json.Unmarshal(patchAction.GetPatch(), &rs); err != nil {
			return true, nil, err
		}

		if captureFunc != nil {
			captureFunc(&rs)
		}
		rs.Namespace = patchAction.GetNamespace()
		return true, &rs, nil
	})
}

// newTestReconciler is a legacy helper retained for tests not yet migrated
// to [TestHarness]. Prefer NewTestHarness for new tests.
func newTestReconciler(client *fake.Clientset, controlPlane *MockClusterClient) *Reconciler {
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
