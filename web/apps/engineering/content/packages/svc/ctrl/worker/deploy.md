---
title: deploy
description: "implements deployment lifecycle orchestration workflows"
---

Package deploy implements deployment lifecycle orchestration workflows.

This package manages the complete deployment lifecycle including deploying new versions, rolling back to previous versions, and promoting deployments. It coordinates between container orchestration (Krane), database updates, domain routing, and sentinel configuration to ensure consistent deployment state.

### Built on Restate

All workflows in this package are built on top of Restate (restate.dev) for durable execution. This provides critical guarantees:

\- Automatic retries on transient failures - Exactly-once execution semantics for each workflow step - Durable state that survives process crashes and restarts - Virtual object concurrency control keyed by project ID

The virtual object model ensures that only one deployment operation runs per project at any given time, preventing race conditions during concurrent deploy/rollback/promote operations that could leave the system in an inconsistent state.

### Key Types

\[Workflow] is the main entry point that implements deployment orchestration. It provides three primary operations:

\- \[Workflow.Deploy] - Deploy a new Docker image and configure routing - \[Workflow.Rollback] - Roll back to a previous deployment - \[Workflow.Promote] - Promote a deployment to live and clear rollback state

### Usage

The workflow is typically initialized with database connections, a Krane client, and configuration:

	workflow := deploy.New(deploy.Config{
	    DB:            mainDB,
	    Krane:         kraneClient,
	    DefaultDomain: "unkey.app",
	})

Deploy a new version:

	_, err := workflow.Deploy(ctx, &hydrav1.DeployRequest{
	    DeploymentId: "dep_123",
	    DockerImage:  "myapp:v1.2.3",
	    KeyAuthId:    "key_auth_456", // optional
	})

Rollback to previous version:

	_, err := workflow.Rollback(ctx, &hydrav1.RollbackRequest{
	    SourceDeploymentId: "dep_current",
	    TargetDeploymentId: "dep_previous",
	})

Promote a deployment to live:

	_, err := workflow.Promote(ctx, &hydrav1.PromoteRequest{
	    TargetDeploymentId: "dep_123",
	})

### Deployment Flow

The deployment process follows these steps:

1\. Deployment lookup - Find and validate deployment record 2. Context gathering - Load workspace, project, and environment data 3. Status update to building - Mark deployment as in-progress 4. Container deployment - Create deployment in Krane 5. Polling for readiness - Wait for all instances to be running 6. VM registration - Register running instances in DB 7. OpenAPI scraping - Fetch API spec from running instances (if available) 8. Domain assignment - Create/update domains and sentinel configs via routing service 9. Status update to ready - Mark deployment as live 10. Project update - Update live deployment pointer (if production)

Each step is wrapped in a restate.Run call, making it durable and retryable. If the workflow crashes at any point, Restate will resume from the last completed step rather than restarting from the beginning. The deferred error handler ensures that failed deployments are properly marked in the database even if the workflow is interrupted.

### Rollback and Promote

Rollbacks switch sticky domains (environment and live domains) from the current deployment to a previous deployment. This is done atomically through the routing service to prevent partial updates. The project is marked as rolled back to prevent new deployments from automatically taking over live domains.

Promotion reverses a rollback by switching domains to a new deployment and clearing the rolled back flag. This allows normal deployment flow to resume.

### Domain Generation

The package generates multiple domain patterns per deployment:

\- Per-commit: \`\<project>-git-\<sha>-\<workspace>.\<apex>\` (never reassigned) - Per-branch: \`\<project>-git-\<branch>-\<workspace>.\<apex>\` (sticky to branch) - Per-environment: \`\<project>-\<env>-\<workspace>.\<apex>\` (sticky to environment)

The sticky behavior ensures that branch and environment domains follow the latest deployment for that branch/environment, while commit domains remain immutable.

### Sentinel Configuration

Sentinel configs are created for all domains (except localhost and .local/.test TLDs) and stored as JSON in the database. Each config includes:

\- Deployment ID and enabled status - VM addresses for load balancing - Optional auth configuration (key auth ID) - Optional validation configuration (OpenAPI spec)

Sentinel configs use protojson encoding for easier debugging and direct database inspection.

### Error Handling

The package uses Restate's error handling model with deferred cleanup. If any step fails, the deployment status is automatically updated to "failed". Terminal errors with appropriate HTTP status codes are returned for client errors (invalid input, not found, etc.). System errors are returned for unexpected failures that may be retried by Restate.

## Constants

```go
const (
	// defaultCacheKeepGB is the maximum cache size in gigabytes for new Depot
	// projects. Depot evicts least-recently-used cache entries when exceeded.
	defaultCacheKeepGB = 50

	// defaultCacheKeepDays is the maximum age in days for cached build layers.
	// Layers older than this are evicted regardless of cache size.
	defaultCacheKeepDays = 14
)
```

```go
const (
	// sentinelNamespace isolates sentinel resources from tenant namespaces to
	// simplify RBAC and keep routing infrastructure separate from workloads.
	sentinelNamespace = "sentinel"

	// sentinelPort is the port exposed by sentinel services for frontline traffic
	// and must match the container port and service configuration.
	sentinelPort = 8040
)
```

```go
const (
	// deploymentPort is the port exposed by customer deployments for sentinel traffic.
	deploymentPort = 8080
)
```


## Variables

```go
var (
	// nonAlphanumericRegex removes characters that are unsafe for domain slugs and
	// avoids double hyphens when combined with whitespace normalization.
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9\s]`)

	// multipleSpacesRegex collapses consecutive whitespace before hyphen conversion.
	multipleSpacesRegex = regexp.MustCompile(`\s+`)
)
```

