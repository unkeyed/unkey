package mutator

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	cleanupInterval = 1 * time.Hour
	managedByLabel  = "app.kubernetes.io/managed-by"
	managedByValue  = "preflight"
)

// StartCleanupLoop starts a background goroutine that periodically deletes expired
// pull secrets. It runs until the context is cancelled.
func (m *Mutator) StartCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	// Run once immediately on startup
	m.cleanupExpiredSecrets(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("stopping secret cleanup loop")
			return
		case <-ticker.C:
			m.cleanupExpiredSecrets(ctx)
		}
	}
}

// cleanupExpiredSecrets deletes all preflight-managed secrets that have expired.
func (m *Mutator) cleanupExpiredSecrets(ctx context.Context) {
	m.logger.Debug("running expired secret cleanup")

	// List all secrets managed by preflight across all namespaces
	secrets, err := m.clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{
		LabelSelector: managedByLabel + "=" + managedByValue,
	})
	if err != nil {
		m.logger.Error("failed to list secrets for cleanup", "error", err)
		return
	}

	var deleted int
	for _, secret := range secrets.Items {
		if m.isSecretValid(&secret) {
			continue
		}

		// Secret is expired or has no valid expiry annotation
		err := m.clientset.CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil {
			m.logger.Warn("failed to delete expired secret",
				"namespace", secret.Namespace,
				"secret", secret.Name,
				"error", err,
			)
			continue
		}

		deleted++
		m.logger.Info("deleted expired pull secret", "namespace", secret.Namespace, "secret", secret.Name)
	}

	m.logger.Info("cleanup complete", "deleted", deleted)

}
