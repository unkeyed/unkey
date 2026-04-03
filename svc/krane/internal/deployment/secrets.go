package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"google.golang.org/protobuf/encoding/protojson"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// decryptSecrets decrypts the encrypted environment variables blob via Vault.
// Returns nil map if there are no secrets. Returns an error if secrets are
// present but vault is not configured.
func (c *Controller) decryptSecrets(ctx context.Context, encrypted []byte, environmentID string) (map[string]string, error) {
	if len(encrypted) == 0 {
		return nil, nil
	}
	if c.vault == nil {
		return nil, fmt.Errorf("deployment has encrypted secrets but vault is not configured (environment %s)", environmentID)
	}

	var secretsConfig ctrlv1.SecretsConfig
	if err := protojson.Unmarshal(encrypted, &secretsConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secrets config: %w", err)
	}

	bulkRes, err := c.vault.DecryptBulk(ctx, &vaultv1.DecryptBulkRequest{
		Keyring: environmentID,
		Items:   secretsConfig.GetSecrets(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to bulk decrypt env vars: %w", err)
	}

	logger.Info("decrypted secrets at deploy time",
		"environment_id", environmentID,
		"num_secrets", len(bulkRes.GetItems()),
	)

	return bulkRes.GetItems(), nil
}

var k8sSecretKeyRegex = regexp.MustCompile(`^[-._a-zA-Z0-9]+$`)

// deploymentResourcePrefix returns the base name used for all K8s resources
// (Secret, ServiceAccount, Role, RoleBinding) owned by a deployment.
// Converts the deployment ID to a valid RFC 1123 subdomain name.
func deploymentResourcePrefix(deploymentID string) string {
	return "deploy-" + strings.ToLower(strings.ReplaceAll(deploymentID, "_", "-"))
}

// ensureDeploymentSecret creates or updates a K8s Secret containing the plaintext
// environment variables for the deployment. Uses server-side apply for idempotency.
// The ownerRef ties the secret's lifecycle to the ReplicaSet for automatic GC.
func (c *Controller) ensureDeploymentSecret(ctx context.Context, namespace, deploymentID string, envVars map[string]string) error {
	secretName := deploymentResourcePrefix(deploymentID)

	// Use Data (not StringData) so SSA tracks ownership of data.* keys directly.
	// StringData is converted to data.* server-side, but SSA tracks stringData.*
	// ownership — removed keys won't be cleaned up from data on re-apply.
	data := make(map[string][]byte, len(envVars))
	for k, v := range envVars {
		if !k8sSecretKeyRegex.MatchString(k) {
			return fmt.Errorf("environment variable key %q contains invalid characters, only letters, numbers, hyphens, underscores, and dots are allowed", k)
		}
		data[k] = []byte(v)
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels:    labels.New().DeploymentID(deploymentID).ManagedByKrane(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}

	patch, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	_, err = c.clientSet.CoreV1().Secrets(namespace).Patch(
		ctx, secretName, types.ApplyPatchType, patch,
		metav1.PatchOptions{FieldManager: fieldManagerKrane},
	)
	if err != nil {
		return fmt.Errorf("failed to apply secret: %w", err)
	}

	return nil
}
