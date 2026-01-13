package reconciler

import (
	"context"
	"crypto/sha256"
	"encoding/json"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ApplySentinel creates or updates a Sentinel CRD based on the provided request.
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
func (r *Reconciler) ApplySentinel(ctx context.Context, req *ctrlv1.ApplySentinel) error {

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

	sentinel, err := r.ensureSentinelExists(ctx, req)
	if err != nil {
		return err
	}

	_, err = r.ensureServiceExists(ctx, req, sentinel)
	if err != nil {
		return err
	}

	err = r.updateSentinelState(ctx, &ctrlv1.UpdateSentinelStateRequest{
		K8SName:           req.GetK8SName(),
		AvailableReplicas: sentinel.Status.AvailableReplicas,
	})
	if err != nil {
		r.logger.Error("failed to reconcile replicaset", "sentinel_id", req.GetSentinelId(), "error", err)
		return err
	}

	return nil
}

func (r *Reconciler) ensureSentinelExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel) (*appsv1.Deployment, error) {

	client := r.clientSet.AppsV1().Deployments(sentinel.GetK8SNamespace())

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sentinel.GetK8SName(),
			Namespace: sentinel.GetK8SNamespace(),
			Labels: labels.New().
				WorkspaceID(sentinel.GetWorkspaceId()).
				ProjectID(sentinel.GetProjectId()).
				EnvironmentID(sentinel.GetEnvironmentId()).
				SentinelID(sentinel.GetSentinelId()).
				ComponentSentinel().
				ManagedByKrane(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.P(sentinel.GetReplicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.New().
					SentinelID(sentinel.GetSentinelId()),
			},

			MinReadySeconds: 30,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels.New().SentinelID(sentinel.GetSentinelId()),
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
							{Name: "UNKEY_REGION", Value: r.region},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: 8040,
							Name:          "sentinel",
						}},
						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/_unkey/internal/health",
									Port: intstr.FromInt(8040),
								},
							},
							InitialDelaySeconds: 10,
							PeriodSeconds:       10,
							TimeoutSeconds:      5,
							FailureThreshold:    3,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/_unkey/internal/health",
									Port: intstr.FromInt(8040),
								},
							},
							InitialDelaySeconds: 5,
							PeriodSeconds:       5,
							TimeoutSeconds:      3,
							FailureThreshold:    2,
						},
						Resources: corev1.ResourceRequirements{
							// nolint:exhaustive
							//	Limits: corev1.ResourceList{
							//		corev1.ResourceCPU:              *resource.NewMilliQuantity(sentinel.GetCpuMillicores(), resource.BinarySI),
							//		corev1.ResourceMemory:           *resource.NewQuantity(sentinel.GetMemoryMib(), resource.BinarySI),
							//		corev1.ResourceEphemeralStorage: *resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
							//	},
							// nolint:exhaustive
							//	Requests: corev1.ResourceList{
							//		corev1.ResourceCPU:              *resource.NewMilliQuantity(sentinel.GetCpuMillicores(), resource.BinarySI),
							//		corev1.ResourceMemory:           *resource.NewQuantity(sentinel.GetMemoryMib(), resource.BinarySI),
							//		corev1.ResourceEphemeralStorage: *resource.NewQuantity(5*1024*1024*1024, resource.BinarySI),
							//	},
						},
					}},
				},
			},
		},
	}

	// Check if the deployment already exists, if not create a new one
	found, err := client.Get(ctx, sentinel.GetK8SName(), metav1.GetOptions{})

	if err == nil {
		foundBytes, err := json.Marshal(found.Spec)
		if err != nil {
			return nil, err
		}
		wantBytes, err := json.Marshal(deployment.Spec)
		if err != nil {
			return nil, err
		}

		if sha256.Sum256(foundBytes) != sha256.Sum256(wantBytes) {
			r.logger.Info("sentinel spec has changed, updating", "name", sentinel.GetK8SName())
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

func (r *Reconciler) ensureServiceExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel, deployment *appsv1.Deployment) (*corev1.Service, error) {
	client := r.clientSet.CoreV1().Services(sentinel.GetK8SNamespace())

	desired := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      sentinel.GetK8SName(),
			Namespace: sentinel.GetK8SNamespace(),
			Labels: labels.New().
				WorkspaceID(sentinel.GetWorkspaceId()).
				ProjectID(sentinel.GetProjectId()).
				EnvironmentID(sentinel.GetEnvironmentId()).
				SentinelID(sentinel.GetSentinelId()).
				ManagedByKrane().
				ComponentSentinel(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       deployment.GetName(),
					UID:        deployment.UID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP, // Use ClusterIP for internal communication
			Selector: labels.New().SentinelID(sentinel.GetSentinelId()),
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
