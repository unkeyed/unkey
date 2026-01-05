package cleanup

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/pkg/otel/logging"
)

const (
	cleanupInterval     = 1 * time.Hour
	labelSelector       = "app.kubernetes.io/managed-by=preflight"
	expiresAtAnnotation = "preflight.unkey.com/expires-at"
)

type Config struct {
	Logger    logging.Logger
	Clientset kubernetes.Interface
}

type Service struct {
	logger    logging.Logger
	clientset kubernetes.Interface
}

func New(cfg *Config) *Service {
	return &Service{
		logger:    cfg.Logger,
		clientset: cfg.Clientset,
	}
}

// Start begins the background cleanup loop. It runs until the context is cancelled.
func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		s.cleanupExpiredSecrets(ctx)

		select {
		case <-ctx.Done():
			s.logger.Info("stopping secret cleanup loop")
			return
		case <-ticker.C:
		}
	}
}

// cleanupExpiredSecrets deletes all preflight-managed secrets that have expired.
func (s *Service) cleanupExpiredSecrets(ctx context.Context) {
	s.logger.Debug("running expired secret cleanup")

	// List all secrets managed by preflight across all namespaces
	secrets, err := s.clientset.CoreV1().Secrets("").List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		s.logger.Error("failed to list secrets for cleanup", "error", err)
		return
	}

	var deleted int
	for _, secret := range secrets.Items {
		if s.isSecretValid(&secret) {
			continue
		}

		// Secret is expired or has no valid expiry annotation
		err := s.clientset.CoreV1().Secrets(secret.Namespace).Delete(ctx, secret.Name, metav1.DeleteOptions{})
		if err != nil {
			s.logger.Warn("failed to delete expired secret",
				"namespace", secret.Namespace,
				"secret", secret.Name,
				"error", err,
			)
			continue
		}

		deleted++
		s.logger.Info("deleted expired pull secret", "namespace", secret.Namespace, "secret", secret.Name)
	}

	s.logger.Info("cleanup complete", "deleted", deleted)
}

// isSecretValid checks if the secret's token hasn't expired.
func (s *Service) isSecretValid(secret *corev1.Secret) bool {
	if secret.Annotations == nil {
		return false
	}

	expiresAtStr, ok := secret.Annotations[expiresAtAnnotation]
	if !ok {
		return false
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		s.logger.Warn("invalid expires-at annotation", "value", expiresAtStr, "error", err)
		return false
	}

	return time.Now().Before(expiresAt)
}
