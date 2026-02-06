// Package deploy implements deployment lifecycle orchestration workflows.
//
// This package manages the complete deployment lifecycle including deploying new versions,
// rolling back to previous versions, and promoting deployments. It coordinates between
// container orchestration (Krane), database updates, domain routing, and sentinel configuration
// to ensure consistent deployment state.
//
// # Built on Restate
//
// All workflows in this package are built on top of Restate (restate.dev) for durable
// execution. This provides critical guarantees:
//
// - Automatic retries on transient failures
// - Exactly-once execution semantics for each workflow step
// - Durable state that survives process crashes and restarts
// - Virtual object concurrency control keyed by project ID
//
// The virtual object model ensures that only one deployment operation runs per project
// at any given time, preventing race conditions during concurrent deploy/rollback/promote
// operations that could leave the system in an inconsistent state.
//
// # Key Types
//
// [Workflow] is the main entry point that implements deployment orchestration.
// It provides three primary operations:
//
// - [Workflow.Deploy] - Deploy a new Docker image and configure routing
// - [Workflow.Rollback] - Roll back to a previous deployment
// - [Workflow.Promote] - Promote a deployment to live and clear rollback state
//
// # Usage
//
// The workflow is typically initialized with database connections, a Krane client,
// and configuration:
//
//	workflow := deploy.New(deploy.Config{
//	    DB:            mainDB,
//	    Krane:         kraneClient,
//	    DefaultDomain: "unkey.app",
//	})
//
// Deploy a new version:
//
//	_, err := workflow.Deploy(ctx, &hydrav1.DeployRequest{
//	    DeploymentId: "dep_123",
//	    DockerImage:  "myapp:v1.2.3",
//	    KeyAuthId:    "key_auth_456", // optional
//	})
//
// Rollback to previous version:
//
//	_, err := workflow.Rollback(ctx, &hydrav1.RollbackRequest{
//	    SourceDeploymentId: "dep_current",
//	    TargetDeploymentId: "dep_previous",
//	})
//
// Promote a deployment to live:
//
//	_, err := workflow.Promote(ctx, &hydrav1.PromoteRequest{
//	    TargetDeploymentId: "dep_123",
//	})
//
// # Deployment Flow
//
// The deployment process follows these steps:
//
// 1. Deployment lookup - Find and validate deployment record
// 2. Context gathering - Load workspace, project, and environment data
// 3. Status update to building - Mark deployment as in-progress
// 4. Container deployment - Create deployment in Krane
// 5. Polling for readiness - Wait for all instances to be running
// 6. VM registration - Register running instances in DB
// 7. OpenAPI scraping - Fetch API spec from running instances (if available)
// 8. Domain assignment - Create/update domains and sentinel configs via routing service
// 9. Status update to ready - Mark deployment as live
// 10. Project update - Update live deployment pointer (if production)
//
// Each step is wrapped in a restate.Run call, making it durable and retryable. If the
// workflow crashes at any point, Restate will resume from the last completed step rather
// than restarting from the beginning. The deferred error handler ensures that failed
// deployments are properly marked in the database even if the workflow is interrupted.
//
// # Rollback and Promote
//
// Rollbacks switch sticky domains (environment and live domains) from the current deployment
// to a previous deployment. This is done atomically through the routing service to prevent
// partial updates. The project is marked as rolled back to prevent new deployments from
// automatically taking over live domains.
//
// Promotion reverses a rollback by switching domains to a new deployment and clearing the
// rolled back flag. This allows normal deployment flow to resume.
//
// # Domain Generation
//
// The package generates multiple domain patterns per deployment:
//
// - Per-commit: `<project>-git-<sha>-<workspace>.<apex>` (never reassigned)
// - Per-branch: `<project>-git-<branch>-<workspace>.<apex>` (sticky to branch)
// - Per-environment: `<project>-<env>-<workspace>.<apex>` (sticky to environment)
//
// The sticky behavior ensures that branch and environment domains follow the latest
// deployment for that branch/environment, while commit domains remain immutable.
//
// # Sentinel Configuration
//
// Sentinel configs are created for all domains (except localhost and .local/.test TLDs)
// and stored as JSON in the database. Each config includes:
//
// - Deployment ID and enabled status
// - VM addresses for load balancing
// - Optional auth configuration (key auth ID)
// - Optional validation configuration (OpenAPI spec)
//
// Sentinel configs use protojson encoding for easier debugging and direct database inspection.
//
// # Error Handling
//
// The package uses Restate's error handling model with deferred cleanup. If any step fails,
// the deployment status is automatically updated to "failed". Terminal errors with appropriate
// HTTP status codes are returned for client errors (invalid input, not found, etc.). System
// errors are returned for unexpected failures that may be retried by Restate.
package deploy
