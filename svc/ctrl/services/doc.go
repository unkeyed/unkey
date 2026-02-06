// Package services provides Connect service implementations for the control plane.
//
// This package contains all Connect service handlers that implement
// the public API surface of the unkey control plane. Each service
// is responsible for a specific domain of functionality including deployment
// orchestration, build operations, certificate management, routing,
// and administrative operations.
//
// # Service Categories
//
// ## Administrative Services
// [ctrl]: Control plane health, liveness, and administrative operations
//
// ## Core Platform Services
// [deployment]: Complete deployment lifecycle management
// [routing]: Domain assignment and traffic routing workflows
//
// ## Infrastructure Services
// [acme]: ACME certificate challenge management and TLS provisioning
// [build]: Container image building via multiple backends (Depot, Docker)
// [openapi]: OpenAPI specification generation and schema documentation
//
// # Architecture
//
// All services follow consistent patterns:
//   - Implement Connect handlers with proper error handling
//   - Use database transactions for data consistency
//   - Integrate with Restate workflows for complex operations
//   - Include comprehensive logging and observability
//   - Validate inputs and provide clear error responses
//
// # Configuration
//
// Each service accepts a Config struct that provides:
//   - Database connection for persistent storage
//   - Logger for structured operation logging
//   - Client connections to external services (krane, vault, etc.)
//
// # Usage
//
// Services are typically created and registered with the main control
// plane server in run.go:
//
//	// Create deployment service
//	deploymentSvc := deployment.New(deployment.Config{
//		Database: database,
//		Restate: restateClient,
//		BuildService: buildService,
//	,
//	})
//
//	// Register with Connect server
//	mux.Handle(ctrlv1connect.NewDeploymentServiceHandler(deploymentSvc))
//
// # Error Handling
//
// All services use standardized error responses with appropriate
// Connect error codes:
//   - InvalidArgument: Bad request parameters
//   - Unauthenticated: Missing or invalid authentication
//   - Internal: Unexpected system failures
//   - NotFound: Requested resources don't exist
//
// # Cross-Cutting Concerns
//
// Services coordinate across these boundaries:
//   - Database consistency through proper transaction management
//   - Workflow orchestration through Restate virtual objects
//   - External service integration (build backends, vault, etc.)
//   - Resource lifecycle management with proper cleanup
package services
