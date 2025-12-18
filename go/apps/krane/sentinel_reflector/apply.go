package sentinelreflector

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// applySentinel creates or updates a Sentinel CRD based on the provided request.
//
// This method validates the request, ensures the namespace exists, and creates
// or updates the Sentinel custom resource. The controller-runtime reconciler
// will handle the actual Kubernetes resource creation based on this CRD.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: Sentinel application request with specifications
//
// Returns an error if validation fails, namespace creation fails,
// or CRD creation/update encounters problems.
func (r *Reflector) applySentinel(ctx context.Context, req *ctrlv1.ApplySentinel) error {

	r.logger.Info("applying sentinel",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
		"sentinel_id", req.GetSentinelId(),
	)
	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetSentinelId(), "Sentinel ID is required"),
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

	deployment, err := r.ensureDeploymentExists(ctx, req)
	if err != nil {
		return err
	}

	_, err = r.ensureServiceExists(ctx, req, deployment)
	if err != nil {
		return err
	}

	err = r.updateState(ctx, types.NamespacedName{Namespace: req.GetK8SNamespace(), Name: req.GetK8SName()})
	if err != nil {
		r.logger.Error("failed to reconcile replicaset", "sentinel_id", req.GetSentinelId(), "error", err)
		return err
	}

	return nil
}

func (r *Reflector) ensureDeploymentExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel) (*appsv1.Deployment, error) {

	client := r.clientSet.AppsV1().Deployments(sentinel.GetK8SNamespace())

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sentinel.GetK8SName(),
			Namespace: sentinel.GetK8SNamespace(),
			Labels: k8s.NewLabels().
				WorkspaceID(sentinel.GetWorkspaceId()).
				ProjectID(sentinel.GetProjectId()).
				EnvironmentID(sentinel.GetEnvironmentId()).
				SentinelID(sentinel.GetSentinelId()).
				ComponentSentinel().
				ManagedByKrane().
				ToMap(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.P(sentinel.GetReplicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: k8s.NewLabels().SentinelID(sentinel.GetSentinelId()).ToMap(),
			},

			MinReadySeconds: 30,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: k8s.NewLabels().SentinelID(sentinel.GetSentinelId()).ToMap(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers: []corev1.Container{{
						Image:           sentinel.GetImage(),
						Name:            "sentinel",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Args:            []string{"run", "sentinel"},
						Env: []corev1.EnvVar{
							{Name: "UNKEY_WORKSPACE_ID", Value: sentinel.GetWorkspaceId()},
							{Name: "UNKEY_PROJECT_ID", Value: sentinel.GetProjectId()},
							{Name: "UNKEY_ENVIRONMENT_ID", Value: sentinel.GetEnvironmentId()},
							{Name: "UNKEY_SENTINEL_ID", Value: sentinel.GetSentinelId()},
							{Name: "UNKEY_DATABASE_PRIMARY", Value: "unkey:password@tcp(mysql:3306)/unkey?parseTime=true&interpolateParams=true"},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: 8040,
							Name:          "sentinel",
						}},

						//Resources: corev1.ResourceRequirements{
						//	// nolint:exhaustive
						//	Limits: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(sentinel.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(sentinel.GetMemoryMib()), resource.BinarySI),
						//	},
						//	// nolint:exhaustive
						//	Requests: corev1.ResourceList{
						//		corev1.ResourceCPU:    *resource.NewMilliQuantity(int64(sentinel.GetCpuMillicores()), resource.BinarySI),
						//		corev1.ResourceMemory: *resource.NewQuantity(int64(sentinel.GetMemoryMib()), resource.BinarySI),
						//	},
						//},
					}},
				},
			},
		},
	}

	// Check if the deployment already exists, if not create a new one
	found, err := client.Get(ctx, sentinel.GetK8SName(), metav1.GetOptions{})

	if err == nil {

		if found.Spec.String() != deployment.Spec.String() {
			found.Spec = deployment.Spec
			return client.Update(ctx, found, metav1.UpdateOptions{})
		}

		// Nothing to do, we're in sync
		return found, nil

	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	return client.Create(ctx, deployment, metav1.CreateOptions{})

}

func (r *Reflector) ensureServiceExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel, deployment *appsv1.Deployment) (*corev1.Service, error) {
	client := r.clientSet.CoreV1().Services(sentinel.GetK8SNamespace())

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sentinel.GetK8SName(),
			Namespace: sentinel.GetK8SNamespace(),
			Labels: k8s.NewLabels().
				WorkspaceID(sentinel.GetWorkspaceId()).
				ProjectID(sentinel.GetProjectId()).
				EnvironmentID(sentinel.GetEnvironmentId()).
				SentinelID(sentinel.GetSentinelId()).
				ManagedByKrane().
				ComponentSentinel().
				ToMap(),
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Deployment",
					Name: deployment.GetName(),
					UID:  deployment.UID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
			Selector: k8s.NewLabels().SentinelID(sentinel.GetSentinelId()).ToMap(),
			//nolint:exhaustruct
			Ports: []corev1.ServicePort{
				{
					Port:       8040,
					TargetPort: intstr.FromInt(8040),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	found, err := client.Get(ctx, sentinel.GetK8SName(), metav1.GetOptions{})
	if err == nil {

		if found.Spec.String() != desired.Spec.String() {
			found.Spec = desired.Spec
			return client.Update(ctx, found, metav1.UpdateOptions{})
		}

		return found, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	return client.Create(ctx, desired, metav1.CreateOptions{})

}
