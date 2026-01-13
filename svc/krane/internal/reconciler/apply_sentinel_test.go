package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
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

// sentinelTestHarness provides a configured Reconciler and fake client for sentinel tests.
type sentinelTestHarness struct {
	reconciler        *Reconciler
	client            *fake.Clientset
	appliedDeployment *appsv1.Deployment
	appliedService    *corev1.Service
}

func newSentinelTestHarness(t *testing.T) *sentinelTestHarness {
	t.Helper()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)

	h := &sentinelTestHarness{
		client: client,
	}

	// Add reactor to capture server-side apply patches for Deployments.
	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}

		h.appliedDeployment = &dep
		dep.Namespace = patchAction.GetNamespace()
		dep.UID = "test-uid-12345"
		return true, &dep, nil
	})

	// Add reactor to capture server-side apply patches for Services.
	client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var svc corev1.Service
		if err := json.Unmarshal(patchAction.GetPatch(), &svc); err != nil {
			return true, nil, err
		}

		h.appliedService = &svc
		svc.Namespace = patchAction.GetNamespace()
		return true, &svc, nil
	})

	h.reconciler = &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	return h
}

func newApplySentinelRequest() *ctrlv1.ApplySentinel {
	return &ctrlv1.ApplySentinel{
		WorkspaceId:   "ws_123",
		ProjectId:     "prj_123",
		EnvironmentId: "env_123",
		SentinelId:    "sent_123",
		K8SNamespace:  "test-namespace",
		K8SName:       "test-sentinel",
		Image:         "unkey/sentinel:v1.0",
		Replicas:      2,
		CpuMillicores: 200,
		MemoryMib:     256,
	}
}

func TestApplySentinel_CreatesDeploymentAndService(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedDeployment, "Deployment should be applied")
	require.NotNil(t, h.appliedService, "Service should be applied")
}

func TestApplySentinel_DeploymentHasCorrectImage(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	req.Image = "unkey/sentinel:v2.0"

	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedDeployment)
	require.Len(t, h.appliedDeployment.Spec.Template.Spec.Containers, 1)
	require.Equal(t, "unkey/sentinel:v2.0", h.appliedDeployment.Spec.Template.Spec.Containers[0].Image)
}

func TestApplySentinel_DeploymentHasCorrectReplicas(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	req.Replicas = 5

	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedDeployment)
	require.Equal(t, int32(5), *h.appliedDeployment.Spec.Replicas)
}

func TestApplySentinel_DeploymentHasCorrectLabels(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedDeployment)

	labels := h.appliedDeployment.Labels
	require.Equal(t, "ws_123", labels["unkey.com/workspace.id"])
	require.Equal(t, "prj_123", labels["unkey.com/project.id"])
	require.Equal(t, "env_123", labels["unkey.com/environment.id"])
	require.Equal(t, "sent_123", labels["unkey.com/sentinel.id"])
	require.Equal(t, "krane", labels["app.kubernetes.io/managed-by"])
	require.Equal(t, "sentinel", labels["app.kubernetes.io/component"])
}

func TestApplySentinel_DeploymentHasEnvironmentVariables(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedDeployment)
	require.Len(t, h.appliedDeployment.Spec.Template.Spec.Containers, 1)

	envVars := h.appliedDeployment.Spec.Template.Spec.Containers[0].Env
	envMap := make(map[string]string)
	for _, env := range envVars {
		envMap[env.Name] = env.Value
	}

	require.Equal(t, "ws_123", envMap["UNKEY_WORKSPACE_ID"])
	require.Equal(t, "prj_123", envMap["UNKEY_PROJECT_ID"])
	require.Equal(t, "env_123", envMap["UNKEY_ENVIRONMENT_ID"])
	require.Equal(t, "sent_123", envMap["UNKEY_SENTINEL_ID"])
	require.Equal(t, "test-region", envMap["UNKEY_REGION"])
}

func TestApplySentinel_ServiceHasCorrectPort(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedService)
	require.Len(t, h.appliedService.Spec.Ports, 1)
	require.Equal(t, int32(8040), h.appliedService.Spec.Ports[0].Port)
	require.Equal(t, corev1.ProtocolTCP, h.appliedService.Spec.Ports[0].Protocol)
}

func TestApplySentinel_ServiceHasCorrectSelector(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedService)
	require.Equal(t, "sent_123", h.appliedService.Spec.Selector["unkey.com/sentinel.id"])
}

func TestApplySentinel_ServiceHasOwnerReference(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedService)
	require.Len(t, h.appliedService.OwnerReferences, 1)
	require.Equal(t, "Deployment", h.appliedService.OwnerReferences[0].Kind)
	require.Equal(t, "test-sentinel", h.appliedService.OwnerReferences[0].Name)
}

func TestApplySentinel_DeploymentSetsTypeMeta(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedDeployment)
	require.Equal(t, "apps/v1", h.appliedDeployment.APIVersion)
	require.Equal(t, "Deployment", h.appliedDeployment.Kind)
}

func TestApplySentinel_ServiceSetsTypeMeta(t *testing.T) {
	ctx := context.Background()
	h := newSentinelTestHarness(t)

	req := newApplySentinelRequest()
	err := h.reconciler.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.appliedService)
	require.Equal(t, "v1", h.appliedService.APIVersion)
	require.Equal(t, "Service", h.appliedService.Kind)
}

