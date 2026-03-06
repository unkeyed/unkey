package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"google.golang.org/protobuf/encoding/protojson"
)

// decryptSecrets decrypts the encrypted environment variables blob via Vault.
// Returns nil map if there are no secrets or vault is not configured.
func (c *Controller) decryptSecrets(ctx context.Context, encrypted []byte, environmentID string) (map[string]string, error) {
	if len(encrypted) == 0 || c.vault == nil {
		return nil, nil
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

// deploymentSecretName returns the deterministic K8s Secret name for a deployment.
func deploymentSecretName(deploymentID string) string {
	return fmt.Sprintf("deploy-%s", sanitizeForK8s(deploymentID))
}

// sanitizeForK8s converts an ID to a valid RFC 1123 subdomain name by
// lowercasing and replacing underscores with dashes.
func sanitizeForK8s(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "_", "-"))
}

// ensureDeploymentSecret creates or updates a K8s Secret containing the plaintext
// environment variables for the deployment. Uses server-side apply for idempotency.
// The ownerRef ties the secret's lifecycle to the ReplicaSet for automatic GC.
func (c *Controller) ensureDeploymentSecret(ctx context.Context, namespace, deploymentID string, envVars map[string]string, ownerRef metav1.OwnerReference) error {
	secretName := deploymentSecretName(deploymentID)

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            secretName,
			Namespace:       namespace,
			Labels:          labels.New().DeploymentID(deploymentID).ManagedByKrane(),
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		StringData: envVars,
		Type:       corev1.SecretTypeOpaque,
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
