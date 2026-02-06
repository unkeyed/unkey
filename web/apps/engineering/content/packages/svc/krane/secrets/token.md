---
title: token
description: "provides validation interfaces and implementations for krane authentication"
---

Package token provides validation interfaces and implementations for krane authentication.

This package defines the Validator interface for authenticating requests using Kubernetes service account tokens. It includes a Kubernetes-based implementation that validates tokens belong to pods with specific deployment annotations.

### Authentication Model

The package implements Kubernetes-native authentication by validating service account tokens through the Kubernetes TokenReview API. This ensures that requests originate from pods within the cluster and belong to expected deployments.

### Validation Flow

1\. Token is submitted to Kubernetes TokenReview API 2. TokenReview response contains user information and pod metadata 3. Pod is retrieved to verify deployment annotations 4. Request is validated if pod belongs to expected deployment

### Key Types

\[Validator]: Interface for token validation implementations \[ValidationResult]: Result of successful token validation \[K8sValidator]: Kubernetes-based validator implementation

### Usage

Basic token validation:

	validator := token.NewK8sValidator(token.K8sValidatorConfig{
		Clientset: kubernetesClientset,
	})

	result, err := validator.Validate(ctx, serviceAccountToken, "deployment-123")
	if err != nil {
		// Handle authentication failure
	}
	// Use validated result for authorization

## Types

### type K8sValidator

```go
type K8sValidator struct {
	clientset kubernetes.Interface
}
```

K8sValidator validates service account tokens using Kubernetes TokenReview API.

This validator authenticates requests by verifying that tokens belong to pods annotated with the expected deployment ID. It provides Kubernetes-native authentication without requiring external identity providers.

#### func NewK8sValidator

```go
func NewK8sValidator(cfg K8sValidatorConfig) *K8sValidator
```

NewK8sValidator creates a Kubernetes-based token validator.

This function initializes a K8sValidator that uses Kubernetes TokenReview API to validate service account tokens and pod information to verify deployment membership. The validator ensures tokens belong to pods with expected deployment annotations.

Returns a configured K8sValidator ready for use.

#### func (K8sValidator) Validate

```go
func (v *K8sValidator) Validate(ctx context.Context, token string, deploymentID string, environmentID string) (*ValidationResult, error)
```

Validate validates Kubernetes service account token against expected deployment and environment.

This method implements token validation using Kubernetes TokenReview API. It verifies that the token is valid, extracts pod information from token review, retrieves the pod, and validates that the pod belongs to the expected deployment and environment by checking pod labels.

Returns ValidationResult on successful validation or error if: - Token is invalid or expired - Token doesn't represent a service account - Pod information cannot be extracted - Pod doesn't have required deployment or environment labels - Pod labels don't match expected deploymentID or environmentID

### type K8sValidatorConfig

```go
type K8sValidatorConfig struct {
	// Clientset is the Kubernetes client for API calls.
	// Must have permissions for TokenReview creation and pod retrieval.
	Clientset kubernetes.Interface
}
```

K8sValidatorConfig holds configuration for Kubernetes token validator.

This configuration provides the Kubernetes clientset needed for TokenReview API calls and pod retrieval during validation.

### type ValidationResult

```go
type ValidationResult struct {
	// DeploymentID is the identifier of the deployment that the validated
	// token belongs to. This can be used for authorization decisions.
	DeploymentID string

	// EnvironmentID is the identifier of the environment that the validated
	// token belongs to. This can be used for authorization decisions.
	EnvironmentID string
}
```

ValidationResult contains the result of successful token validation.

This struct provides information about the validated token, including the deployment ID and environment ID that the requesting pod belongs to.

### type Validator

```go
type Validator interface {
	// Validate validates that token belongs to expected deployment and environment.
	//
	// The token string represents authentication credentials (typically a
	// Kubernetes service account token). The deploymentID represents the
	// expected deployment that should be making the request. The environmentID
	// represents the expected environment for the request.
	//
	// Returns ValidationResult on successful validation or error if token is
	// invalid, expired, or belongs to different deployment or environment.
	Validate(ctx context.Context, token string, deploymentID string, environmentID string) (*ValidationResult, error)
}
```

Validator defines interface for token validation implementations.

Implementations should validate that the provided token belongs to a pod or service account that is authorized to access resources for the specified deployment ID and environment ID. Different validation strategies can be implemented (Kubernetes, external auth providers, mock for testing, etc.).

