// Package deployment provides the control-plane deployment service for managing
// application deployments, promotions, and rollbacks.
//
// This package implements a ConnectRPC service that orchestrates deployment
// workflows through Restate for durable execution. It acts as the API layer
// between clients (CLI, dashboard) and the underlying Hydra deployment workflows.
//
// # Concurrency Model
//
// All operations use Restate virtual objects keyed by project ID, ensuring only
// one deployment operation runs per project at a time. This prevents race
// conditions when multiple deployments, promotions, or rollbacks are triggered
// simultaneously for the same project.
//
// # Deployment Sources
//
// [CreateDeployment] supports two deployment sources:
//
//   - Build from source: provide a build context path (S3 key to a tar.gz archive)
//     and optionally a Dockerfile path (defaults to "./Dockerfile")
//   - Prebuilt image: provide a Docker image reference directly
//
// # Workflow Lifecycle
//
// Deployments follow this lifecycle:
//
//  1. [CreateDeployment] validates the request, stores metadata in the database
//     with status "pending", and triggers an async Restate workflow
//  2. The Hydra workflow (separate service) builds the image, deploys containers,
//     and configures networking
//  3. [GetDeployment] retrieves current deployment status and metadata
//  4. [Promote] switches traffic to the target deployment
//  5. [Rollback] reverts traffic to a previous deployment
//
// # Error Handling
//
// All methods return Connect error codes following standard conventions:
// [connect.CodeInvalidArgument] for validation errors, [connect.CodeNotFound]
// for missing resources, and [connect.CodeInternal] for system failures.
//
// # Usage
//
// Creating the deployment service:
//
//	svc := deployment.New(deployment.Config{
//		Database:         db,
//		Restate:          restateClient,
//		Logger:           logger,
//		AvailableRegions: []string{"us-east-1", "eu-west-1"},
//		BuildStorage:     s3Storage,
//	})
package deployment
