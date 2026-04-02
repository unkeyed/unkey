package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ApplyDeployment creates or updates a user workload as a Kubernetes ReplicaSet
// with an associated HorizontalPodAutoscaler (HPA).
//
// The method uses server-side apply to create or update the ReplicaSet.
// spec.replicas is omitted so the HPA owns the replica count. The HPA's
// minReplicas determines the minimum capacity; users should set this high
// enough to handle traffic during rollouts.
//
// After applying, it queries the resulting pods and reports their addresses and status
// to the control plane so the routing layer knows where to send traffic.
//
// ApplyDeployment validates all required fields and returns an error if any are missing
// or invalid: WorkspaceId, ProjectId, EnvironmentId, DeploymentId, K8sNamespace, K8sName,
// and Image must be non-empty; CpuMillicores and MemoryMib must be > 0.
//
// The namespace is created automatically if it doesn't exist, along with a
// CiliumNetworkPolicy restricting ingress to matching sentinels. Pods run with gVisor
// isolation (RuntimeClass "gvisor") since they execute untrusted user code, and are
// scheduled on Karpenter-managed untrusted nodes with zone-spread constraints.
func (c *Controller) ApplyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) error {
	logger.Info("applying deployment",
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
		assert.GreaterOrEqual(req.GetAutoscaling().GetMinReplicas(), uint32(1), "Autoscaling min_replicas must be at least 1"),
		assert.GreaterOrEqual(req.GetAutoscaling().GetMaxReplicas(), req.GetAutoscaling().GetMinReplicas(), "Autoscaling max_replicas must be >= min_replicas"),
	)
	if err != nil {
		return err
	}

	if err := c.ensureNamespaceExists(ctx, req.GetK8SNamespace()); err != nil {
		return err
	}

	if err := c.ensureRegistryPullSecret(ctx, req.GetK8SNamespace()); err != nil {
		return fmt.Errorf("failed to ensure registry pull secret: %w", err)
	}

	plaintext, err := c.decryptSecrets(ctx, req.GetEncryptedEnvironmentVariables(), req.GetEnvironmentId())
	if err != nil {
		return fmt.Errorf("failed to decrypt secrets: %w", err)
	}

	hasSecrets := len(plaintext) > 0

	usedLabels := labels.New().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		AppID(req.GetAppId()).
		EnvironmentID(req.GetEnvironmentId()).
		DeploymentID(req.GetDeploymentId()).
		BuildID(req.GetBuildId()).
		Platform(c.platform).
		ManagedByKrane().
		ComponentDeployment()

	container := corev1.Container{
		Image:           req.GetImage(),
		Name:            "deployment",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{Name: "PORT", Value: strconv.Itoa(int(req.GetPort()))},
			{Name: "UNKEY_DEPLOYMENT_ID", Value: req.GetDeploymentId()},
			{Name: "UNKEY_ENVIRONMENT_SLUG", Value: req.GetEnvironmentSlug()},
			{Name: "UNKEY_REGION", Value: req.GetRegion()},
			{Name: "UNKEY_GIT_COMMIT_SHA", Value: req.GetGitCommitSha()},
			{Name: "UNKEY_GIT_BRANCH", Value: req.GetGitBranch()},
			{Name: "UNKEY_GIT_REPO", Value: req.GetGitRepo()},
			{Name: "UNKEY_GIT_COMMIT_MESSAGE", Value: req.GetGitCommitMessage()},
			{Name: "UNKEY_INSTANCE_ID", ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"},
			}},
			// Override kubelet-injected K8s service env vars with empty strings.
			// These can't be suppressed via enableServiceLinks, but setting them
			// explicitly prevents leaking cluster internals to customer code.
			{Name: "KUBERNETES_SERVICE_HOST", Value: ""},
			{Name: "KUBERNETES_SERVICE_PORT", Value: ""},
			{Name: "KUBERNETES_SERVICE_PORT_HTTPS", Value: ""},
			{Name: "KUBERNETES_PORT", Value: ""},
			{Name: "KUBERNETES_PORT_443_TCP", Value: ""},
			{Name: "KUBERNETES_PORT_443_TCP_PROTO", Value: ""},
			{Name: "KUBERNETES_PORT_443_TCP_PORT", Value: ""},
			{Name: "KUBERNETES_PORT_443_TCP_ADDR", Value: ""},
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: req.GetPort(),
			Name:          "deployment",
		}},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", max(req.GetCpuMillicores()/resourceRequestFraction, 1))),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", max(req.GetMemoryMib()/resourceRequestFraction, 1))),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", req.GetCpuMillicores())),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", req.GetMemoryMib())),
			},
		},
	}

	// Configure healthcheck probes if provided
	if hc := unmarshalHealthcheck(req.GetHealthcheck()); hc != nil {
		handler := buildProbeHandler(hc, req.GetPort())
		probe := &corev1.Probe{
			ProbeHandler:        handler,
			InitialDelaySeconds: int32(hc.InitialDelaySeconds),
			PeriodSeconds:       int32(hc.IntervalSeconds),
			TimeoutSeconds:      int32(hc.TimeoutSeconds),
			FailureThreshold:    int32(hc.FailureThreshold),
		}
		container.LivenessProbe = probe
		container.ReadinessProbe = probe
	}

	// For non-SIGTERM shutdown signals, use a preStop lifecycle hook
	// since K8s always sends SIGTERM natively
	if req.GetShutdownSignal() != "" && req.GetShutdownSignal() != "SIGTERM" {
		container.Lifecycle = &corev1.Lifecycle{
			PreStop: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"kill", fmt.Sprintf("-%s", req.GetShutdownSignal()), "1"},
				},
			},
		}
	}

	// Mount the deployment secret as env vars if present
	if hasSecrets {
		container.EnvFrom = []corev1.EnvFromSource{{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: deploymentResourcePrefix(req.GetDeploymentId())},
			},
		}}
	}

	podSpec := corev1.PodSpec{
		RuntimeClassName:             ptr.P(runtimeClassGvisor),
		RestartPolicy:                corev1.RestartPolicyAlways,
		AutomountServiceAccountToken: ptr.P(false),
		EnableServiceLinks:           ptr.P(false),
		Tolerations:                  []corev1.Toleration{untrustedToleration},
		TopologySpreadConstraints:    deploymentTopologySpread(req.GetDeploymentId()),
		Affinity:                     deploymentAffinity(req.GetEnvironmentId()),
		Containers:                   []corev1.Container{container},
	}

	if hasSecrets {
		podSpec.ServiceAccountName = deploymentResourcePrefix(req.GetDeploymentId())
	}

	podSpec.ImagePullSecrets = c.imagePullSecrets

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
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.New().DeploymentID(req.GetDeploymentId()),
			},
			MinReadySeconds: 30,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("%s-", req.GetK8SName()),
					Labels:       usedLabels,
				},
				Spec: podSpec,
			},
		},
	}

	// Create the Secret and ServiceAccount before the ReplicaSet so they
	// exist by the time pods are scheduled. This prevents the
	// "serviceaccount not found" race condition. We patch ownerReferences
	// onto them after the RS is created so K8s still garbage-collects them.
	if hasSecrets {
		if err := c.ensureDeploymentSecret(ctx, req.GetK8SNamespace(), req.GetDeploymentId(), plaintext); err != nil {
			return fmt.Errorf("failed to ensure deployment secret: %w", err)
		}
		if err := c.ensureDeploymentServiceAccount(ctx, req.GetK8SNamespace(), req.GetDeploymentId()); err != nil {
			return fmt.Errorf("failed to ensure deployment service account: %w", err)
		}
	}

	client := c.clientSet.AppsV1().ReplicaSets(req.GetK8SNamespace())

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

	// Patch ownerReferences onto the Secret and SA so K8s garbage-collects
	// them when the ReplicaSet is deleted.
	if hasSecrets {
		ownerRef := metav1.OwnerReference{
			APIVersion:         "apps/v1",
			Kind:               "ReplicaSet",
			Name:               applied.Name,
			UID:                applied.UID,
			Controller:         ptr.P(true),
			BlockOwnerDeletion: ptr.P(true),
		}
		resName := deploymentResourcePrefix(req.GetDeploymentId())
		if err := c.patchOwnerRef(ctx, req.GetK8SNamespace(), resName, ownerRef); err != nil {
			return fmt.Errorf("failed to patch owner references: %w", err)
		}
	}

	if err := c.ensureHPAExists(ctx, req, applied); err != nil {
		return fmt.Errorf("failed to ensure HPA: %w", err)
	}

	status, err := c.buildDeploymentStatus(ctx, applied)
	if err != nil {
		return err
	}

	err = c.reportDeploymentStatus(ctx, status)
	if err != nil {
		logger.Error("failed to report deployment status", "deployment_id", req.GetDeploymentId(), "error", err)
		return err
	}

	return nil
}

