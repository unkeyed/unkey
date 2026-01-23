package sentinel

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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ApplySentinel creates or updates a sentinel as a Kubernetes Deployment with a Service.
//
// Sentinels are infrastructure proxies that route traffic to user deployments within
// an environment. Each sentinel gets both a Deployment (for the actual pods) and a
// ClusterIP Service (for stable in-cluster addressing). The Service is owned by the
// Deployment, so deleting the Deployment automatically cleans up the Service.
//
// ApplySentinel reports the available replica count back to the control plane after
// applying, so the platform knows when the sentinel is ready to receive traffic.
func (c *Controller) ApplySentinel(ctx context.Context, req *ctrlv1.ApplySentinel) error {
	c.logger.Info("applying sentinel",
		"namespace", NamespaceSentinel,
		"name", req.GetK8SName(),
		"sentinel_id", req.GetSentinelId(),
	)

	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetSentinelId(), "Sentinel ID is required"),
		assert.NotEmpty(req.GetK8SName(), "K8s CRD name is required"),
		assert.NotEmpty(req.GetImage(), "Image is required"),
		assert.GreaterOrEqual(req.GetReplicas(), int32(0), "Replicas must be greater than or equal to 0"),
		assert.Greater(req.GetCpuMillicores(), int64(0), "CPU millicores must be greater than 0"),
		assert.Greater(req.GetMemoryMib(), int64(0), "MemoryMib must be greater than 0"),
	)
	if err != nil {
		return err
	}

	if err := c.ensureNamespaceExists(ctx); err != nil {
		return err
	}

	sentinel, err := c.ensureSentinelExists(ctx, req)
	if err != nil {
		return err
	}

	_, err = c.ensureServiceExists(ctx, req, sentinel)
	if err != nil {
		return err
	}

	var health ctrlv1.Health
	if req.GetReplicas() == 0 {
		health = ctrlv1.Health_HEALTH_PAUSED
	} else if sentinel.Status.AvailableReplicas > 0 {
		health = ctrlv1.Health_HEALTH_HEALTHY
	} else {
		health = ctrlv1.Health_HEALTH_UNHEALTHY
	}

	err = c.reportSentinelStatus(ctx, &ctrlv1.ReportSentinelStatusRequest{
		K8SName:           req.GetK8SName(),
		AvailableReplicas: sentinel.Status.AvailableReplicas,
		Health:            health,
	})
	if err != nil {
		c.logger.Error("failed to reconcile sentinel", "sentinel_id", req.GetSentinelId(), "error", err)
		return err
	}

	return nil
}

// ensureNamespaceExists creates the sentinel namespace if it doesn't already exist.
func (c *Controller) ensureNamespaceExists(ctx context.Context) error {
	_, err := c.clientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: NamespaceSentinel,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// ensureSentinelExists creates or updates the sentinel's Kubernetes Deployment using
// server-side apply. Returns the resulting Deployment so the caller can extract
// its UID for setting owner references on related resources.
func (c *Controller) ensureSentinelExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel) (*appsv1.Deployment, error) {
	client := c.clientSet.AppsV1().Deployments(NamespaceSentinel)

	desired := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sentinel.GetK8SName(),
			Namespace: NamespaceSentinel,
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
					Labels: labels.New().
						WorkspaceID(sentinel.GetWorkspaceId()).
						EnvironmentID(sentinel.GetEnvironmentId()).
						SentinelID(sentinel.GetSentinelId()).
						ComponentSentinel(),
				},
				Spec: corev1.PodSpec{
					RestartPolicy:             corev1.RestartPolicyAlways,
					Tolerations:               []corev1.Toleration{sentinelToleration},
					TopologySpreadConstraints: sentinelTopologySpread(sentinel.GetSentinelId()),
					Containers: []corev1.Container{{
						Image:           sentinel.GetImage(),
						Name:            "sentinel",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Args:            []string{"run", "sentinel"},

						EnvFrom: []corev1.EnvFromSource{
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "database",
									},
								},
							},
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "clickhouse",
									},
								},
							},
							{
								SecretRef: &corev1.SecretEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "otel",
									},
									Optional: ptr.P(true),
								},
							},
						},

						Env: []corev1.EnvVar{
							{Name: "UNKEY_HTTP_PORT", Value: strconv.Itoa(SentinelPort)},
							{Name: "UNKEY_WORKSPACE_ID", Value: sentinel.GetWorkspaceId()},
							{Name: "UNKEY_PROJECT_ID", Value: sentinel.GetProjectId()},
							{Name: "UNKEY_ENVIRONMENT_ID", Value: sentinel.GetEnvironmentId()},
							{Name: "UNKEY_SENTINEL_ID", Value: sentinel.GetSentinelId()},
							{Name: "UNKEY_REGION", Value: c.region},
						},

						Ports: []corev1.ContainerPort{{
							ContainerPort: SentinelPort,
							Name:          "sentinel",
						}},

						LivenessProbe: &corev1.Probe{
							ProbeHandler: corev1.ProbeHandler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/_unkey/internal/health",
									Port: intstr.FromInt(SentinelPort),
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
									Port: intstr.FromInt(SentinelPort),
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

	patch, err := json.Marshal(desired)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment: %w", err)
	}

	return client.Patch(ctx, sentinel.GetK8SName(), types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
}

// ensureServiceExists creates or updates the ClusterIP Service that provides stable
// addressing for the sentinel's pods. The Service is owned by the Deployment, which
// means Kubernetes garbage collection will delete the Service when the Deployment
// is deleted.
func (c *Controller) ensureServiceExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel, deployment *appsv1.Deployment) (*corev1.Service, error) {
	client := c.clientSet.CoreV1().Services(NamespaceSentinel)

	desired := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sentinel.GetK8SName(),
			Namespace: NamespaceSentinel,
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
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels.New().SentinelID(sentinel.GetSentinelId()),
			Ports: []corev1.ServicePort{
				{
					Port:       SentinelPort,
					TargetPort: intstr.FromInt(SentinelPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	patch, err := json.Marshal(desired)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service: %w", err)
	}

	return client.Patch(ctx, sentinel.GetK8SName(), types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
}

// sentinelTopologySpread returns topology spread constraints for sentinel pods.
// Spreads pods evenly across availability zones with maxSkew of 1.
func sentinelTopologySpread(sentinelID string) []corev1.TopologySpreadConstraint {
	return []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       topologyKeyZone,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: labels.New().SentinelID(sentinelID),
			},
		},
	}
}
