package secrets

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	kranev1 "github.com/unkeyed/unkey/go/gen/proto/krane/v1"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/unkeyed/unkey/go/apps/krane/secrets/token"
)

type Config struct {
	Logger         logging.Logger
	Vault          *vault.Service
	TokenValidator token.Validator
}

type Service struct {
	kranev1connect.UnimplementedSecretsServiceHandler
	logger         logging.Logger
	vault          *vault.Service
	tokenValidator token.Validator
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedSecretsServiceHandler: kranev1connect.UnimplementedSecretsServiceHandler{},
		logger:                             cfg.Logger,
		vault:                              cfg.Vault,
		tokenValidator:                     cfg.TokenValidator,
	}
}

func (s *Service) DecryptSecretsBlob(
	ctx context.Context,
	req *connect.Request[kranev1.DecryptSecretsBlobRequest],
) (*connect.Response[kranev1.DecryptSecretsBlobResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	environmentID := req.Msg.GetEnvironmentId()
	requestToken := req.Msg.GetToken()
	Encrypted := req.Msg.Encrypted()

	s.logger.Info("DecryptSecretsBlob request",
		"deployment_id", deploymentID,
		"environment_id", environmentID,
	)

	_, err := s.tokenValidator.Validate(ctx, requestToken, deploymentID)
	if err != nil {
		s.logger.Warn("token validation failed",
			"deployment_id", deploymentID,
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