```go
var _ hydrav1.DeploymentServiceServer = (*Workflow)(nil)
```


## Functions


## Types

### type BuildPlatform

```go
type BuildPlatform struct {
	Platform     string
	Architecture string
}
```

BuildPlatform specifies the target platform for container builds.

### type Config

```go
type Config struct {
	// DB is the main database connection for workspace, project, and deployment data.
	DB db.Database

	// DefaultDomain is the apex domain for generated deployment URLs (e.g., "unkey.app").
	DefaultDomain string

	// Vault provides encryption/decryption services for secrets.
	Vault vaultv1connect.VaultServiceClient

	// SentinelImage is the Docker image used for sentinel containers.
	SentinelImage string

	// AvailableRegions is the list of available regions for deployments.
	AvailableRegions []string

	// GitHub provides access to GitHub API for downloading tarballs.
	GitHub githubclient.GitHubClient

	// DepotConfig configures the Depot API connection.
	DepotConfig DepotConfig

	// RegistryConfig provides credentials for the container registry.
	RegistryConfig RegistryConfig

	// BuildPlatform specifies the target platform for all builds.
	BuildPlatform BuildPlatform

	// Clickhouse receives build step telemetry for observability.
	Clickhouse clickhouse.ClickHouse
}
```

Config holds the configuration for creating a deployment workflow.

### type DepotConfig

```go
type DepotConfig struct {
	APIUrl        string
	ProjectRegion string
}
```

DepotConfig holds configuration for connecting to the Depot.dev API.

### type RegistryConfig

```go
type RegistryConfig struct {
	URL      string
	Username string
	Password string
}
```

RegistryConfig holds credentials for the container registry.

### type Workflow

```go
type Workflow struct {
	hydrav1.UnimplementedDeploymentServiceServer
	db db.Database

	defaultDomain    string
	vault            vaultv1connect.VaultServiceClient
	sentinelImage    string
	availableRegions []string
	github           githubclient.GitHubClient

	// Build dependencies
	depotConfig    DepotConfig
	registryConfig RegistryConfig
	buildPlatform  BuildPlatform
	clickhouse     clickhouse.ClickHouse
}
```

Workflow orchestrates deployment lifecycle operations.

This workflow manages the complete deployment lifecycle including deploying new versions, rolling back to previous versions, and promoting deployments to live. It coordinates between container orchestration (Krane), database updates, domain routing, and sentinel configuration to ensure consistent deployment state.

The workflow uses Restate virtual objects keyed by project ID to ensure that only one deployment operation runs per project at any time, preventing race conditions during concurrent deploy/rollback/promote operations.

#### func New

```go
func New(cfg Config) *Workflow
```

New creates a new deployment workflow instance.

#### func (Workflow) Deploy

```go
func (w *Workflow) Deploy(ctx restate.WorkflowSharedContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error)
```

Deploy executes a full deployment workflow for a new application version.

This durable workflow orchestrates the complete deployment lifecycle: building Docker images (if source is provided), provisioning containers across regions, waiting for instances to become healthy, and configuring domain routing. The workflow is idempotent and can safely resume from any step after a crash.

The deployment request must specify either a build context path (to build from source) or a pre-built Docker image. If BuildContextPath is set, the workflow triggers a Docker build through the build service before deployment. Otherwise, the provided DockerImage is deployed directly.

The workflow creates deployment topologies for all configured regions, each with its own version number for independent scaling and rollback. Sentinel containers are automatically provisioned for environments that don't already have them, with production environments getting 3 replicas and others getting 1.

Domain routing is configured through frontline routes, with sticky domains (branch and environment) automatically updating to point to the new deployment. For production deployments, the project's live deployment pointer is updated unless the project is in a rolled-back state.

If any step fails, the deployment status is automatically set to failed via a deferred cleanup handler, ensuring the database reflects the true deployment state.

Returns terminal errors for validation failures (missing image/context) and retryable errors for transient system failures.

#### func (Workflow) Promote

```go
func (w *Workflow) Promote(ctx restate.WorkflowSharedContext, req *hydrav1.PromoteRequest) (*hydrav1.PromoteResponse, error)
```

Promote reassigns all sticky domains to a deployment and clears the rolled back state.

This durable workflow moves sticky domains (environment and live domains) from the current live deployment to a new target deployment. It reverses a previous rollback and allows normal deployment flow to resume.

The workflow validates that: - Target deployment is ready (not building, deploying, or failed) - Target deployment has running VMs - Target deployment is not already the live deployment - Project has sticky domains to promote

After switching domains atomically through the routing service, the project's live deployment pointer is updated and the rolled back flag is cleared, allowing future deployments to automatically take over sticky domains.

Returns terminal errors (400/404) for validation failures and retryable errors for system failures.

#### func (Workflow) Rollback

```go
func (w *Workflow) Rollback(ctx restate.WorkflowSharedContext, req *hydrav1.RollbackRequest) (*hydrav1.RollbackResponse, error)
```

Rollback performs a rollback to a previous deployment.

This durable workflow switches sticky frontlineRoutes (environment and live frontlineRoutes) from the current live deployment back to a previous deployment. The operation is performed atomically through the routing service to prevent partial updates that could leave the system in an inconsistent state.

The workflow validates that: - Source deployment is the current live deployment - Target deployment has running VMs - Both deployments are in the same project and environment - There are sticky frontlineRoutes to rollback

After switching frontlineRoutes, the project is marked as rolled back to prevent new deployments from automatically taking over the live frontlineRoutes.

Returns terminal errors (400/404) for validation failures and retryable errors for system failures.