// TestApplySentinel_EnsuresNamespaceExists verifies that ApplySentinel creates
// the target namespace if it doesn't already exist before applying the deployment
// and service resources.
func TestApplySentinel_EnsuresNamespaceExists(t *testing.T) {
	ctx := context.Background()

	// Create client without namespace to verify it gets created.
	client := fake.NewSimpleClientset()

	var namespaceCreated bool
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		namespaceCreated = true
		createAction := action.(k8stesting.CreateAction)
		return false, createAction.GetObject(), nil
	})

	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}
		dep.Namespace = patchAction.GetNamespace()
		dep.UID = "test-uid-12345"
		return true, &dep, nil
	})

	client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var svc corev1.Service
		if err := json.Unmarshal(patchAction.GetPatch(), &svc); err != nil {
			return true, nil, err
		}
		svc.Namespace = patchAction.GetNamespace()
		return true, &svc, nil
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)
	require.True(t, namespaceCreated, "namespace should be created if missing")
}

// TestApplySentinel_DeploymentPatchError verifies that errors from the Kubernetes
// API during server-side apply of the Deployment resource are properly propagated
// back to the caller.
func TestApplySentinel_DeploymentPatchError(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)

	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}
		return true, nil, fmt.Errorf("simulated deployment patch failure")
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated deployment patch failure")
}

// TestApplySentinel_ServicePatchError verifies that errors from the Kubernetes
// API during server-side apply of the Service resource are properly propagated
// back to the caller, even when the Deployment was applied successfully.
func TestApplySentinel_ServicePatchError(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)

	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}
		dep.Namespace = patchAction.GetNamespace()
		dep.UID = "test-uid-12345"
		return true, &dep, nil
	})

	client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}
		return true, nil, fmt.Errorf("simulated service patch failure")
	})

	r := &Reconciler{
		clientSet: client,
		cluster:   &MockClusterClient{},
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated service patch failure")
}

func TestApplySentinel_CallsUpdateSentinelState(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)

	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}
		dep.Namespace = patchAction.GetNamespace()
		dep.UID = "test-uid-12345"
		dep.Status.AvailableReplicas = 2
		return true, &dep, nil
	})

	client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var svc corev1.Service
		if err := json.Unmarshal(patchAction.GetPatch(), &svc); err != nil {
			return true, nil, err
		}
		svc.Namespace = patchAction.GetNamespace()
		return true, &svc, nil
	})

	mockCluster := &MockClusterClient{}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.Len(t, mockCluster.UpdateSentinelStateCalls, 1)
	call := mockCluster.UpdateSentinelStateCalls[0]
	require.Equal(t, req.GetK8SName(), call.GetK8SName())
	require.Equal(t, int32(2), call.GetAvailableReplicas())
}

func TestApplySentinel_ControlPlaneError(t *testing.T) {
	ctx := context.Background()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}
	client := fake.NewSimpleClientset(namespace)

	client.PrependReactor("patch", "deployments", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var dep appsv1.Deployment
		if err := json.Unmarshal(patchAction.GetPatch(), &dep); err != nil {
			return true, nil, err
		}
		dep.Namespace = patchAction.GetNamespace()
		dep.UID = "test-uid-12345"
		return true, &dep, nil
	})

	client.PrependReactor("patch", "services", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		patchAction := action.(k8stesting.PatchAction)
		if patchAction.GetPatchType() != types.ApplyPatchType {
			return false, nil, nil
		}

		var svc corev1.Service
		if err := json.Unmarshal(patchAction.GetPatch(), &svc); err != nil {
			return true, nil, err
		}
		svc.Namespace = patchAction.GetNamespace()
		return true, &svc, nil
	})

	mockCluster := &MockClusterClient{
		UpdateSentinelStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
			return nil, fmt.Errorf("control plane unavailable")
		},
	}
	r := &Reconciler{
		clientSet: client,
		cluster:   mockCluster,
		cb:        circuitbreaker.New[any]("test"),
		logger:    logging.NewNoop(),
		region:    "test-region",
	}

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "control plane unavailable")
}

func TestApplySentinel_ValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ctrlv1.ApplySentinel)
	}{
		{
			name:   "missing workspace id",
			mutate: func(req *ctrlv1.ApplySentinel) { req.WorkspaceId = "" },
		},
		{
			name:   "missing project id",
			mutate: func(req *ctrlv1.ApplySentinel) { req.ProjectId = "" },
		},
		{
			name:   "missing environment id",
			mutate: func(req *ctrlv1.ApplySentinel) { req.EnvironmentId = "" },
		},
		{
			name:   "missing sentinel id",
			mutate: func(req *ctrlv1.ApplySentinel) { req.SentinelId = "" },
		},
		{
			name:   "missing namespace",
			mutate: func(req *ctrlv1.ApplySentinel) { req.K8SNamespace = "" },
		},
		{
			name:   "missing k8s name",
			mutate: func(req *ctrlv1.ApplySentinel) { req.K8SName = "" },
		},
		{
			name:   "missing image",
			mutate: func(req *ctrlv1.ApplySentinel) { req.Image = "" },
		},
		{
			name:   "zero cpu millicores",
			mutate: func(req *ctrlv1.ApplySentinel) { req.CpuMillicores = 0 },
		},
		{
			name:   "zero memory",
			mutate: func(req *ctrlv1.ApplySentinel) { req.MemoryMib = 0 },
		},
		{
			name:   "negative replicas",
			mutate: func(req *ctrlv1.ApplySentinel) { req.Replicas = -1 },
		},
		{
			name:   "negative cpu millicores",
			mutate: func(req *ctrlv1.ApplySentinel) { req.CpuMillicores = -100 },
		},
		{
			name:   "negative memory",
			mutate: func(req *ctrlv1.ApplySentinel) { req.MemoryMib = -128 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := newSentinelTestHarness(t)

			req := newApplySentinelRequest()
			tt.mutate(req)

			err := h.reconciler.ApplySentinel(ctx, req)
			require.Error(t, err)
		})
	}
}
