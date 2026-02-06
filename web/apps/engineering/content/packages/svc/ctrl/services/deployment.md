---
title: deployment
description: "provides the control-plane deployment service for managing"
---

Package deployment provides the control-plane deployment service for managing application deployments, promotions, and rollbacks.

This package implements a ConnectRPC service that orchestrates deployment workflows through Restate for durable execution. It acts as the API layer between clients (CLI, dashboard) and the underlying Hydra deployment workflows.

### Concurrency Model

All operations use Restate virtual objects keyed by project ID, ensuring only one deployment operation runs per project at a time. This prevents race conditions when multiple deployments, promotions, or rollbacks are triggered simultaneously for the same project.

### Deployment Sources

\[CreateDeployment] supports two deployment sources:

  - Build from source: provide a build context path (S3 key to a tar.gz archive) and optionally a Dockerfile path (defaults to "./Dockerfile")
  - Prebuilt image: provide a Docker image reference directly

### Workflow Lifecycle

Deployments follow this lifecycle:

 1. \[CreateDeployment] validates the request, stores metadata in the database with status "pending", and triggers an async Restate workflow
 2. The Hydra workflow (separate service) builds the image, deploys containers, and configures networking
 3. \[GetDeployment] retrieves current deployment status and metadata
 4. \[Promote] switches traffic to the target deployment
 5. \[Rollback] reverts traffic to a previous deployment

### Error Handling

All methods return Connect error codes following standard conventions: \[connect.CodeInvalidArgument] for validation errors, \[connect.CodeNotFound] for missing resources, and \[connect.CodeInternal] for system failures.

### Usage

Creating the deployment service:

	svc := deployment.New(deployment.Config{
		Database:         db,
		Restate:          restateClient,
		AvailableRegions: []string{"us-east-1", "eu-west-1"},
		BuildStorage:     s3Storage,
	})

## Constants

```go
const (
	// maxCommitMessageLength limits commit messages to prevent oversized database entries.
	maxCommitMessageLength = 10240
	// maxCommitAuthorHandleLength limits author handles (e.g., GitHub usernames).
	maxCommitAuthorHandleLength = 256
	// maxCommitAuthorAvatarLength limits avatar URL length.
	maxCommitAuthorAvatarLength = 512
)
```


## Functions


## Types

### type Config

```go
type Config struct {
	// Database provides read/write access to deployment metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable workflows.
	Restate *restateingress.Client
	// AvailableRegions lists the regions where deployments can be created.
	AvailableRegions []string
}
```

Config holds the configuration for creating a new \[Service].

### type Service

```go
type Service struct {
	ctrlv1connect.UnimplementedDeploymentServiceHandler
	db               db.Database
	restate          *restateingress.Client
	availableRegions []string
}
```

Service implements the DeploymentService ConnectRPC API. It coordinates deployment operations by persisting state to the database and delegating workflow execution to Restate.

#### func New

```go
func New(cfg Config) *Service
```

New creates a new \[Service] with the given configuration. All fields in \[Config] are required.

#### func (Service) CreateDeployment

```go
func (s *Service) CreateDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateDeploymentRequest],
) (*connect.Response[ctrlv1.CreateDeploymentResponse], error)
```

CreateDeployment creates a new deployment record and initiates an async Restate workflow. The deployment source must be a prebuilt Docker image.

The method looks up the project to infer the workspace, validates the environment exists, fetches environment variables, and persists the deployment with status "pending" before triggering the workflow. Git commit metadata is optional but validated when provided: timestamps must be Unix epoch milliseconds and cannot be more than one hour in the future.

The workflow runs asynchronously keyed by project ID, so only one deployment per project executes at a time. Returns the deployment ID and initial status.

#### func (Service) GetDeployment

```go
func (s *Service) GetDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.GetDeploymentRequest],
) (*connect.Response[ctrlv1.GetDeploymentResponse], error)
```

GetDeployment retrieves a deployment by ID including its current status, git metadata, and associated hostnames. Returns \[connect.CodeNotFound] if the deployment does not exist. Hostnames are fetched separately from frontline routes; if that lookup fails, the response still succeeds but with an empty hostname list.

#### func (Service) Promote

```go
func (s *Service) Promote(ctx context.Context, req *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error)
```

Promote reassigns all domains to the target deployment via a Restate workflow. This is typically used after a rollback to restore the original deployment, or to switch traffic to a new deployment that was previously in a preview state. The workflow runs synchronously (blocking until complete) and is keyed by project ID to prevent concurrent promotion operations on the same project.

#### func (Service) Rollback

```go
func (s *Service) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error)
```

Rollback switches traffic from the source deployment to a previous target deployment via a Restate workflow. This is typically called from the dashboard when a deployment needs to be reverted. The workflow runs synchronously (blocking until complete) and is keyed by project ID to prevent concurrent rollback operations on the same project.

