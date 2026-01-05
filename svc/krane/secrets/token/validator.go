package token

import "context"

// ValidationResult contains the result of successful token validation.
//
// This struct provides information about the validated token, including
// the deployment ID that the requesting pod belongs to.
type ValidationResult struct {
	// DeploymentID is the identifier of the deployment that the validated
	// token belongs to. This can be used for authorization decisions.
	DeploymentID string
}

// Validator defines interface for token validation implementations.
//
// Implementations should validate that the provided token belongs to a pod
// or service account that is authorized to access resources for the specified
// deployment ID. Different validation strategies can be implemented (Kubernetes,
// external auth providers, mock for testing, etc.).
type Validator interface {
	// Validate validates that token belongs to expected deployment.
	//
	// The token string represents authentication credentials (typically a
	// Kubernetes service account token). The deploymentID represents the
	// expected deployment that should be making the request.
	//
	// Returns ValidationResult on successful validation or error if token is
	// invalid, expired, or belongs to different deployment.
	Validate(ctx context.Context, token string, deploymentID string) (*ValidationResult, error)
}
