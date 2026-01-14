package reconciler

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	corev1 "k8s.io/api/core/v1"
)

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
	client := NewFakeClient(t)
	depCapture := AddDeploymentPatchReactor(client)
	svcCapture := AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, depCapture.Applied, "Deployment should be applied")
	require.NotNil(t, svcCapture.Applied, "Service should be applied")
}

func TestApplySentinel_DeploymentHasCorrectImage(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	depCapture := AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	req.Image = "unkey/sentinel:v2.0"

	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, depCapture.Applied)
	require.Len(t, depCapture.Applied.Spec.Template.Spec.Containers, 1)
	require.Equal(t, "unkey/sentinel:v2.0", depCapture.Applied.Spec.Template.Spec.Containers[0].Image)
}

func TestApplySentinel_DeploymentHasCorrectReplicas(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	depCapture := AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	req.Replicas = 5

	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, depCapture.Applied)
	require.Equal(t, int32(5), *depCapture.Applied.Spec.Replicas)
}

func TestApplySentinel_DeploymentHasCorrectLabels(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	depCapture := AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, depCapture.Applied)

	labels := depCapture.Applied.Labels
	require.Equal(t, "ws_123", labels["unkey.com/workspace.id"])
	require.Equal(t, "prj_123", labels["unkey.com/project.id"])
	require.Equal(t, "env_123", labels["unkey.com/environment.id"])
	require.Equal(t, "sent_123", labels["unkey.com/sentinel.id"])
	require.Equal(t, "krane", labels["app.kubernetes.io/managed-by"])
	require.Equal(t, "sentinel", labels["app.kubernetes.io/component"])
}

func TestApplySentinel_DeploymentHasEnvironmentVariables(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	depCapture := AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, depCapture.Applied)
	require.Len(t, depCapture.Applied.Spec.Template.Spec.Containers, 1)

	envVars := depCapture.Applied.Spec.Template.Spec.Containers[0].Env
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
	client := NewFakeClient(t)
	AddDeploymentPatchReactor(client)
	svcCapture := AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, svcCapture.Applied)
	require.Len(t, svcCapture.Applied.Spec.Ports, 1)
	require.Equal(t, int32(8040), svcCapture.Applied.Spec.Ports[0].Port)
	require.Equal(t, corev1.ProtocolTCP, svcCapture.Applied.Spec.Ports[0].Protocol)
}

func TestApplySentinel_ServiceHasCorrectSelector(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddDeploymentPatchReactor(client)
	svcCapture := AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, svcCapture.Applied)
	require.Equal(t, "sent_123", svcCapture.Applied.Spec.Selector["unkey.com/sentinel.id"])
}

func TestApplySentinel_ServiceHasOwnerReference(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddDeploymentPatchReactor(client)
	svcCapture := AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, svcCapture.Applied)
	require.Len(t, svcCapture.Applied.OwnerReferences, 1)
	require.Equal(t, "Deployment", svcCapture.Applied.OwnerReferences[0].Kind)
	require.Equal(t, "test-sentinel", svcCapture.Applied.OwnerReferences[0].Name)
}

func TestApplySentinel_DeploymentSetsTypeMeta(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	depCapture := AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, depCapture.Applied)
	require.Equal(t, "apps/v1", depCapture.Applied.APIVersion)
	require.Equal(t, "Deployment", depCapture.Applied.Kind)
}

func TestApplySentinel_ServiceSetsTypeMeta(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddDeploymentPatchReactor(client)
	svcCapture := AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, svcCapture.Applied)
	require.Equal(t, "v1", svcCapture.Applied.APIVersion)
	require.Equal(t, "Service", svcCapture.Applied.Kind)
}

// TestApplySentinel_EnsuresNamespaceExists verifies that ApplySentinel creates
// the target namespace if it doesn't already exist before applying the deployment
// and service resources.
func TestApplySentinel_EnsuresNamespaceExists(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClientWithoutNamespace(t)
	nsCapture := AddNamespaceCreateTracker(client)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)
	require.True(t, nsCapture.Created, "namespace should be created if missing")
}

// TestApplySentinel_DeploymentPatchError verifies that errors from the Kubernetes
// API during server-side apply of the Deployment resource are properly propagated
// back to the caller.
func TestApplySentinel_DeploymentPatchError(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddPatchErrorReactor(client, "deployments", fmt.Errorf("simulated deployment patch failure"))
	r := NewTestReconciler(client, nil)

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
	client := NewFakeClient(t)
	AddDeploymentPatchReactor(client)
	AddPatchErrorReactor(client, "services", fmt.Errorf("simulated service patch failure"))
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "simulated service patch failure")
}

func TestApplySentinel_CallsUpdateSentinelState(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	mockCluster := &MockClusterClient{}
	r := NewTestReconciler(client, mockCluster)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.Len(t, mockCluster.UpdateSentinelStateCalls, 1)
	call := mockCluster.UpdateSentinelStateCalls[0]
	require.Equal(t, req.GetK8SName(), call.GetK8SName())
}

func TestApplySentinel_ControlPlaneError(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)

	mockCluster := &MockClusterClient{
		UpdateSentinelStateFunc: func(ctx context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
			return nil, fmt.Errorf("control plane unavailable")
		},
	}
	r := NewTestReconciler(client, mockCluster)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "control plane unavailable")
}

func TestApplySentinel_SetsTolerations(t *testing.T) {
	ctx := context.Background()
	client := NewFakeClient(t)
	depCapture := AddDeploymentPatchReactor(client)
	AddServicePatchReactor(client)
	r := NewTestReconciler(client, nil)

	req := newApplySentinelRequest()
	err := r.ApplySentinel(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, depCapture.Applied)
	require.Len(t, depCapture.Applied.Spec.Template.Spec.Tolerations, 1)

	toleration := depCapture.Applied.Spec.Template.Spec.Tolerations[0]
	require.Equal(t, "node-class", toleration.Key)
	require.Equal(t, corev1.TolerationOpEqual, toleration.Operator)
	require.Equal(t, "customer-code", toleration.Value)
	require.Equal(t, corev1.TaintEffectNoSchedule, toleration.Effect)
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
			client := NewFakeClient(t)
			AddDeploymentPatchReactor(client)
			AddServicePatchReactor(client)
			r := NewTestReconciler(client, nil)

			req := newApplySentinelRequest()
			tt.mutate(req)

			err := r.ApplySentinel(ctx, req)
			require.Error(t, err)
		})
	}
}
