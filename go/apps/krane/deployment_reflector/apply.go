package deploymentreflector

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// applyDeployment creates or updates a Deployment CRD based on the provided request.
//
// This method validates the request, ensures the namespace exists, and creates
// or updates the Deployment custom resource. The controller-runtime reconciler
// will handle the actual Kubernetes resource creation based on this CRD.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: Deployment application request with specifications
//
// Returns an error if validation fails, namespace creation fails,
// or CRD creation/update encounters problems.
func (r *Reflector) applyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) error {

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
		assert.Greater(req.GetCpuMillicores(), int64(0), "CPU millicores must be greater than 0"),
		assert.Greater(req.GetMemoryMib(), int64(0), "MemoryMib must be greater than 0"),
	)
	if err != nil {
		return err
	}

	if err := r.ensureNamespaceExists(ctx, req.GetK8SNamespace()); err != nil {
		return err
	}

	// Define labels for resource selection
	usedLabels := k8s.NewLabels().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		EnvironmentID(req.GetEnvironmentId()).
		DeploymentID(req.GetDeploymentId()).
		ManagedByKrane().
		ComponentDeployment().
		ToMap()

	desired := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.GetK8SName(),
			Namespace: req.GetK8SNamespace(),
			Labels:    usedLabels,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: ptr.P(req.GetReplicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: k8s.NewLabels().DeploymentID(req.GetDeploymentId()).ToMap(),
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

					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{{
						Image:           req.GetImage(),
						Name:            "deployment",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         []string{},
						Env: []corev1.EnvVar{
							{Name: "UNKEY_WORKSPACE_ID", Value: req.GetWorkspaceId()},
							{Name: "UNKEY_PROJECT_ID", Value: req.GetProjectId()},
							{Name: "UNKEY_ENVIRONMENT_ID", Value: req.GetEnvironmentId()},
							{Name: "UNKEY_DEPLOYMENT_ID", Value: req.GetDeploymentId()},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: 8040,
							Name:          "deployment",
						}},

						//Resources: corev1.ResourceRequirements{
						//	// nolint:exhaustive
						//	Limits: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(req.GetMemoryMib()), resource.BinarySI),
						//	},
						//	// nolint:exhaustive
						//	Requests: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(req.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(req.GetMemoryMib()), resource.BinarySI),
						//	},
						//},
					}},
				},
			},
		},
	}

	client := r.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace())

	existing, err := client.Get(ctx, req.GetK8SName(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			_, err = client.Create(ctx, desired, metav1.CreateOptions{})
			return err
		}
		return err
	}

	if existing.Spec.String() != desired.Spec.String() {
		existing.Spec = desired.Spec
		_, err := client.Update(ctx, existing, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	err = r.updateState(ctx, types.NamespacedName{Namespace: existing.Namespace, Name: existing.Name})
	if err != nil {
		r.logger.Error("failed to reconcile replicaset", "deployment_id", req.GetDeploymentId(), "error", err)
		return err
	}

	return nil
}
