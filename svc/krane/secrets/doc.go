// Package secrets provides secure secrets management and decryption services for krane deployments.
//
// This package implements the SecretsService gRPC interface that handles encrypted
// environment variable decryption for deployments. It integrates with the vault service
// for secure secrets storage and uses Kubernetes service account token validation
// for authentication.
//
// # Architecture
//
// The package provides a Service type that combines three key components:
//   - Vault integration for secure secrets decryption
//   - Token validation for request authentication
//   - gRPC service for exposing secrets API
//
// # Security Model
//
// Requests are authenticated using Kubernetes service account tokens. The service validates
// that the requesting pod belongs to the expected deployment by checking pod annotations
// and service account membership. This ensures that only authorized deployments can
// access their own secrets.
//
// # Decryption Flow
//
// 1. Client presents Kubernetes service account token and deployment ID
// 2. Service validates token belongs to pod with matching deployment annotation
// 3. Service decrypts master secrets blob using vault service
// 4. Individual environment variables are decrypted separately
// 5. Clear text secrets are returned to authorized deployment
//
// # Key Types
//
// [Service]: Main implementation of the SecretsService gRPC interface
// [Config]: Configuration for secrets service initialization
//
// # Usage
//
// Basic service setup with vault and token validation:
//
//	cfg := secrets.Config{
//		Vault:          vaultService,
//		TokenValidator: tokenValidator,
//	}
//	service := secrets.New(cfg)
//
//	// Register with gRPC server
//	mux.Handle(kranev1connect.NewSecretsServiceHandler(service))
package secrets
