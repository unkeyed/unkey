package mutator

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/unkeyed/unkey/go/apps/secrets-webhook/internal/services/registry"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const (
	// Volume name for the unkey-env binary
	unkeyEnvVolumeName = "unkey-env-bin"

	// Mount path for the unkey-env binary in containers
	unkeyEnvMountPath = "/unkey"

	// Binary name
	unkeyEnvBinary = "/unkey/unkey-env"
)

// Annotation keys (will be prefixed with AnnotationPrefix)
const (
	// Required annotations
	AnnotationDeploymentID = "deployment-id"

	// Optional annotations with defaults
	AnnotationProviderEndpoint = "provider-endpoint"
)

// Well-known Kubernetes paths
const (
	// ServiceAccountTokenPath is the standard K8s service account token location
	ServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// Config holds configuration for the mutator (webhook-level defaults).
type Config struct {
	UnkeyEnvImage    string
	AnnotationPrefix string

	// Default provider endpoint (can be overridden by pod annotation)
	DefaultProviderEndpoint string
}

// GetAnnotation returns the full annotation key for a given suffix.
func (c *Config) GetAnnotation(suffix string) string {
	return fmt.Sprintf("%s/%s", c.AnnotationPrefix, suffix)
}

// podConfig holds per-pod configuration loaded from annotations.
type podConfig struct {
	DeploymentID     string
	ProviderEndpoint string
}

// loadPodConfig loads configuration from pod annotations with defaults from webhook config.
func (m *Mutator) loadPodConfig(annotations map[string]string) (*podConfig, error) {
	cfg := &podConfig{}

	// Required: deployment-id
	cfg.DeploymentID = annotations[m.cfg.GetAnnotation(AnnotationDeploymentID)]
	if cfg.DeploymentID == "" {
		return nil, fmt.Errorf("missing required annotation: %s", m.cfg.GetAnnotation(AnnotationDeploymentID))
	}

	// Optional: provider-endpoint (default from webhook config)
	if val, ok := annotations[m.cfg.GetAnnotation(AnnotationProviderEndpoint)]; ok && val != "" {
		cfg.ProviderEndpoint = val
	} else {
		cfg.ProviderEndpoint = m.cfg.DefaultProviderEndpoint
	}

	return cfg, nil
}

// Mutator handles pod mutation for secrets injection.
type Mutator struct {
	cfg      *Config
	logger   logging.Logger
	registry *registry.Registry
}

// New creates a new pod mutator.
func New(cfg *Config, logger logging.Logger, reg *registry.Registry) *Mutator {
	return &Mutator{
		cfg:      cfg,
		logger:   logger,
		registry: reg,
	}
}

// Result contains the result of a mutation operation.
type Result struct {
	Mutated bool
	Patch   []byte
	Message string
}

// ShouldMutate checks if a pod should be mutated.
// The webhook's label selector already filters for unkey.com/inject=true,
// so we just check if the required deployment-id annotation is present.
func (m *Mutator) ShouldMutate(pod *corev1.Pod) bool {
	annotations := pod.GetAnnotations()
	if annotations == nil {
		return false
	}

	// Check for required deployment-id annotation
	deploymentID := annotations[m.cfg.GetAnnotation(AnnotationDeploymentID)]
	return deploymentID != ""
}

// Mutate transforms a pod to inject the unkey-env sidecar.
func (m *Mutator) Mutate(ctx context.Context, pod *corev1.Pod, namespace string) (*Result, error) {
	if !m.ShouldMutate(pod) {
		return &Result{
			Mutated: false,
			Message: "pod not annotated for injection",
		}, nil
	}

	annotations := pod.GetAnnotations()

	// Load configuration from annotations (with defaults from webhook config)
	podCfg, err := m.loadPodConfig(annotations)
	if err != nil {
		return nil, err
	}

	m.logger.Info("loaded pod config from annotations",
		"deployment_id", podCfg.DeploymentID,
		"provider_endpoint", podCfg.ProviderEndpoint,
	)

	// Build JSON patches
	var patches []map[string]interface{}

	// 1. Add init container to copy unkey-env binary
	initContainer := m.buildInitContainer()
	patches = append(patches, map[string]interface{}{
		"op":    "add",
		"path":  "/spec/initContainers/-",
		"value": initContainer,
	})

	// If initContainers doesn't exist, we need to create it first
	if len(pod.Spec.InitContainers) == 0 {
		patches = []map[string]interface{}{
			{
				"op":    "add",
				"path":  "/spec/initContainers",
				"value": []corev1.Container{initContainer},
			},
		}
	}

	// 2. Add emptyDir volume for unkey-env binary
	volume := m.buildVolume()
	if len(pod.Spec.Volumes) == 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/volumes",
			"value": []corev1.Volume{volume},
		})
	} else {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/volumes/-",
			"value": volume,
		})
	}

	// 3. Mutate each container
	for i, container := range pod.Spec.Containers {
		containerPatches, err := m.buildContainerPatches(ctx, i, &container, &pod.Spec, namespace, podCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to build patches for container %s: %w", container.Name, err)
		}
		patches = append(patches, containerPatches...)
	}

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal patches: %w", err)
	}

	return &Result{
		Mutated: true,
		Patch:   patchBytes,
		Message: fmt.Sprintf("injected unkey-env for deployment %s", podCfg.DeploymentID),
	}, nil
}

