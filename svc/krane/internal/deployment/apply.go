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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ApplyDeployment creates or updates a user workload as a Kubernetes Deployment.
//
// The method uses server-side apply to create or update the Deployment, enabling
// concurrent modifications from different sources without conflicts. After applying,
// it queries the resulting pods and reports their addresses and status to the control
// plane so the routing layer knows where to send traffic.
//
// Using Deployments instead of ReplicaSets gives us rolling updates: when the pod
// template changes (env overrides, resource defaults, security settings), pods are
// automatically rolled without requiring a user redeploy. The rolling update strategy
// uses MaxUnavailable=0, MaxSurge=1 to ensure zero-downtime updates.
//
// ApplyDeployment validates all required fields and returns an error if any are missing
// or invalid: WorkspaceId, ProjectId, EnvironmentId, DeploymentId, K8sNamespace, K8sName,
// and Image must be non-empty; Replicas must be >= 0; CpuMillicores and MemoryMib must be > 0.
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
		assert.GreaterOrEqual(req.GetReplicas(), int32(0), "Replicas must be greater than or equal to 0"),
		assert.Greater(req.GetCpuMillicores(), int64(0), "CPU millicores must be greater than 0"),
		assert.Greater(req.GetMemoryMib(), int64(0), "MemoryMib must be greater than 0"),
	)
	if err != nil {
		return err
	}

	if err := c.ensureNamespaceExists(ctx, req.GetK8SNamespace()); err != nil {
		return err
	}

	// Ensure registry pull secret exists in the namespace
	if err := c.ensureRegistryPullSecret(ctx, req.GetK8SNamespace()); err != nil {
		return fmt.Errorf("failed to ensure registry pull secret: %w", err)
	}

	// Decrypt secrets at deploy time
	plaintext, err := c.decryptSecrets(ctx, req.GetEncryptedEnvironmentVariables(), req.GetEnvironmentId())
	if err != nil {
		return fmt.Errorf("failed to decrypt secrets: %w", err)
	}

	// Determine secret and SA names upfront so the Deployment pod spec can reference them.
	// The actual resources are created after the Deployment so we can set ownerReferences.
	secretName := ""
	saName := ""
	if len(plaintext) > 0 {
		secretName = deploymentSecretName(req.GetDeploymentId())
		saName = fmt.Sprintf("deploy-%s", sanitizeForK8s(req.GetDeploymentId()))
	}

	usedLabels := labels.New().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		AppID(req.GetAppId()).
		EnvironmentID(req.GetEnvironmentId()).
		DeploymentID(req.GetDeploymentId()).
		BuildID(req.GetBuildId()).
		ManagedByKrane().
		ComponentDeployment()

	container := corev1.Container{
		Image:           req.GetImage(),
		Name:            "deployment",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{Name: "PORT", Value: strconv.Itoa(int(req.GetPort()))},
			{Name: "UNKEY_WORKSPACE_ID", Value: req.GetWorkspaceId()},
			{Name: "UNKEY_PROJECT_ID", Value: req.GetProjectId()},
			{Name: "UNKEY_ENVIRONMENT_ID", Value: req.GetEnvironmentId()},
			{Name: "UNKEY_DEPLOYMENT_ID", Value: req.GetDeploymentId()},
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
	if secretName != "" {
		container.EnvFrom = []corev1.EnvFromSource{{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
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

	if saName != "" {
		podSpec.ServiceAccountName = saName
	}

	if c.registry != nil {
		podSpec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: registryPullSecretName}}
	}

	desired := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.GetK8SName(),
			Namespace: req.GetK8SNamespace(),
			Labels:    usedLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.P(req.GetReplicas()),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.New().DeploymentID(req.GetDeploymentId()),
			},
			MinReadySeconds: 30,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 0, StrVal: ""},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1, StrVal: ""},
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("%s-", req.GetK8SName()),
					Labels:       usedLabels,
				},
				Spec: podSpec,
			},
		},
	}

	client := c.clientSet.AppsV1().Deployments(req.GetK8SNamespace())

	patch, err := json.Marshal(desired)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	// Apply the Deployment first so we get its UID for ownerReferences.
	// Pods will retry until the secret/SA exist (created immediately after).
	applied, err := client.Patch(ctx, req.GetK8SName(), types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
	if err != nil {
		return fmt.Errorf("failed to apply deployment: %w", err)
	}

	// Create owned resources (Secret, SA, Role, RoleBinding) with ownerReferences
	// so K8s garbage-collects them when the Deployment is deleted.
	ownerRef := metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "Deployment",
		Name:               applied.Name,
		UID:                applied.UID,
		Controller:         ptr.P(true),
		BlockOwnerDeletion: ptr.P(true),
	}

	if secretName != "" {
		if err := c.ensureDeploymentSecret(ctx, req.GetK8SNamespace(), req.GetDeploymentId(), plaintext, ownerRef); err != nil {
			return fmt.Errorf("failed to ensure deployment secret: %w", err)
		}
		if err := c.ensureDeploymentRBAC(ctx, req.GetK8SNamespace(), req.GetDeploymentId(), secretName, ownerRef); err != nil {
			return fmt.Errorf("failed to ensure deployment RBAC: %w", err)
		}
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