// buildProbeHandler creates the K8s probe handler based on the healthcheck method.
// GET uses a native HTTPGetAction. POST uses an exec probe with wget since K8s
// doesn't support HTTP POST probes natively.
func buildProbeHandler(hc *dbtype.Healthcheck, port int32) corev1.ProbeHandler {
	if hc.Method == "POST" {
		return corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"wget", "--spider", "--post-data=", "-q",
					fmt.Sprintf("http://localhost:%d%s", port, hc.Path),
				},
			},
		}
	}
	return corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Path: hc.Path,
			Port: intstr.FromInt32(port),
		},
	}
}

// unmarshalHealthcheck deserializes the JSON-encoded healthcheck bytes from the proto.
// Returns nil if the input is nil or empty.
func unmarshalHealthcheck(data []byte) *dbtype.Healthcheck {
	if len(data) == 0 {
		return nil
	}
	var hc dbtype.Healthcheck
	if err := json.Unmarshal(data, &hc); err != nil {
		logger.Error("failed to unmarshal healthcheck", "error", err)
		return nil
	}
	return &hc
}

// ensureHPAExists creates or updates a HorizontalPodAutoscaler that scales the
// deployment's ReplicaSet using the autoscaling policy from the control plane.
// The HPA is owned by the ReplicaSet for automatic garbage collection.
func (c *Controller) ensureHPAExists(ctx context.Context, req *ctrlv1.ApplyDeployment, rs *appsv1.ReplicaSet) error {
	client := c.clientSet.AutoscalingV2().HorizontalPodAutoscalers(req.GetK8SNamespace())

	policy := req.GetAutoscaling()
	minReplicas := int32(max(policy.GetMinReplicas(), 1))
	maxReplicas := max(int32(policy.GetMaxReplicas()), minReplicas)
	cpuThreshold := ptr.P(int32(defaultCPUTargetUtilization))

	var metrics []autoscalingv2.MetricSpec

	if policy.CpuThreshold != nil {
		cpuThreshold = policy.CpuThreshold
	}
	if policy.MemoryThreshold != nil {
		metrics = append(metrics,
			//nolint:exhaustruct
			autoscalingv2.MetricSpec{
				Type: autoscalingv2.ResourceMetricSourceType,
				Resource: &autoscalingv2.ResourceMetricSource{
					Name: corev1.ResourceMemory,
					//nolint:exhaustruct
					Target: autoscalingv2.MetricTarget{
						Type:               autoscalingv2.UtilizationMetricType,
						AverageUtilization: policy.MemoryThreshold,
					},
				},
			},
		)
	}

	// CPU is always a scaling signal.
	metrics = append(metrics,
		//nolint:exhaustruct
		autoscalingv2.MetricSpec{
			Type: autoscalingv2.ResourceMetricSourceType,
			Resource: &autoscalingv2.ResourceMetricSource{
				Name: corev1.ResourceCPU,
				//nolint:exhaustruct
				Target: autoscalingv2.MetricTarget{
					Type:               autoscalingv2.UtilizationMetricType,
					AverageUtilization: cpuThreshold,
				},
			},
		},
	)

	//nolint:exhaustruct // k8s API types have many optional fields
	desired := &autoscalingv2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "autoscaling/v2",
			Kind:       "HorizontalPodAutoscaler",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.GetK8SName(),
			Namespace: req.GetK8SNamespace(),
			Labels: labels.New().
				WorkspaceID(req.GetWorkspaceId()).
				ProjectID(req.GetProjectId()).
				AppID(req.GetAppId()).
				EnvironmentID(req.GetEnvironmentId()).
				DeploymentID(req.GetDeploymentId()).
				ManagedByKrane().
				ComponentDeployment(),
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "apps/v1",
					Kind:               "ReplicaSet",
					Name:               rs.Name,
					UID:                rs.UID,
					Controller:         ptr.P(true),
					BlockOwnerDeletion: ptr.P(true),
				},
			},
		},
		//nolint:exhaustruct
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "ReplicaSet",
				Name:       req.GetK8SName(),
			},
			MinReplicas: ptr.P(minReplicas),
			MaxReplicas: maxReplicas,
			//nolint:exhaustruct
			Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
				//nolint:exhaustruct
				ScaleDown: &autoscalingv2.HPAScalingRules{
					StabilizationWindowSeconds: ptr.P(scaleDownStabilizationSeconds),
				},
			},
			Metrics: metrics,
		},
	}

	patch, err := json.Marshal(desired)
	if err != nil {
		return fmt.Errorf("failed to marshal HPA: %w", err)
	}

	_, err = client.Patch(ctx, req.GetK8SName(), types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
	return err
}
