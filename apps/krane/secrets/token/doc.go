// Package token provides validation interfaces and implementations for krane authentication.
//
// This package defines the Validator interface for authenticating requests using
// Kubernetes service account tokens. It includes a Kubernetes-based implementation
// that validates tokens belong to pods with specific deployment annotations.
//
// # Authentication Model
//
// The package implements Kubernetes-native authentication by validating service account
// tokens through the Kubernetes TokenReview API. This ensures that requests
// originate from pods within the cluster and belong to expected deployments.
//
// # Validation Flow
//
// 1. Token is submitted to Kubernetes TokenReview API
// 2. TokenReview response contains user information and pod metadata
// 3. Pod is retrieved to verify deployment annotations
// 4. Request is validated if pod belongs to expected deployment
//
// # Key Types
//
// [Validator]: Interface for token validation implementations
// [ValidationResult]: Result of successful token validation
// [K8sValidator]: Kubernetes-based validator implementation
//
// # Usage
//
// Basic token validation:
//
//	validator := token.NewK8sValidator(token.K8sValidatorConfig{
//		Clientset: kubernetesClientset,
//	})
//
//	result, err := validator.Validate(ctx, serviceAccountToken, "deployment-123")
//	if err != nil {
//		// Handle authentication failure
//	}
//	// Use validated result for authorization
package token
