package mutator

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/unkeyed/unkey/go/apps/secrets-webhook/internal/services/registry"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Mutator struct {
	cfg      *Config
	logger   logging.Logger
	registry *registry.Registry
}

func New(cfg *Config, logger logging.Logger, reg *registry.Registry) *Mutator {
	return &Mutator{
		cfg:      cfg,
		logger:   logger,
		registry: reg,
	}
}

type Result struct {
	Mutated bool
	Patch   []byte
	Message string
}

func (m *Mutator) ShouldMutate(pod *corev1.Pod) bool {
	annotations := pod.GetAnnotations()
	if annotations == nil {
		return false
	}
	return annotations[m.cfg.GetAnnotation(AnnotationDeploymentID)] != ""
}

func (m *Mutator) Mutate(ctx context.Context, pod *corev1.Pod, namespace string) (*Result, error) {
	if !m.ShouldMutate(pod) {
		return &Result{Mutated: false, Message: "pod not annotated for injection"}, nil
	}

	annotations := pod.GetAnnotations()

	podCfg, err := m.loadPodConfig(annotations)
	if err != nil {
		return nil, err
	}

	m.logger.Info("loaded pod config from annotations",
		"deployment_id", podCfg.DeploymentID,
		"provider_endpoint", podCfg.ProviderEndpoint,
	)

	var patches []map[string]interface{}

	initContainer := m.buildInitContainer()
	if len(pod.Spec.InitContainers) == 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/initContainers",
			"value": []corev1.Container{initContainer},
		})
	} else {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/initContainers/-",
			"value": initContainer,
		})
	}

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

	for i, container := range pod.Spec.Containers {
		containerPatches, patchErr := m.buildContainerPatches(ctx, i, &container, &pod.Spec, namespace, podCfg)
		if patchErr != nil {
			return nil, fmt.Errorf("failed to build patches for container %s: %w", container.Name, patchErr)
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
