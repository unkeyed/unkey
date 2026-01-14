package reconciler

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	corev1 "k8s.io/api/core/v1"
)

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
	client := NewFakeClient(t)
	capture := AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, capture.Applied, "should have captured applied ReplicaSet")

	var patchCount int
	for _, action := range client.Actions() {
		if action.GetVerb() == "patch" && action.GetResource().Resource == "replicasets" {
			patchCount++
		}
	}
	require.Equal(t, 1, patchCount, "expected exactly one patch action")
}

func TestApplyDeployment_SetsCorrectImage(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	capture := AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	req.Image = "nginx:1.25"

	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, capture.Applied, "should have captured applied ReplicaSet")
	require.Len(t, capture.Applied.Spec.Template.Spec.Containers, 1)
	require.Equal(t, "nginx:1.25", capture.Applied.Spec.Template.Spec.Containers[0].Image)
}

func TestApplyDeployment_SetsCorrectReplicas(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	capture := AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	req.Replicas = 5

	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, capture.Applied)
	require.Equal(t, int32(5), *capture.Applied.Spec.Replicas)
}

func TestApplyDeployment_SetsCorrectLabels(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	capture := AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, capture.Applied)

	labels := capture.Applied.Spec.Template.Labels
	require.Equal(t, "ws_123", labels["unkey.com/workspace.id"])
	require.Equal(t, "prj_123", labels["unkey.com/project.id"])
	require.Equal(t, "env_123", labels["unkey.com/environment.id"])
	require.Equal(t, "dep_123", labels["unkey.com/deployment.id"])
	require.Equal(t, "krane", labels["app.kubernetes.io/managed-by"])
}

func TestApplyDeployment_SetsEnvironmentVariables(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	capture := AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, capture.Applied)
	require.Len(t, capture.Applied.Spec.Template.Spec.Containers, 1)

	envVars := capture.Applied.Spec.Template.Spec.Containers[0].Env
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
	client := NewFakeClient(t)
	capture := AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, capture.Applied)
	require.Equal(t, "apps/v1", capture.Applied.APIVersion)
	require.Equal(t, "ReplicaSet", capture.Applied.Kind)
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
			client := NewFakeClient(t)
			AddReplicaSetPatchReactor(client)
			r := NewTestReconciler(client, nil)

			req := newApplyDeploymentRequest()
			tt.mutate(req)

			err := r.ApplyDeployment(ctx, req)
			require.Error(t, err)
		})
	}
}

func TestApplyDeployment_CallsUpdateDeploymentState(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddReplicaSetPatchReactor(client)
	mockCluster := &MockClusterClient{}
	r := NewTestReconciler(client, mockCluster)

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
	client := NewFakeClient(t)
	AddReplicaSetPatchReactor(client)

	mockCluster := &MockClusterClient{
		UpdateDeploymentStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
			return nil, fmt.Errorf("control plane unavailable")
		},
	}
	r := NewTestReconciler(client, mockCluster)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "control plane unavailable")
}

func TestApplyDeployment_EnsuresNamespaceExists(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClientWithoutNamespace(t)
	nsCapture := AddNamespaceCreateTracker(client)
	AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)
	require.True(t, nsCapture.Created, "namespace should be created if missing")
}

func TestApplyDeployment_SetsTolerations(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	capture := AddReplicaSetPatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, capture.Applied)
	require.Len(t, capture.Applied.Spec.Template.Spec.Tolerations, 1)

	toleration := capture.Applied.Spec.Template.Spec.Tolerations[0]
	require.Equal(t, "node-class", toleration.Key)
	require.Equal(t, corev1.TolerationOpEqual, toleration.Operator)
	require.Equal(t, "customer-code", toleration.Value)
	require.Equal(t, corev1.TaintEffectNoSchedule, toleration.Effect)
}

func TestApplyDeployment_ReplicaSetPatchError(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddPatchErrorReactor(client, "replicasets", fmt.Errorf("simulated replicaset patch failure"))
	r := NewTestReconciler(client, nil)

	req := newApplyDeploymentRequest()
	err := r.ApplyDeployment(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated replicaset patch failure")
}
