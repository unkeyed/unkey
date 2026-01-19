package reconciler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ApplyDeployment creates or updates a user workload as a Kubernetes ReplicaSet.
//
// The deployment represents a specific build of user code. ApplyDeployment uses
// server-side apply to create or update the ReplicaSet, which allows concurrent
// modifications from different sources without conflicts. After applying, it
// queries the resulting pods and reports their addresses and status back to the
// control plane so the routing layer knows where to send traffic.
//
// The namespace is created automatically if it doesn't exist. Pods run with gVisor
// isolation (RuntimeClass "gvisor") for security since they execute untrusted user code.
func (r *Reconciler) ApplyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) error {

	r.logger.Info("applying deployment",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
		"deployment_id", req.GetDeploymentId(),
	)
	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetDeploymentId(), "Deployment ID is required"),
		assert.NotEmpty(req.GetK8SNamespace(), "Namespace is required"),
		assert.NotEmpty(req.GetK8SName(), "K8s CRD name is required"),
		assert.NotEmpty(req.GetImage(), "Image is required"),
		assert.GreaterOrEqual(req.GetReplicas(), int32(0), "Replicas must be greater than or equal to 0"),
		assert.Greater(req.GetCpuMillicores(), int64(0), "CPU millicores must be greater than 0"),
		assert.Greater(req.GetMemoryMib(), int64(0), "MemoryMib must be greater than 0"),
	)
	if err != nil {
		return err
	}

	if err := r.ensureNamespaceExists(ctx, req.GetK8SNamespace(), req.GetWorkspaceId(), req.GetEnvironmentId()); err != nil {
		return err
	}

	// Define labels for resource selection
	usedLabels := labels.New().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		EnvironmentID(req.GetEnvironmentId()).
		DeploymentID(req.GetDeploymentId()).
		BuildID(req.GetBuildId()).
		ManagedByKrane().
		ComponentDeployment().
		Inject()

	desired := &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.GetK8SName(),
			Namespace: req.GetK8SNamespace(),
			Labels:    usedLabels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: ptr.P(req.GetReplicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.New().DeploymentID(req.GetDeploymentId()),
			},
			MinReadySeconds: 30,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// We need to prefix the name with our unkey-deployment name, to ensure uniqueness
					// This becomes important when sending status updates back to our database
					GenerateName: fmt.Sprintf("%s-", req.GetK8SName()),
					Labels:       usedLabels,
				},
				Spec: corev1.PodSpec{
					RuntimeClassName:          ptr.P(runtimeClassGvisor),
					RestartPolicy:             corev1.RestartPolicyAlways,
					Tolerations:               []corev1.Toleration{untrustedToleration},
					TopologySpreadConstraints: deploymentTopologySpread(req.GetDeploymentId()),
					Affinity:                  deploymentAffinity(req.GetEnvironmentId()),
					Containers: []corev1.Container{{
						Image:           req.GetImage(),
						Name:            "deployment",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         req.GetCommand(),
						Env: []corev1.EnvVar{
							{Name: "PORT", Value: strconv.Itoa(DeploymentPort)},
							{Name: "UNKEY_WORKSPACE_ID", Value: req.GetWorkspaceId()},
							{Name: "UNKEY_PROJECT_ID", Value: req.GetProjectId()},
							{Name: "UNKEY_ENVIRONMENT_ID", Value: req.GetEnvironmentId()},
							{Name: "UNKEY_DEPLOYMENT_ID", Value: req.GetDeploymentId()},
							{Name: "UNKEY_ENCRYPTED_ENV", Value: string(req.GetEncryptedEnvironmentVariables())},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: DeploymentPort,
							Name:          "deployment",
						}},

						Resources: corev1.ResourceRequirements{
							// nolint:exhaustive
							//Limits: corev1.ResourceList{
							//	corev1.ResourceCPU:              *resource.NewMilliQuantity(req.GetCpuMillicores(), resource.BinarySI),
							//	corev1.ResourceMemory:           *resource.NewQuantity(req.GetMemoryMib(), resource.BinarySI),
							//	corev1.ResourceEphemeralStorage: *resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
							//},
							//// nolint:exhaustive
							//Requests: corev1.ResourceList{
							//	corev1.ResourceCPU:              *resource.NewMilliQuantity(req.GetCpuMillicores(), resource.BinarySI),
							//	corev1.ResourceMemory:           *resource.NewQuantity(req.GetMemoryMib(), resource.BinarySI),
							//	corev1.ResourceEphemeralStorage: *resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
							//},
						},
					}},
				},
			},
		},
	}

	client := r.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace())

	patch, err := json.Marshal(desired)
	if err != nil {
		return fmt.Errorf("failed to marshal replicaset: %w", err)
	}

	applied, err := client.Patch(ctx, req.GetK8SName(), types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
	if err != nil {
		return fmt.Errorf("failed to apply replicaset: %w", err)
	}

	state, err := r.getDeploymentState(ctx, applied)
	if err != nil {
		return err
	}

	err = r.updateDeploymentState(ctx, state)
	if err != nil {
		r.logger.Error("failed to reconcile replicaset", "deployment_id", req.GetDeploymentId(), "error", err)
		return err
	}

	return nil
}