// buildInitContainer creates the init container that copies unkey-env binary.
func (m *Mutator) buildInitContainer() corev1.Container {
	return corev1.Container{
		Name:            "copy-unkey-env",
		Image:           m.cfg.UnkeyEnvImage,
		ImagePullPolicy: corev1.PullNever,
		Command:         []string{"cp", "/unkey-env", unkeyEnvBinary},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      unkeyEnvVolumeName,
				MountPath: unkeyEnvMountPath,
			},
		},
		Resources: corev1.ResourceRequirements{},
	}
}

// buildVolume creates the emptyDir volume for the unkey-env binary.
func (m *Mutator) buildVolume() corev1.Volume {
	return corev1.Volume{
		Name: unkeyEnvVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
	}
}

// buildContainerPatches creates JSON patches for a single container.
func (m *Mutator) buildContainerPatches(
	ctx context.Context,
	containerIndex int,
	container *corev1.Container,
	podSpec *corev1.PodSpec,
	namespace string,
	podCfg *podConfig,
) ([]map[string]interface{}, error) {
	var patches []map[string]interface{}
	basePath := fmt.Sprintf("/spec/containers/%d", containerIndex)

	// Add volume mount for unkey-env binary
	volumeMount := corev1.VolumeMount{
		Name:      unkeyEnvVolumeName,
		MountPath: unkeyEnvMountPath,
		ReadOnly:  true,
	}

	if len(container.VolumeMounts) == 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/volumeMounts", basePath),
			"value": []corev1.VolumeMount{volumeMount},
		})
	} else {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/volumeMounts/-", basePath),
			"value": volumeMount,
		})
	}

	// Add environment variables
	envVars := m.buildEnvVars(podCfg)
	for _, env := range envVars {
		if len(container.Env) == 0 {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  fmt.Sprintf("%s/env", basePath),
				"value": []corev1.EnvVar{env},
			})
		} else {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  fmt.Sprintf("%s/env/-", basePath),
				"value": env,
			})
		}
	}

	// Build the arguments for unkey-env (the original command to execute)
	var args []string

	// If container has no explicit command, fetch it from the image registry
	if len(container.Command) == 0 {
		m.logger.Info("container has no command, fetching from registry",
			"container", container.Name,
			"image", container.Image,
		)

		imageConfig, err := m.registry.GetImageConfig(ctx, namespace, container, podSpec)
		if err != nil {
			return nil, fmt.Errorf("failed to get image config for %s: %w", container.Image, err)
		}

		// Use image entrypoint
		args = append(args, imageConfig.Entrypoint...)

		// If container has no args, use image CMD as well
		if len(container.Args) == 0 {
			args = append(args, imageConfig.Cmd...)
		}
	} else {
		// Use container's explicit command
		args = append(args, container.Command...)
	}

	// Append any container args
	args = append(args, container.Args...)

	// Set command to unkey-env
	patches = append(patches, map[string]interface{}{
		"op":    "add",
		"path":  fmt.Sprintf("%s/command", basePath),
		"value": []string{unkeyEnvBinary},
	})

	// Set args to the original command/entrypoint
	if len(args) > 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  fmt.Sprintf("%s/args", basePath),
			"value": args,
		})
	}

	return patches, nil
}

// buildEnvVars creates environment variables for unkey-env from pod config.
func (m *Mutator) buildEnvVars(podCfg *podConfig) []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name:  "UNKEY_PROVIDER_ENDPOINT",
			Value: podCfg.ProviderEndpoint,
		},
		{
			Name:  "UNKEY_DEPLOYMENT_ID",
			Value: podCfg.DeploymentID,
		},
		{
			Name:  "UNKEY_TOKEN_PATH",
			Value: ServiceAccountTokenPath,
		},
	}
}
