package reconciler

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// testHarness provides a configured Reconciler and fake client for testing.
type testHarness struct {
	reconciler *Reconciler
	client     *fake.Clientset
	applied    *appsv1.ReplicaSet
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	namespace := newTestNamespace()
	client := fake.NewSimpleClientset(namespace)

	h := &testHarness{
		client: client,
	}

	addReplicaSetPatchReactor(client, func(rs *appsv1.ReplicaSet) {
		h.applied = rs
	})

	h.reconciler = newTestReconciler(client, nil)

	return h
}

func newApplyDeploymentRequest() *ctrlv1.ApplyDeployment {
	return &ctrlv1.ApplyDeployment{
		WorkspaceId:   "ws_123",
		ProjectId:     "prj_123",
		EnvironmentId: "env_123",
		DeploymentId:  "dep_123",
		K8SNamespace:  "test-namespace",
		K8SName:       "test-deployment",
		Image:         "nginx:1.19",
		Replicas:      3,
		CpuMillicores: 100,
		MemoryMib:     128,
		BuildId:       ptr.P("build_123"),
	}
}

func TestApplyDeployment_UsesServerSideApply(t *testing.T) {
	ctx := context.Background()
	h := newTestHarness(t)

	req := newApplyDeploymentRequest()
	err := h.reconciler.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	// Verify a patch action was issued.
	var patchCount int
	for _, action := range h.client.Actions() {
		if action.GetVerb() == "patch" && action.GetResource().Resource == "replicasets" {
			patchCount++
		}
	}
	require.Equal(t, 1, patchCount, "expected exactly one patch action")
}

func TestApplyDeployment_SetsCorrectImage(t *testing.T) {
	ctx := context.Background()
	h := newTestHarness(t)

	req := newApplyDeploymentRequest()
	req.Image = "nginx:1.25"

	err := h.reconciler.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.applied, "should have captured applied ReplicaSet")
	require.Len(t, h.applied.Spec.Template.Spec.Containers, 1)
	require.Equal(t, "nginx:1.25", h.applied.Spec.Template.Spec.Containers[0].Image)
}

func TestApplyDeployment_SetsCorrectReplicas(t *testing.T) {
	ctx := context.Background()
	h := newTestHarness(t)

	req := newApplyDeploymentRequest()
	req.Replicas = 5

	err := h.reconciler.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.applied)
	require.Equal(t, int32(5), *h.applied.Spec.Replicas)
}

func TestApplyDeployment_SetsCorrectLabels(t *testing.T) {
	ctx := context.Background()
	h := newTestHarness(t)

	req := newApplyDeploymentRequest()
	err := h.reconciler.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.applied)

	labels := h.applied.Spec.Template.Labels
	require.Equal(t, "ws_123", labels["unkey.com/workspace.id"])
	require.Equal(t, "prj_123", labels["unkey.com/project.id"])
	require.Equal(t, "env_123", labels["unkey.com/environment.id"])
	require.Equal(t, "dep_123", labels["unkey.com/deployment.id"])
	require.Equal(t, "krane", labels["app.kubernetes.io/managed-by"])
}

func TestApplyDeployment_SetsEnvironmentVariables(t *testing.T) {
	ctx := context.Background()
	h := newTestHarness(t)

	req := newApplyDeploymentRequest()
	err := h.reconciler.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.applied)
	require.Len(t, h.applied.Spec.Template.Spec.Containers, 1)

	envVars := h.applied.Spec.Template.Spec.Containers[0].Env
	envMap := make(map[string]string)
	for _, env := range envVars {
		envMap[env.Name] = env.Value
	}

	require.Equal(t, "ws_123", envMap["UNKEY_WORKSPACE_ID"])
	require.Equal(t, "prj_123", envMap["UNKEY_PROJECT_ID"])
	require.Equal(t, "env_123", envMap["UNKEY_ENVIRONMENT_ID"])
	require.Equal(t, "dep_123", envMap["UNKEY_DEPLOYMENT_ID"])
}

func TestApplyDeployment_SetsTypeMeta(t *testing.T) {
	ctx := context.Background()
	h := newTestHarness(t)

	req := newApplyDeploymentRequest()
	err := h.reconciler.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, h.applied)
	require.Equal(t, "apps/v1", h.applied.APIVersion)
	require.Equal(t, "ReplicaSet", h.applied.Kind)
}

func TestApplyDeployment_ValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ctrlv1.ApplyDeployment)
	}{
		{
			name:   "missing workspace id",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.WorkspaceId = "" },
		},
		{
			name:   "missing project id",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.ProjectId = "" },
		},
		{
			name:   "missing environment id",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.EnvironmentId = "" },
		},
		{
			name:   "missing deployment id",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.DeploymentId = "" },
		},
		{
			name:   "missing namespace",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.K8SNamespace = "" },
		},
		{
			name:   "missing k8s name",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.K8SName = "" },
		},
		{
			name:   "missing image",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.Image = "" },
		},
		{
			name:   "zero cpu millicores",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.CpuMillicores = 0 },
		},
		{
			name:   "zero memory",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.MemoryMib = 0 },
		},
		{
			name:   "negative replicas",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.Replicas = -1 },
		},
		{
			name:   "negative cpu millicores",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.CpuMillicores = -100 },
		},
		{
			name:   "negative memory",
			mutate: func(req *ctrlv1.ApplyDeployment) { req.MemoryMib = -128 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			h := newTestHarness(t)

			req := newApplyDeploymentRequest()
			tt.mutate(req)

			err := h.reconciler.ApplyDeployment(ctx, req)
			require.Error(t, err)
		})
	}
}

func TestApplyDeployment_CallsUpdateDeploymentState(t *testing.T) {
	ctx := context.Background()

	namespace := newTestNamespace()
	client := fake.NewSimpleClientset(namespace)

	addReplicaSetPatchReactor(client, nil)

	mockCluster := &MockClusterClient{}
	r := newTestReconciler(client, mockCluster)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.Len(t, mockCluster.UpdateDeploymentStateCalls, 1)
	call := mockCluster.UpdateDeploymentStateCalls[0]
	update := call.GetUpdate()
	require.NotNil(t, update)
	require.Equal(t, req.GetK8SName(), update.GetK8SName())
}

func TestApplyDeployment_ControlPlaneError(t *testing.T) {
	ctx := context.Background()

	namespace := newTestNamespace()
	client := fake.NewSimpleClientset(namespace)

	addReplicaSetPatchReactor(client, nil)

	mockCluster := &MockClusterClient{
		UpdateDeploymentStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
			return nil, fmt.Errorf("control plane unavailable")
		},
	}
	r := newTestReconciler(client, mockCluster)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "control plane unavailable")
}

func TestApplyDeployment_EnsuresNamespaceExists(t *testing.T) {
	ctx := context.Background()

	// Create client without namespace to verify it gets created.
	client := fake.NewSimpleClientset()

	var namespaceCreated bool
	client.PrependReactor("create", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		namespaceCreated = true
		createAction := action.(k8stesting.CreateAction)
		return false, createAction.GetObject(), nil
	})

	addReplicaSetPatchReactor(client, nil)

	r := newTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)
	require.True(t, namespaceCreated, "namespace should be created if missing")
}
