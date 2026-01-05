// Package routing manages domain assignment and traffic routing workflows.
//
// This package orchestrates the relationship between domains,
// deployments, and sentinel configurations. It handles creating
// new domain assignments during deployments and switching
// domains during rollback or promotion operations.
//
// # Architecture
//
// The routing service manages domain lifecycle and ensures
// traffic is routed to the correct deployments. It provides:
//   - Domain assignment during deployment creation
//   - Sticky domain behavior for different deployment types
//   - Atomic domain switching during operations
//   - Integration with sentinel configurations
//
// # Built on Restate
//
// All routing workflows use Restate for durable execution:
//   - Automatic retries on transient failures
//   - Exactly-once guarantees for each workflow step
//   - Durable state that survives process crashes
//   - Virtual object concurrency control keyed by project ID
//
// # Key Operations
//
// [AssignDomains]: Create domain assignments for new deployments
// [SwitchDomains]: Reassign domains during rollback/promote operations
//
// # Domain Behavior Types
//
// - UNSPECIFIED: Per-commit domains (immutable, never reassigned)
// - BRANCH: Per-branch domains (follows latest deployment for branch)
// - ENVIRONMENT: Per-environment domains (follows latest deployment for environment)
// - LIVE: Per-live domains (follows current production deployment)
//
// # Sentinel Configuration
//
// Sentinel configs are automatically created for all domains
// (except local development hostnames) and stored as JSON
// in the database. Each config includes deployment ID,
// VM addresses for load balancing, and optional auth/validation configs.
//
// # Usage
//
// Creating routing service:
//
//	routingSvc := routing.New(routing.Config{
//		DB:            mainDB,
//		Logger:        logger,
//		DefaultDomain: "unkey.app",
//	})
//
// Assign domains during deployment:
//
//	resp, err := routingSvc.AssignDomains(ctx, &hydrav1.AssignDomainsRequest{
//		WorkspaceId:   "ws_123",
//		ProjectId:     "proj_456",
//		DeploymentId:  "dep_abc",
//		Domains: []*hydrav1.DomainToAssign{
//		{Name: "api.example.com", Sticky: hydrav1.DomainSticky_ENVIRONMENT},
//	},
//		IsRolledBack: false,
//	})
//
// # Error Handling
//
// The service ensures atomic operations and provides detailed error
// reporting for routing failures and domain conflicts.
package routing
