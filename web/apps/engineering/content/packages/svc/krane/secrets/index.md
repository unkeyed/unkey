---
title: secrets
description: "provides secure secrets management and decryption services for krane deployments"
---

Package secrets provides secure secrets management and decryption services for krane deployments.

This package implements the SecretsService gRPC interface that handles encrypted environment variable decryption for deployments. It integrates with the vault service for secure secrets storage and uses Kubernetes service account token validation for authentication.

### Architecture

The package provides a Service type that combines three key components:

  - Vault integration for secure secrets decryption
  - Token validation for request authentication
  - gRPC service for exposing secrets API

### Security Model

Requests are authenticated using Kubernetes service account tokens. The service validates that the requesting pod belongs to the expected deployment by checking pod annotations and service account membership. This ensures that only authorized deployments can access their own secrets.

### Decryption Flow

1\. Client presents Kubernetes service account token and deployment ID 2. Service validates token belongs to pod with matching deployment annotation 3. Service decrypts master secrets blob using vault service 4. Individual environment variables are decrypted separately 5. Clear text secrets are returned to authorized deployment

### Key Types

\[Service]: Main implementation of the SecretsService gRPC interface \[Config]: Configuration for secrets service initialization

### Usage

Basic service setup with vault and token validation:

	cfg := secrets.Config{
		Vault:          vaultService,
		TokenValidator: tokenValidator,
	}
	service := secrets.New(cfg)

	// Register with gRPC server
	mux.Handle(kranev1connect.NewSecretsServiceHandler(service))

## Types

### type Config

```go
type Config struct {
	// Vault provides secure decryption services for encrypted secrets via the vault API.
	Vault vaultv1connect.VaultServiceClient

	// TokenValidator validates Kubernetes service account tokens
	// to ensure requests originate from authorized deployments.
	TokenValidator token.Validator
}
```

Config holds configuration for the secrets service.

This configuration provides the vault client for decryption operations and token validator for request authentication.

### type Service

```go
type Service struct {
	kranev1connect.UnimplementedSecretsServiceHandler
	vault          vaultv1connect.VaultServiceClient
	tokenValidator token.Validator
}
```

#### func New

```go
func New(cfg Config) *Service
```

New creates a secrets service with the provided configuration.

This function initializes a Service instance that implements the SecretsService gRPC interface. The service combines vault integration for secrets decryption with token validation for secure request authentication.

Returns a configured Service instance ready for gRPC handler registration.

#### func (Service) DecryptSecretsBlob

```go
func (s *Service) DecryptSecretsBlob(
	ctx context.Context,
	req *connect.Request[kranev1.DecryptSecretsBlobRequest],
) (*connect.Response[kranev1.DecryptSecretsBlobResponse], error)
```

DecryptSecretsBlob decrypts and returns environment variables for specified deployment.

This method handles secure secrets decryption by validating the requesting service account token belongs to the expected deployment, then decrypting both the main secrets blob and individual encrypted environment variables.

The method performs these security and decryption steps: 1. Validates that the request token belongs to a pod with matching deployment ID annotation 2. Decrypts the main secrets blob containing environment variable configurations 3. Individually decrypts each encrypted environment variable value 4. Returns clear text environment variables to authorized deployment

Returns an authentication error if token validation fails, or internal errors if decryption operations fail. Empty encrypted blob returns empty environment map.

Security: This method only returns secrets to deployments that can prove they belong to the expected deployment through Kubernetes service account token validation.

