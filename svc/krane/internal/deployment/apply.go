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
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/krane/pkg/metrics"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// ApplyDeployment creates or updates a user workload as a Kubernetes Deployment
// with an associated HorizontalPodAutoscaler (HPA) and PodDisruptionBudget (PDB).
//
// The method uses server-side apply to update the Deployment. spec.replicas
// and spec.paused are omitted so the HPA owns the replica count and the
// rollout gate owns when a template change rolls out. A legacy ReplicaSet with
// the same name is adopted by the Deployment; see [Controller.createDeployment].
//
// After applying, it queries the resulting pods and reports their addresses and status
// to the control plane so the routing layer knows where to send traffic.
//
// ApplyDeployment validates all required fields and returns an error if any are missing
// or invalid: WorkspaceId, ProjectId, EnvironmentId, DeploymentId, K8sNamespace, K8sName,
// and Image must be non-empty; CpuMillicores and MemoryMib must be > 0.
//
// The namespace is created automatically if it doesn't exist. After the
// Deployment is applied a CiliumNetworkPolicy is installed in the same
// namespace, owned by the Deployment, that permits ingress only from
// frontline pods on the deployment's container port. Pods run under the
// configured RuntimeClass, gVisor in production, since they execute untrusted
// user code, and are scheduled on Karpenter-managed untrusted nodes with
// node- and zone-spread constraints so replicas don't stack on a single node.
func (c *Controller) ApplyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) (retErr error) {
	defer func() { metrics.RecordReconcile("deployment", "apply", retErr) }()
	logger.Info(
		"applying deployment",
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
		assert.Greater(req.GetPort(), int32(0), "Port must be greater than 0"),
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

	desired := c.buildDeployment(req, hasSecrets)

	// Create the Secret and ServiceAccount before the Deployment so they
	// exist by the time pods are scheduled. This prevents the
	// "serviceaccount not found" race condition. We patch ownerReferences
	// onto them after the Deployment is created so K8s still garbage-collects them.
	if hasSecrets {
		if err := c.ensureDeploymentSecret(ctx, req.GetK8SNamespace(), req.GetDeploymentId(), plaintext); err != nil {
			return fmt.Errorf("failed to ensure deployment secret: %w", err)
		}

		if err := c.ensureDeploymentServiceAccount(ctx, req.GetK8SNamespace(), req.GetDeploymentId()); err != nil {
			return fmt.Errorf("failed to ensure deployment service account: %w", err)
		}
	}

	applied, err := c.applyDeploymentObject(ctx, desired)
	if err != nil {
		return err
	}
	owner := deploymentOwnerRef(applied)

	// Patch ownerReferences onto the Secret and SA so K8s garbage-collects
	// them when the Deployment is deleted.
	if hasSecrets {
		resName := deploymentResourcePrefix(req.GetDeploymentId())
		if err := c.patchOwnerRef(ctx, req.GetK8SNamespace(), resName, owner); err != nil {
			return fmt.Errorf("failed to patch owner references: %w", err)
		}
	}

	if err := c.ensureHPAExists(ctx, req, owner); err != nil {
		return fmt.Errorf("failed to ensure HPA: %w", err)
	}

	if err := c.ensureCiliumNetworkPolicy(ctx, req, owner); err != nil {
		return fmt.Errorf("failed to ensure cilium network policy: %w", err)
	}

	status, err := c.buildDeploymentStatus(ctx, workload{
		namespace:    applied.Namespace,
		k8sName:      applied.Name,
		deploymentID: req.GetDeploymentId(),
		selector:     applied.Spec.Selector,
	})
	if err != nil {
		return err
	}

	err = c.reportDeploymentStatus(ctx, status)
	if err != nil {
		logger.Error("failed to report deployment status", "deployment_id", req.GetDeploymentId(), "error", err)
		return err
	}

	if err := c.ensurePodDisruptionBudget(ctx, req, owner); err != nil {
		logger.Error("failed to ensure pod disruption budget", "deployment_id", req.GetDeploymentId(), "error", err)
	}

	return nil
}

// applyDeploymentObject creates the Deployment if it does not exist, then
// server-side applies the desired spec.
//
// The apply forces ownership because the creating request, made by the
// rollout gate's field manager, also set every field krane renders. Krane
// stays the only writer of those fields; the gate keeps spec.paused and the
// seeded spec.replicas, which krane never sends.
func (c *Controller) applyDeploymentObject(ctx context.Context, desired *appsv1.Deployment) (*appsv1.Deployment, error) {
	client := c.clientSet.AppsV1().Deployments(desired.Namespace)

	_, err := client.Get(ctx, desired.Name, metav1.GetOptions{})
	switch {
	case k8serrors.IsNotFound(err):
		if err := c.createDeployment(ctx, desired); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	patch, err := json.Marshal(desired)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment: %w", err)
	}

	applied, err := client.Patch(ctx, desired.Name, types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
		Force:        new(true),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply deployment: %w", err)
	}

	return applied, nil
}

// createDeployment creates the Deployment for a workload that has none.
//
// A new workload starts unpaused so its first pods come up at once; a paused
// Deployment without a matching ReplicaSet runs no pods. A legacy ReplicaSet
// with the same name is adopted instead: the Deployment starts paused, with
// the ReplicaSet's replica count, so adoption restarts no pods and does not
// scale the ReplicaSet to the default of one replica. If the rendered template
// differs from the legacy one, the rollout gate rolls the workload later.
func (c *Controller) createDeployment(ctx context.Context, desired *appsv1.Deployment) error {
	initial := desired.DeepCopy()

	legacy, err := c.clientSet.AppsV1().ReplicaSets(desired.Namespace).Get(ctx, desired.Name, metav1.GetOptions{})
	switch {
	case k8serrors.IsNotFound(err):
	case err != nil:
		return fmt.Errorf("failed to get legacy replicaset: %w", err)
	case isLegacyReplicaSet(legacy):
		initial.Spec.Replicas = legacy.Spec.Replicas
		initial.Spec.Paused = true
		logger.Info("adopting legacy replicaset", "namespace", desired.Namespace, "name", desired.Name)
	}

	_, err = c.clientSet.AppsV1().Deployments(desired.Namespace).Create(ctx, initial, metav1.CreateOptions{
		FieldManager: fieldManagerRolloutGate,
	})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	return nil
}

// buildDeployment renders the desired Deployment for a deployment request.
//
// It performs no I/O so the mapping from every ApplyDeployment proto field to
// the Kubernetes object is unit-testable without a cluster (see apply_test.go).
// hasSecrets indicates whether a per-deployment K8s Secret will be mounted via
// envFrom; the caller computes it from the decrypted environment variables.
//
// The pod template must stay identical to what krane rendered for legacy
// ReplicaSets, so a Deployment adopts an unchanged legacy ReplicaSet without
// replacing its pods.
func (c *Controller) buildDeployment(req *ctrlv1.ApplyDeployment, hasSecrets bool) *appsv1.Deployment {
	usedLabels := deploymentLabels(req).
		BuildID(req.GetBuildId()).
		Platform(c.platform)
	container := corev1.Container{
		Image:           req.GetImage(),
		Name:            "deployment",
		Command:         req.GetCommand(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{},
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
				corev1.ResourceCPU:              resource.MustParse(fmt.Sprintf("%dm", max(req.GetCpuMillicores()/resourceRequestFraction, 1))),
				corev1.ResourceMemory:           resource.MustParse(fmt.Sprintf("%dMi", max(req.GetMemoryMib()/resourceRequestFraction, 1))),
				corev1.ResourceEphemeralStorage: resource.MustParse(fmt.Sprintf("%dMi", max(defaultContainerEphemeralStorageMib/resourceRequestFraction, 1))),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse(fmt.Sprintf("%dm", req.GetCpuMillicores())),
				corev1.ResourceMemory:           resource.MustParse(fmt.Sprintf("%dMi", req.GetMemoryMib())),
				corev1.ResourceEphemeralStorage: resource.MustParse(fmt.Sprintf("%dMi", defaultContainerEphemeralStorageMib)),
			},
		},
	}

	var volumes []corev1.Volume

	// Add ephemeral volume when storage is configured.
	// Uses a Kubernetes generic ephemeral volume backed by the configured StorageClass.
	// The volume is auto-created when the pod starts and auto-deleted when the pod dies.
	if es := req.GetEphemeralStorage(); es != nil && es.GetSizeMib() > 0 {
		volumes = append(volumes, corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				Ephemeral: &corev1.EphemeralVolumeSource{
					VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							StorageClassName: new(c.storageClassName),
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dMi", es.GetSizeMib())),
								},
							},
						},
					},
				},
			},
		})

		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "data",
			MountPath: "/data",
		})
		container.Env = append(container.Env, corev1.EnvVar{Name: "UNKEY_EPHEMERAL_DISK_PATH", Value: "/data"})
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

	runtimeClass := new(runtimeClassGvisor)
	if c.disableGvisor {
		runtimeClass = nil
	}

	podSpec := corev1.PodSpec{
		RuntimeClassName:             runtimeClass,
		RestartPolicy:                corev1.RestartPolicyAlways,
		AutomountServiceAccountToken: new(false),
		EnableServiceLinks:           new(false),
		NodeSelector:                 map[string]string{nodeClassLabelKey: CustomerNodeClass},
		Tolerations:                  []corev1.Toleration{untrustedToleration},
		TopologySpreadConstraints:    deploymentTopologySpread(req.GetDeploymentId(), req.GetAutoscaling().GetMaxReplicas()),
		Containers:                   []corev1.Container{container},
	}

	if len(volumes) > 0 {
		podSpec.Volumes = volumes
	}

	if hasSecrets {
		podSpec.ServiceAccountName = deploymentResourcePrefix(req.GetDeploymentId())
	}

	podSpec.ImagePullSecrets = c.imagePullSecrets

	maxSurge := intstr.FromInt32(1)
	maxUnavailable := intstr.FromInt32(0)

	return &appsv1.Deployment{
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
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.New().DeploymentID(req.GetDeploymentId()),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: fmt.Sprintf("%s-", req.GetK8SName()),
					Labels:       usedLabels,
				},
				Spec: podSpec,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &maxSurge,
					MaxUnavailable: &maxUnavailable,
				},
			},
			RevisionHistoryLimit: new(revisionHistoryLimit),
		},
	}
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
// workload's Deployment using the autoscaling policy from the control plane.
// The HPA is owned by the Deployment for automatic garbage collection.
func (c *Controller) ensureHPAExists(ctx context.Context, req *ctrlv1.ApplyDeployment, owner metav1.OwnerReference) error {
	client := c.clientSet.AutoscalingV2().HorizontalPodAutoscalers(req.GetK8SNamespace())

	policy := req.GetAutoscaling()
	minReplicas := int32(max(policy.GetMinReplicas(), 1))
	maxReplicas := max(int32(policy.GetMaxReplicas()), minReplicas)
	cpuThreshold := new(int32(defaultCPUTargetUtilization))

	var metrics []autoscalingv2.MetricSpec

	if policy.CpuThreshold != nil {
		cpuThreshold = policy.CpuThreshold
	}
	if policy.MemoryThreshold != nil {
		metrics = append(
			metrics,
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
	metrics = append(
		metrics,
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
			Name:            req.GetK8SName(),
			Namespace:       req.GetK8SNamespace(),
			Labels:          deploymentLabels(req),
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		//nolint:exhaustruct
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       req.GetK8SName(),
			},
			MinReplicas: new(minReplicas),
			MaxReplicas: maxReplicas,
			//nolint:exhaustruct
			Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
				//nolint:exhaustruct
				ScaleDown: &autoscalingv2.HPAScalingRules{
					StabilizationWindowSeconds: new(scaleDownStabilizationSeconds),
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

// ensurePodDisruptionBudget creates or updates a PodDisruptionBudget that caps
// voluntary disruptions (Karpenter consolidation, node drains, cluster upgrades)
// of the deployment's pods. The PDB is owned by the Deployment for automatic
// garbage collection.
func (c *Controller) ensurePodDisruptionBudget(ctx context.Context, req *ctrlv1.ApplyDeployment, owner metav1.OwnerReference) error {
	client := c.clientSet.PolicyV1().PodDisruptionBudgets(req.GetK8SNamespace())

	desired := buildPodDisruptionBudget(req, owner)

	patch, err := json.Marshal(desired)
	if err != nil {
		return fmt.Errorf("failed to marshal PDB: %w", err)
	}

	_, err = client.Patch(ctx, req.GetK8SName(), types.ApplyPatchType, patch, metav1.PatchOptions{
		FieldManager: fieldManagerKrane,
	})
	return err
}

// buildPodDisruptionBudget constructs the PDB for a deployment, owned by
// owner and selecting pods by deployment ID.
//
// maxUnavailable is the absolute integer 1, deliberately not a percentage: a
// percentage maxUnavailable rounds down, so anything under 100% of a
// single-replica deployment computes disruptionsAllowed=0 and would deadlock
// node drains on that pod. With an absolute 1, the budget is a no-op for a
// 1-replica deployment (its single pod stays evictable, since it is not HA to
// begin with) and keeps at least N-1 pods running for multi-replica deployments
// through planned node churn. If a multi-replica deployment is already degraded,
// the PDB can still block voluntary eviction; the untrusted NodePool
// terminationGracePeriod is the bounded escape hatch for that case.
func buildPodDisruptionBudget(req *ctrlv1.ApplyDeployment, owner metav1.OwnerReference) *policyv1.PodDisruptionBudget {
	maxUnavailable := intstr.FromInt32(1)
	alwaysAllow := policyv1.AlwaysAllow

	//nolint:exhaustruct
	pdb := &policyv1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            req.GetK8SName(),
			Namespace:       req.GetK8SNamespace(),
			Labels:          deploymentLabels(req),
			OwnerReferences: []metav1.OwnerReference{owner},
		},
		//nolint:exhaustruct
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable:             &maxUnavailable,
			UnhealthyPodEvictionPolicy: &alwaysAllow,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels.New().DeploymentID(req.GetDeploymentId()),
			},
		},
	}
	return pdb
}

func deploymentLabels(req *ctrlv1.ApplyDeployment) labels.Labels {
	return labels.New().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		AppID(req.GetAppId()).
		EnvironmentID(req.GetEnvironmentId()).
		DeploymentID(req.GetDeploymentId()).
		ManagedByKrane().
		ComponentDeployment()
}

func deploymentOwnerRef(d *appsv1.Deployment) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "Deployment",
		Name:               d.Name,
		UID:                d.UID,
		Controller:         new(true),
		BlockOwnerDeletion: new(true),
	}
}
