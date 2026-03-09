package deployment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const registryPullSecretName = "depot-pull-secret"

// RegistryConfig holds container registry credentials for creating imagePullSecrets.
type RegistryConfig struct {
	URL      string
	Username string
	Password string

	// dockerConfigJSON is the pre-built dockerconfigjson blob, computed once at construction.
	dockerConfigJSON []byte
}

// NewRegistryConfig creates a RegistryConfig and pre-builds the dockerconfigjson blob.
func NewRegistryConfig(url, username, password string) *RegistryConfig {
	return &RegistryConfig{
		URL:              url,
		Username:         username,
		Password:         password,
		dockerConfigJSON: buildDockerConfigJSON(url, username, password),
	}
}

// ensureRegistryPullSecret creates or updates the dockerconfigjson pull secret
// in the given namespace. Uses server-side apply for idempotency.
func (c *Controller) ensureRegistryPullSecret(ctx context.Context, namespace string) error {
	if c.registry == nil {
		return nil
	}

	dockerConfigJSON := c.registry.dockerConfigJSON

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      registryPullSecretName,
			Namespace: namespace,
			Labels:    labels.New().ManagedByKrane(),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: dockerConfigJSON,
		},
	}

	patch, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal pull secret: %w", err)
	}

	_, err = c.clientSet.CoreV1().Secrets(namespace).Patch(
		ctx, registryPullSecretName, types.ApplyPatchType, patch,
		metav1.PatchOptions{FieldManager: fieldManagerKrane},
	)
	if err != nil {
		return fmt.Errorf("failed to apply pull secret: %w", err)
	}

	return nil
}

func buildDockerConfigJSON(registryURL, username, password string) []byte {
	config := map[string]any{
		"auths": map[string]any{
			registryURL: map[string]string{
				"username": username,
				"password": password,
			},
		},
	}
	b, _ := json.Marshal(config)
	return b
}
