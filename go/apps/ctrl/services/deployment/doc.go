// Package deployment provides complete deployment lifecycle orchestration.
//
// This package implements the core deployment functionality of the
// unkey platform, managing the entire deployment process from
// container creation through scaling, routing, and promotion.
// It coordinates with Krane agents for container orchestration
// and integrates with build services for container image creation.
//
// # Architecture
//
// The deployment service provides comprehensive workflow orchestration:
//   - Deploy new container images and configure routing
//   - Scale deployments by adjusting replica counts
//   - Promote successful deployments to production traffic
//   - Rollback failed deployments to previous working versions
//   - Manage domain assignments and sticky behavior
//   - Coordinate with sentinel configurations for edge routing
//
// # Built on Restate
//
// All deployment workflows use Restate for durable execution:
//   - Automatic retries on transient failures
//   - Exactly-once guarantees for each workflow step
//   - Durable state that survives process crashes and restarts
//   - Virtual object concurrency control keyed by project ID
//
// # Key Workflow Types
//
// [Workflow.Deploy]: Deploy new applications from container images
// [Workflow.Rollback]: Revert to previous deployment version
// [Workflow.Promote]: Mark deployment as production-ready
//
// # Deployment Sources
//
// The service supports multiple deployment sources:
//   - Source from build: Create from existing container image
//   - Source from git: Build from repository with Dockerfile
//
// # Integration Points
//
// - Build Services: Coordinates with Depot or Docker backends
// - Krane Agents: Container orchestration and deployment
// - Database: Persistent state and metadata management
// - Routing Service: Domain assignment and traffic management
// - Vault Service: Secure storage of secrets and certificates
//
// # Usage
//
// Creating deployment service:
//
//	deploymentSvc := deployment.New(deployment.Config{
//		Database:     mainDB,
//		Restate:      restateClient,
//		BuildService: buildService,
//		Logger:        logger,
//		DefaultDomain: "unkey.app",
//	})
//
//	// Deploy new application
//	_, err := deploymentSvc.Deploy(ctx, &hydrav1.DeployRequest{
//		DeploymentId: "deploy-123",
//		Source: &hydrav1.DeployRequest_Git{
//			Git: &hydrav1.DeployRequest_Git_Source{
//				Repository: "https://github.com/user/repo.git",
//				Branch:     "main",
//				CommitSha:  "abc123def456",
//			},
//		},
//	})
//
// # Error Handling
//
// The service uses comprehensive error handling with proper HTTP
// status codes and database transaction management.
package deployment
