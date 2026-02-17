package sentinel

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	sentinelcfg "github.com/unkeyed/unkey/svc/sentinel"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	logger.Info("applying sentinel",
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

	err = c.ensurePDBExists(ctx, req, sentinel)
	if err != nil {
		return err
	}

	_, err = c.ensureGossipServiceExists(ctx, req)
	if err != nil {
		return err
	}

	err = c.ensureGossipCiliumPolicyExists(ctx, req)
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
		logger.Error("failed to reconcile sentinel", "sentinel_id", req.GetSentinelId(), "error", err)
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

	configEnv, err := toml.Marshal(sentinelcfg.Config{
		SentinelID:    sentinel.GetSentinelId(),
		WorkspaceID:   sentinel.GetWorkspaceId(),
		EnvironmentID: sentinel.GetEnvironmentId(),
		Region:        c.region,
		HttpPort:      SentinelPort,
		Database: config.DatabaseConfig{
			Primary:         "${UNKEY_DATABASE_PRIMARY}",
			ReadonlyReplica: "${UNKEY_DATABASE_REPLICA}",
		},
		ClickHouse: sentinelcfg.ClickHouseConfig{
			URL: "${UNKEY_CLICKHOUSE_URL}",
		},
		PrometheusPort: 0,
		Gossip:         nil,
		Tracing:        nil,
		Logging: config.LoggingConfig{
			SampleRate:    1.0,
			SlowThreshold: time.Second,
		},
	})
	if err != nil {
		return nil, err
	}

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
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: ptr.P(intstr.FromInt(0)),
					MaxSurge:       ptr.P(intstr.FromInt(1)),
				},
			},
			MinReadySeconds: 30,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"reloader.stakater.com/auto": "true",
					},
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
										Name: "otel",
									},
									Optional: ptr.P(true),
								},
							},
						},

						Env: []corev1.EnvVar{
							{Name: "UNKEY_CONFIG_DATA", Value: string(configEnv)},
						},

						Ports: []corev1.ContainerPort{
							{ContainerPort: SentinelPort, Name: "sentinel"},
							{ContainerPort: GossipLANPort, Name: "gossip-lan", Protocol: corev1.ProtocolTCP},
							{ContainerPort: GossipLANPort, Name: "gossip-lan-udp", Protocol: corev1.ProtocolUDP},
						},

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

// ensurePDBExists creates or updates a PodDisruptionBudget for the sentinel.
// The PDB ensures at least one pod remains available during voluntary disruptions
// (node drains, rolling updates, etc.). It is owned by the Deployment for automatic cleanup.
func (c *Controller) ensurePDBExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel, deployment *appsv1.Deployment) error {
	client := c.clientSet.PolicyV1().PodDisruptionBudgets(NamespaceSentinel)

	//nolint:exhaustruct // k8s API types have many optional fields
	desired := &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
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
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: ptr.P(intstr.FromInt(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.New().SentinelID(sentinel.GetSentinelId()),
			},
		},
	}

	patch, err := json.Marshal(desired)
	if err != nil {
		return fmt.Errorf("failed to marshal pdb: %w", err)
	}

	_, err = client.Patch(ctx, sentinel.GetK8SName(), types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
	return err
}

// ensureGossipServiceExists creates or updates a headless Service for gossip LAN peer
// discovery. The Service uses clusterIP: None so that DNS resolves to individual pod IPs,
// allowing memberlist to discover all peers in the environment. The selector matches all
// sentinel pods in the environment (not just one k8sName) for cross-sentinel peer discovery.
func (c *Controller) ensureGossipServiceExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel) (*corev1.Service, error) {
	client := c.clientSet.CoreV1().Services(NamespaceSentinel)

	gossipName := fmt.Sprintf("%s-gossip-lan", sentinel.GetK8SName())

	desired := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      gossipName,
			Namespace: NamespaceSentinel,
			Labels: labels.New().
				WorkspaceID(sentinel.GetWorkspaceId()).
				ProjectID(sentinel.GetProjectId()).
				EnvironmentID(sentinel.GetEnvironmentId()).
				SentinelID(sentinel.GetSentinelId()).
				ComponentGossipLAN(),
			// No OwnerReferences: this Service is environment-scoped (selector matches all
			// sentinel pods in the environment), so it must not be owned by a single Deployment.
			// Krane manages its lifecycle via server-side apply.
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "None",
			Selector: labels.New().
				EnvironmentID(sentinel.GetEnvironmentId()).
				ComponentSentinel(),
			Ports: []corev1.ServicePort{
				{
					Name:       "gossip-lan",
					Port:       GossipLANPort,
					TargetPort: intstr.FromInt(GossipLANPort),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "gossip-lan-udp",
					Port:       GossipLANPort,
					TargetPort: intstr.FromInt(GossipLANPort),
					Protocol:   corev1.ProtocolUDP,
				},
			},
		},
	}

	patch, err := json.Marshal(desired)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gossip service: %w", err)
	}

	return client.Patch(ctx, gossipName, types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
}

// ensureGossipCiliumPolicyExists creates or updates a CiliumNetworkPolicy that allows
// gossip traffic (TCP+UDP on GossipLANPort) between sentinel pods in the same environment.
func (c *Controller) ensureGossipCiliumPolicyExists(ctx context.Context, sentinel *ctrlv1.ApplySentinel) error {
	policyName := fmt.Sprintf("%s-gossip-lan", sentinel.GetK8SName())

	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      policyName,
				"namespace": NamespaceSentinel,
				"labels": labels.New().
					WorkspaceID(sentinel.GetWorkspaceId()).
					ProjectID(sentinel.GetProjectId()).
					EnvironmentID(sentinel.GetEnvironmentId()).
					SentinelID(sentinel.GetSentinelId()).
					ComponentGossipLAN(),
				// No ownerReferences: this policy is environment-scoped (selects all sentinel
				// pods in the environment), so it must not be owned by a single Deployment.
				// Krane manages its lifecycle via server-side apply.
			},
			"spec": map[string]interface{}{
				"endpointSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						labels.LabelKeyEnvironmentID: sentinel.GetEnvironmentId(),
						labels.LabelKeyComponent:     "sentinel",
					},
				},
				"ingress": []interface{}{
					map[string]interface{}{
						"fromEndpoints": []interface{}{
							map[string]interface{}{
								"matchLabels": map[string]interface{}{
									labels.LabelKeyEnvironmentID: sentinel.GetEnvironmentId(),
									labels.LabelKeyComponent:     "sentinel",
								},
							},
						},
						"toPorts": []interface{}{
							map[string]interface{}{
								"ports": []interface{}{
									map[string]interface{}{
										"port":     strconv.Itoa(GossipLANPort),
										"protocol": "TCP",
									},
									map[string]interface{}{
										"port":     strconv.Itoa(GossipLANPort),
										"protocol": "UDP",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumnetworkpolicies",
	}

	_, err := c.dynamicClient.Resource(gvr).Namespace(NamespaceSentinel).Apply(
		ctx,
		policyName,
		policy,
		metav1.ApplyOptions{FieldManager: fieldManagerKrane},
	)
	if err != nil {
		return fmt.Errorf("failed to apply gossip cilium network policy: %w", err)
	}

	return nil
}
