// Package routing implements domain assignment and gateway configuration workflows.
//
// This package manages the relationship between domains, deployments, and gateway
// configurations. It handles creating new domain assignments during deployments and
// switching existing domains between deployments during rollback/promote operations.
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
// The virtual object model ensures that domain operations for a project are serialized,
// preventing race conditions where concurrent operations could create inconsistent routing
// state between the main database and partition database.
//
// # Key Types
//
// [Service] is the main entry point that implements routing operations.
// It provides two primary operations:
//
// - [Service.AssignDomains] - Create or reassign domains during deployment
// - [Service.SwitchDomains] - Switch existing domains during rollback/promote
//
// # Usage
//
// The service is typically initialized with database connections:
//
//	svc := routing.New(routing.Config{
//	    DB:            mainDB,
//	    PartitionDB:   partitionDB,
//	    Logger:        logger,
//	    DefaultDomain: "unkey.app",
//	})
//
// Assign domains during deployment:
//
//	resp, err := svc.AssignDomains(ctx, &hydrav1.AssignDomainsRequest{
//	    WorkspaceId:   "ws_123",
//	    ProjectId:     "proj_456",
//	    EnvironmentId: "env_789",
//	    DeploymentId:  "dep_abc",
//	    Domains: []*hydrav1.DomainToAssign{
//	        {Name: "api.example.com", Sticky: hydrav1.DomainSticky_DOMAIN_STICKY_ENVIRONMENT},
//	    },
//	    GatewayConfig: gatewayConfig,
//	    IsRolledBack:  false,
//	})
//
// Switch domains during rollback/promote:
//
//	_, err := svc.SwitchDomains(ctx, &hydrav1.SwitchDomainsRequest{
//	    TargetDeploymentId: "dep_previous",
//	    DomainIds:          []string{"dom_1", "dom_2"},
//	})
//
// # Domain Assignment Flow
//
// The AssignDomains operation performs these steps:
//
// 1. For each domain, check if it exists in the database
// 2. If new, create domain record with specified sticky behavior
// 3. If existing and not rolled back, reassign to new deployment
// 4. If existing and rolled back, skip reassignment
// 5. Create gateway configs for all changed domains (except local hostnames)
// 6. Return list of domains that were actually modified
//
// Each domain upsert is wrapped in a restate.Run call with a unique name, allowing
// partial completion tracking. If the workflow fails after creating some domains,
// Restate will skip the already-created domains on retry.
//
// # Domain Switching Flow
//
// The SwitchDomains operation performs these steps:
//
// 1. Fetch gateway config for the target deployment
// 2. Fetch domain information (hostnames, workspace IDs) for given domain IDs
// 3. Upsert gateway configs first (atomic update of routing)
// 4. Reassign domains to the target deployment
//
// The gateway configs are updated before domain reassignment to ensure that when a domain
// points to a new deployment, the gateway config is already in place. This prevents a
// window where a domain might route to a deployment without proper configuration.
//
// # Sticky Domain Behavior
//
// Domains can have different sticky behaviors:
//
// - UNSPECIFIED: Per-commit domain, never reassigned (immutable)
// - BRANCH: Sticky to branch, follows latest deployment for that branch
// - ENVIRONMENT: Sticky to environment, follows latest deployment for that environment
// - LIVE: Sticky to live deployment, follows the current production deployment
//
// During deployment, sticky domains (branch, environment, live) are automatically reassigned
// to point to the new deployment. Non-sticky domains remain pointing to their original
// deployment, allowing stable URLs for specific versions.
//
// # Local Hostname Handling
//
// Gateway configs are NOT created for local development hostnames (localhost, 127.0.0.1,
// *.local, *.test). This prevents unnecessary config creation during local development.
// Hostnames using the default domain (e.g., *.unkey.app) ARE configured, as they represent
// production/staging environments.
//
// # Gateway Configuration Format
//
// Gateway configs are stored as JSON (using protojson.Marshal) in the partition database.
// This format was chosen for easier debugging and direct database inspection during
// development. Each config includes deployment ID, VM list, optional auth config, and
// optional validation config.
//
// # Atomicity and Consistency
//
// Domain assignment and switching operations affect two databases (main DB for domains,
// partition DB for gateway configs). While not using distributed transactions, the
// operations are ordered carefully:
//
// - On assignment: Domains first, then gateway configs
// - On switching: Gateway configs first, then domain reassignment
//
// Restate's durable execution ensures that if either step fails, the operation will be
// retried until both complete, maintaining eventual consistency between the databases.
//
// # Error Handling
//
// The package uses Restate's error handling model. Terminal errors with appropriate HTTP
// status codes are returned for client errors. System errors are returned for unexpected
// failures that may be retried. Partial failures during bulk operations are logged but
// do not fail the entire operation.
package routing
