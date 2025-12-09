package mutator

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Mutator struct {
	cfg *Config
}

func New(cfg *Config) *Mutator {
	return &Mutator{cfg: cfg}
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
		return &Result{Mutated: false, Patch: nil, Message: "pod not annotated for injection"}, nil
	}

	annotations := pod.GetAnnotations()

	podCfg, err := m.loadPodConfig(annotations)
	if err != nil {
		return nil, err
	}

	m.cfg.Logger.Info("loaded pod config from annotations",
		"deployment_id", podCfg.DeploymentID,
		"provider_endpoint", podCfg.ProviderEndpoint,
	)

	var patches []map[string]interface{}
	buildID := annotations[m.cfg.GetAnnotation("build-id")]

	// Check if any container uses a private registry image and inject imagePullSecret if needed
	privateImages := m.collectPrivateRegistryImages(pod)
	if len(privateImages) > 0 {
		secretPatches, secretErr := m.ensurePullSecrets(ctx, pod, namespace, privateImages, buildID)
		if secretErr != nil {
			m.cfg.Logger.Error("failed to ensure pull secrets", "error", secretErr)
			// Continue without registry auth - don't fail the entire mutation
		} else {
			patches = append(patches, secretPatches...)
		}
	}

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
		containerPatches, patchErr := m.buildContainerPatches(ctx, i, &container, &pod.Spec, namespace, podCfg, buildID)
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

// collectPrivateRegistryImages returns all unique images that need credentials.
func (m *Mutator) collectPrivateRegistryImages(pod *corev1.Pod) []string {
	seen := make(map[string]bool)
	var images []string

	for _, container := range pod.Spec.Containers {
		if m.cfg.Credentials.Matches(container.Image) && !seen[container.Image] {
			seen[container.Image] = true
			images = append(images, container.Image)
		}
	}
	for _, container := range pod.Spec.InitContainers {
		if m.cfg.Credentials.Matches(container.Image) && !seen[container.Image] {
			seen[container.Image] = true
			images = append(images, container.Image)
		}
	}

	return images
}

// ensurePullSecrets creates a pull secret for the pod and returns patches to add it
// to imagePullSecrets. The secret is tied to the pod's lifecycle via OwnerReference
// so it gets garbage collected when the pod is deleted.
func (m *Mutator) ensurePullSecrets(ctx context.Context, pod *corev1.Pod, namespace string, images []string, buildID string) ([]map[string]interface{}, error) {
	secretName, err := m.createPodPullSecret(ctx, pod, namespace, images, buildID)
	if err != nil {
		return nil, err
	}

	// Check if this secret is already referenced
	for _, secret := range pod.Spec.ImagePullSecrets {
		if secret.Name == secretName {
			return nil, nil
		}
	}

	var patches []map[string]interface{}
	newSecret := corev1.LocalObjectReference{Name: secretName}

	if len(pod.Spec.ImagePullSecrets) == 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/imagePullSecrets",
			"value": []corev1.LocalObjectReference{newSecret},
		})
	} else {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/imagePullSecrets/-",
			"value": newSecret,
		})
	}

	return patches, nil
}

// createPodPullSecret creates an ephemeral pull secret for the pod with credentials
// for all private images. The secret has an OwnerReference to the pod so it gets
// garbage collected when the pod is deleted.
func (m *Mutator) createPodPullSecret(ctx context.Context, pod *corev1.Pod, namespace string, images []string, buildID string) (string, error) {
	secretName := m.generateSecretName(pod)

	// Merge credentials for all images into one docker config
	mergedConfig := m.cfg.Credentials.NewDockerConfig()
	for _, image := range images {
		dockerConfig, err := m.cfg.Credentials.GetCredentials(ctx, image, buildID)
		if err != nil {
			m.cfg.Logger.Error("failed to get credentials for image",
				"image", image,
				"error", err,
			)
			continue
		}
		if dockerConfig != nil {
			mergedConfig.Merge(dockerConfig)
		}
	}

	if len(mergedConfig.Auths) == 0 {
		return "", fmt.Errorf("no credentials found for any image")
	}

	dockerConfigJSON, err := mergedConfig.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal docker config: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "preflight",
				"app.kubernetes.io/component":  "registry-credentials",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       pod.Name,
					UID:        pod.UID,
				},
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	_, err = m.cfg.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Secret already exists (maybe from a previous attempt), that's fine
			return secretName, nil
		}
		return "", fmt.Errorf("failed to create secret: %w", err)
	}

	m.cfg.Logger.Info("created ephemeral pull secret",
		"namespace", namespace,
		"secret", secretName,
		"pod", pod.Name,
		"images", images,
	)

	return secretName, nil
}

// generateSecretName creates a unique secret name for the pod.
func (m *Mutator) generateSecretName(pod *corev1.Pod) string {
	// Use pod UID to ensure uniqueness per pod
	shortUID := string(pod.UID)
	if len(shortUID) > 8 {
		shortUID = shortUID[:8]
	}
	return fmt.Sprintf("preflight-pull-%s", shortUID)
}
