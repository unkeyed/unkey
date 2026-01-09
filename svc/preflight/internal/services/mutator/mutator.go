package mutator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/registry"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/registry/credentials"
)

const (
	// tokenTTL is the lifetime of Depot pull tokens. We use a slightly shorter
	// duration for the expiry check to account for clock skew and processing time.
	tokenTTL            = 55 * time.Minute
	expiresAtAnnotation = "preflight.unkey.com/expires-at"
)

type Mutator struct {
	logger                  logging.Logger
	registry                *registry.Registry
	clientset               kubernetes.Interface
	credentials             *credentials.Manager
	injectImage             string
	injectImagePullPolicy   string
	defaultProviderEndpoint string
}

func New(cfg Config) *Mutator {
	return &Mutator{
		logger:                  cfg.Logger,
		registry:                cfg.Registry,
		clientset:               cfg.Clientset,
		credentials:             cfg.Credentials,
		injectImage:             cfg.InjectImage,
		injectImagePullPolicy:   cfg.InjectImagePullPolicy,
		defaultProviderEndpoint: cfg.DefaultProviderEndpoint,
	}
}

type Result struct {
	Mutated bool
	Patch   []byte
	Message string
}

func (m *Mutator) ShouldMutate(pod *corev1.Pod) bool {
	labels := pod.GetLabels()
	if labels == nil {
		return false
	}

	return labels[LabelDeploymentID] != ""
}

func (m *Mutator) Mutate(ctx context.Context, pod *corev1.Pod, namespace string) (*Result, error) {
	if !m.ShouldMutate(pod) {
		return &Result{Mutated: false, Patch: nil, Message: "pod not labeled for injection"}, nil
	}

	labels := pod.GetLabels()

	podCfg, err := m.loadPodConfig(labels)
	if err != nil {
		return nil, err
	}

	var patches []map[string]interface{}
	buildID := labels[LabelBuildID]

	// Check if any container uses a private registry image and inject imagePullSecret if needed
	privateImages := m.collectPrivateRegistryImages(pod)
	if len(privateImages) > 0 {
		secretPatches := m.ensurePullSecrets(ctx, pod, namespace, privateImages, buildID)
		patches = append(patches, secretPatches...)
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
		Message: fmt.Sprintf("injected secrets for deployment %s", podCfg.DeploymentID),
	}, nil
}

// collectPrivateRegistryImages returns all unique images that need credentials.
func (m *Mutator) collectPrivateRegistryImages(pod *corev1.Pod) []string {
	seen := make(map[string]bool)
	var images []string

	for _, container := range pod.Spec.Containers {
		if m.credentials.Matches(container.Image) && !seen[container.Image] {
			seen[container.Image] = true
			images = append(images, container.Image)
		}
	}

	for _, container := range pod.Spec.InitContainers {
		if m.credentials.Matches(container.Image) && !seen[container.Image] {
			seen[container.Image] = true
			images = append(images, container.Image)
		}
	}

	return images
}

// ensurePullSecrets creates or reuses pull secrets for each private image and returns
// patches to add them to the pod's imagePullSecrets.
func (m *Mutator) ensurePullSecrets(ctx context.Context, pod *corev1.Pod, namespace string, images []string, buildID string) []map[string]interface{} {
	var secretNames []string

	for _, image := range images {
		secretName, err := m.ensurePullSecretForImage(ctx, namespace, image, buildID)
		if err != nil {
			m.logger.Error("failed to ensure pull secret for image",
				"image", image,
				"error", err,
			)
			continue
		}
		secretNames = append(secretNames, secretName)
	}

	if len(secretNames) == 0 {
		return nil
	}

	// Build patches to add imagePullSecrets
	var patches []map[string]interface{}
	existingSecrets := make(map[string]bool)
	for _, secret := range pod.Spec.ImagePullSecrets {
		existingSecrets[secret.Name] = true
	}

	var newSecrets []corev1.LocalObjectReference
	for _, name := range secretNames {
		if !existingSecrets[name] {
			newSecrets = append(newSecrets, corev1.LocalObjectReference{Name: name})
		}
	}

	if len(newSecrets) == 0 {
		return nil
	}

	if len(pod.Spec.ImagePullSecrets) == 0 {
		patches = append(patches, map[string]interface{}{
			"op":    "add",
			"path":  "/spec/imagePullSecrets",
			"value": newSecrets,
		})
	} else {
		for _, secret := range newSecrets {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  "/spec/imagePullSecrets/-",
				"value": secret,
			})
		}
	}

	return patches
}

// ensurePullSecretForImage creates or reuses a pull secret for a specific image.
// If a valid (non-expired) secret exists, it's reused without calling the registry API.
func (m *Mutator) ensurePullSecretForImage(ctx context.Context, namespace, image, buildID string) (string, error) {
	secretName := m.generateSecretName(image)

	// Check if a valid secret already exists
	existing, err := m.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		// Secret exists - check if it's still valid
		if m.isSecretValid(existing) {
			m.logger.Debug("reusing existing pull secret",
				"secret", secretName,
				"image", image,
			)
			return secretName, nil
		}
		// Secret expired - delete and recreate
		m.logger.Info("pull secret expired, refreshing",
			"secret", secretName,
			"image", image,
		)
		if delErr := m.clientset.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{}); delErr != nil {
			m.logger.Warn("failed to delete expired secret", "error", delErr)
		}
	} else if !apierrors.IsNotFound(err) {
		return "", fmt.Errorf("failed to check for existing secret: %w", err)
	}

	// Fetch fresh credentials
	dockerConfig, err := m.credentials.GetCredentials(ctx, image, buildID)
	if err != nil {
		return "", fmt.Errorf("failed to get registry credentials: %w", err)
	}
	if dockerConfig == nil {
		return "", fmt.Errorf("no credentials found for image: %s", image)
	}

	dockerConfigJSON, err := dockerConfig.ToJSON()
	if err != nil {
		return "", fmt.Errorf("failed to marshal docker config: %w", err)
	}

	expiresAt := time.Now().Add(tokenTTL).Format(time.RFC3339)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "preflight",
				"app.kubernetes.io/component":  "registry-credentials",
			},
			Annotations: map[string]string{
				"preflight.unkey.com/image": image,
				expiresAtAnnotation:         expiresAt,
			},
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	_, err = m.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Race condition - another pod created it, that's fine
			return secretName, nil
		}
		return "", fmt.Errorf("failed to create secret: %w", err)
	}

	m.logger.Info("created pull secret",
		"namespace", namespace,
		"secret", secretName,
		"image", image,
		"expires_at", expiresAt,
	)

	return secretName, nil
}

// isSecretValid checks if the secret's token hasn't expired.
func (m *Mutator) isSecretValid(secret *corev1.Secret) bool {
	if secret.Annotations == nil {
		return false
	}

	expiresAtStr, ok := secret.Annotations[expiresAtAnnotation]
	if !ok {
		return false
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		m.logger.Warn("invalid expires-at annotation", "value", expiresAtStr, "error", err)
		return false
	}

	return time.Now().Before(expiresAt)
}

// generateSecretName creates a deterministic secret name for an image.
func (m *Mutator) generateSecretName(image string) string {
	hash := sha256.Sum256([]byte(image))
	shortHash := hex.EncodeToString(hash[:])[:8]
	return fmt.Sprintf("preflight-pull-%s", shortHash)
}
