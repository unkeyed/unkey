package secrets

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	kranev1 "github.com/unkeyed/unkey/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/gen/proto/krane/v1/kranev1connect"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/vault"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/unkeyed/unkey/svc/krane/secrets/token"
)

// Config holds configuration for the secrets service.
//
// This configuration provides the vault service for decryption operations
// and token validator for request authentication.
type Config struct {
	// Logger for secrets operations and security events.
	// Should include correlation information for audit trails.
	Logger logging.Logger

	// Vault provides secure decryption services for encrypted secrets.
	// Must be initialized with appropriate master keys and storage backend.
	Vault *vault.Service

	// TokenValidator validates Kubernetes service account tokens
	// to ensure requests originate from authorized deployments.
	TokenValidator token.Validator
}

type Service struct {
	kranev1connect.UnimplementedSecretsServiceHandler
	logger         logging.Logger
	vault          *vault.Service
	tokenValidator token.Validator
}

// New creates a secrets service with the provided configuration.
//
// This function initializes a Service instance that implements the SecretsService
// gRPC interface. The service combines vault integration for secrets decryption
// with token validation for secure request authentication.
//
// Returns a configured Service instance ready for gRPC handler registration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedSecretsServiceHandler: kranev1connect.UnimplementedSecretsServiceHandler{},
		logger:                             cfg.Logger,
		vault:                              cfg.Vault,
		tokenValidator:                     cfg.TokenValidator,
	}
}

// DecryptSecretsBlob decrypts and returns environment variables for specified deployment.
//
// This method handles secure secrets decryption by validating the requesting service
// account token belongs to the expected deployment, then decrypting both the main
// secrets blob and individual encrypted environment variables.
//
// The method performs these security and decryption steps:
// 1. Validates that the request token belongs to a pod with matching deployment ID annotation
// 2. Decrypts the main secrets blob containing environment variable configurations
// 3. Individually decrypts each encrypted environment variable value
// 4. Returns clear text environment variables to authorized deployment
//
// Returns an authentication error if token validation fails, or internal errors
// if decryption operations fail. Empty encrypted blob returns empty environment map.
//
// Security: This method only returns secrets to deployments that can prove they
// belong to the expected deployment through Kubernetes service account token validation.
func (s *Service) DecryptSecretsBlob(
	ctx context.Context,
	req *connect.Request[kranev1.DecryptSecretsBlobRequest],
) (*connect.Response[kranev1.DecryptSecretsBlobResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	environmentID := req.Msg.GetEnvironmentId()
	requestToken := req.Msg.GetToken()
	Encrypted := req.Msg.GetEncryptedBlob()

	s.logger.Info("DecryptSecretsBlob request",
		"deployment_id", deploymentID,
		"environment_id", environmentID,
	)

	_, err := s.tokenValidator.Validate(ctx, requestToken, deploymentID, environmentID)
	if err != nil {
		s.logger.Warn("token validation failed",
			"deployment_id", deploymentID,
			"environment_id", environmentID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid or expired token: %w", err))
	}

	if len(Encrypted) == 0 {
		return connect.NewResponse(&kranev1.DecryptSecretsBlobResponse{
			EnvVars: map[string]string{},
		}), nil
	}

	decryptedBlobResp, err := s.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   environmentID,
		Encrypted: string(Encrypted),
	})
	if err != nil {
		s.logger.Error("failed to decrypt secrets blob",
			"deployment_id", deploymentID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decrypt secrets blob"))
	}

	var secretsConfig ctrlv1.SecretsConfig
	if err = protojson.Unmarshal([]byte(decryptedBlobResp.GetPlaintext()), &secretsConfig); err != nil {
		s.logger.Error("failed to unmarshal secrets config",
			"deployment_id", deploymentID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to parse secrets config"))
	}

	envVars := make(map[string]string, len(secretsConfig.GetSecrets()))
	for key, encryptedValue := range secretsConfig.GetSecrets() {
		decrypted, decryptErr := s.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   environmentID,
			Encrypted: encryptedValue,
		})
		if decryptErr != nil {
			s.logger.Error("failed to decrypt env var",
				"deployment_id", deploymentID,
				"key", key,
				"error", decryptErr,
			)
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to decrypt env var %s", key))
		}
		envVars[key] = decrypted.GetPlaintext()
	}

	s.logger.Info("decrypted secrets blob",
		"deployment_id", deploymentID,
		"num_secrets", len(envVars),
	)

	return connect.NewResponse(&kranev1.DecryptSecretsBlobResponse{
		EnvVars: envVars,
	}), nil
}
