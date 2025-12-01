// Package backend provides idempotent container orchestration interfaces for Krane.
//
// This package defines the Backend interface that abstracts container deployment
// and management operations across different orchestration platforms. All operations
// are designed to be fully idempotent, ensuring safe retries and recovery from
// partial failures.
//
// # Idempotency Guarantee
//
// Every operation in this package is idempotent by design. This means:
//   - Apply operations create resources if they don't exist, or reuse existing ones
//   - Delete operations succeed even if resources are already deleted
//   - Multiple identical calls produce the same result as a single call
//   - Operations can be safely retried without creating duplicates or errors
//
// This idempotent design is critical for reliability in distributed systems where
// network failures, timeouts, and partial failures are common.
//
// # Supported Backends
//
// Currently implemented backends:
//   - Docker: Single-node deployments using Docker Engine API
//   - Kubernetes: Multi-node deployments using Kubernetes API
//
// # Key Types
//
// The main interface is [Backend], which provides methods for:
//   - [Backend.ApplyDeployment]: Create or update deployments (idempotent)
//   - [Backend.ApplyGateway]: Create or update gateways (idempotent)
//   - [Backend.DeleteDeployment]: Remove deployments (idempotent)
//   - [Backend.DeleteGateway]: Remove gateways (idempotent)
//   - [Backend.GetDeployment]: Query deployment status
//   - [Backend.GetGateway]: Query gateway status
//
// Configuration types include:
//   - [ApplyDeploymentRequest]: Parameters for deployment creation/update
//   - [ApplyGatewayRequest]: Parameters for gateway creation/update
//   - [DeleteDeploymentRequest]: Parameters for deployment deletion
//   - [DeleteGatewayRequest]: Parameters for gateway deletion
//
// # Architecture Decision
//
// We chose to use Apply semantics (similar to Kubernetes) instead of separate
// Create/Update methods because:
//  1. It naturally provides idempotency
//  2. It simplifies client code by removing the need to check existence
//  3. It handles race conditions gracefully
//  4. It aligns with infrastructure-as-code practices
//
// # Usage Example
//
//	// Create a backend (Docker or Kubernetes)
//	backend := docker.New(config)
//
//	// Apply a deployment (idempotent - safe to retry)
//	err := backend.ApplyDeployment(ctx, ApplyDeploymentRequest{
//	    DeploymentID: "web-app-123",
//	    Image:        "myapp:latest",
//	    Replicas:     3,
//	})
//	if err != nil {
//	    // Safe to retry - won't create duplicates
//	    return fmt.Errorf("failed to apply deployment: %w", err)
//	}
//
//	// Get deployment status
//	resp, err := backend.GetDeployment(ctx, GetDeploymentRequest{
//	    DeploymentID: "web-app-123",
//	})
//
//	// Delete deployment (idempotent - succeeds even if already deleted)
//	err = backend.DeleteDeployment(ctx, DeleteDeploymentRequest{
//	    DeploymentID: "web-app-123",
//	})
//
// # Error Handling
//
// Operations distinguish between:
//   - Transient errors: Network issues, timeouts - safe to retry
//   - Permanent errors: Invalid configuration, insufficient resources
//   - Success conditions: Including when delete targets don't exist
//
// The idempotent design means most errors can be handled with simple retry logic.
package backend
