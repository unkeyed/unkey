package deployment

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/assert"
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

// ApplyDeployment creates or updates a user workload as a Kubernetes ReplicaSet.
//
// The method uses server-side apply to create or update the ReplicaSet, enabling
// concurrent modifications from different sources without conflicts. After applying,
// it queries the resulting pods and reports their addresses and status to the control
// plane so the routing layer knows where to send traffic.
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

	usedLabels := labels.New().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		EnvironmentID(req.GetEnvironmentId()).
		DeploymentID(req.GetDeploymentId()).
		BuildID(req.GetBuildId()).
		ManagedByKrane().
		ComponentDeployment().
		Inject()

	container := corev1.Container{
		Image:           req.GetImage(),
		Name:            "deployment",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         req.GetCommand(),
		Env:             buildDeploymentEnv(req),
		Ports: []corev1.ContainerPort{{
			ContainerPort: req.GetPort(),
			Name:          "deployment",
		}},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", req.GetCpuMillicores())),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", req.GetMemoryMib())),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", req.GetCpuMillicores())),
				corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", req.GetMemoryMib())),
			},
		},
	}

	// Configure healthcheck probes if provided
	if hc := unmarshalHealthcheck(req.GetHealthcheck()); hc != nil {
		httpGet := &corev1.HTTPGetAction{
			Path: hc.Path,
			Port: intstr.FromInt32(req.GetPort()),
		}
		container.LivenessProbe = &corev1.Probe{
			ProbeHandler:        corev1.ProbeHandler{HTTPGet: httpGet},
			InitialDelaySeconds: int32(hc.InitialDelaySeconds),
			PeriodSeconds:       int32(hc.IntervalSeconds),
			TimeoutSeconds:      int32(hc.TimeoutSeconds),
			FailureThreshold:    int32(hc.FailureThreshold),
		}
		container.ReadinessProbe = &corev1.Probe{
			ProbeHandler:        corev1.ProbeHandler{HTTPGet: httpGet},
			InitialDelaySeconds: int32(hc.InitialDelaySeconds),
			PeriodSeconds:       int32(hc.IntervalSeconds),
			TimeoutSeconds:      int32(hc.TimeoutSeconds),
			FailureThreshold:    int32(hc.FailureThreshold),
		}
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
					GenerateName: fmt.Sprintf("%s-", req.GetK8SName()),
					Labels:       usedLabels,
				},
				Spec: corev1.PodSpec{
					RuntimeClassName:          ptr.P(runtimeClassGvisor),
					RestartPolicy:             mapRestartPolicy(req.GetRestartPolicy()),
					Tolerations:               []corev1.Toleration{untrustedToleration},
					TopologySpreadConstraints: deploymentTopologySpread(req.GetDeploymentId()),
					Affinity:                  deploymentAffinity(req.GetEnvironmentId()),
					Containers:                []corev1.Container{container},
				},
			},
		},
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

// buildDeploymentEnv constructs the environment variables injected into deployment
// containers. It includes the PORT, workspace/project/environment/deployment IDs,
// and optionally the base64-encoded encrypted environment variables if present.
func buildDeploymentEnv(req *ctrlv1.ApplyDeployment) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "PORT", Value: strconv.Itoa(int(req.GetPort()))},
		{Name: "UNKEY_WORKSPACE_ID", Value: req.GetWorkspaceId()},
		{Name: "UNKEY_PROJECT_ID", Value: req.GetProjectId()},
		{Name: "UNKEY_ENVIRONMENT_ID", Value: req.GetEnvironmentId()},
		{Name: "UNKEY_DEPLOYMENT_ID", Value: req.GetDeploymentId()},
	}

	if len(req.GetEncryptedEnvironmentVariables()) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "UNKEY_ENCRYPTED_ENV",
			Value: base64.StdEncoding.EncodeToString(req.GetEncryptedEnvironmentVariables()),
		})
	}

	return env
}

// mapRestartPolicy converts the string restart policy from the proto to a K8s RestartPolicy enum.
func mapRestartPolicy(policy string) corev1.RestartPolicy {
	switch policy {
	case "on-failure":
		return corev1.RestartPolicyOnFailure
	case "never":
		return corev1.RestartPolicyNever
	default:
		return corev1.RestartPolicyAlways
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
