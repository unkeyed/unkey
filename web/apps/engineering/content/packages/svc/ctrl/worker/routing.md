---
title: routing
description: "manages frontline route assignment for deployments"
---

Package routing manages frontline route assignment for deployments.

This package provides a Restate-based service for reassigning frontline routes (the edge routing layer) to point at different deployments. When a deployment is promoted or traffic needs to shift, this service updates the database records that control which deployment receives traffic for each route.

### Architecture

The routing service implements \[hydrav1.RoutingServiceServer] and runs as a Restate virtual object. Virtual objects provide serialized access per key, preventing race conditions when multiple operations target the same routes.

### Restate Integration

All route reassignment operations use Restate's durable execution model. Each route update is wrapped in \[restate.Run], which provides automatic retries on transient failures and exactly-once execution guarantees. If the service crashes mid-operation, Restate replays completed steps and resumes from where it left off.

### Key Operations

\[Service.AssignFrontlineRoutes] reassigns a set of frontline routes to a new deployment by updating the deployment\_id column in the frontline\_routes table.

### Usage

Create a routing service:

	routingSvc := routing.New(routing.Config{
		DB:            mainDB,
		DefaultDomain: "unkey.app",
	})

Register with Restate and invoke via the generated client to reassign routes during deployment promotion or rollback operations.

## Variables

```go
var _ hydrav1.RoutingServiceServer = (*Service)(nil)
```


## Types

### type Config

```go
type Config struct {
	// DB provides access to frontline route records.
	DB db.Database
	// DefaultDomain is the apex domain for generated deployment URLs.
	DefaultDomain string
}
```

Config holds the configuration for creating a \[Service].

### type Service

```go
type Service struct {
	hydrav1.UnimplementedRoutingServiceServer
	db            db.Database
	defaultDomain string
}
```

Service implements the routing service for frontline route management.

Service embeds \[hydrav1.UnimplementedRoutingServiceServer] to satisfy the gRPC interface. It uses Restate virtual objects to serialize route reassignment operations, preventing concurrent modifications to the same routes.

#### func New

```go
func New(cfg Config) *Service
```

New creates a new \[Service] with the provided configuration.

#### func (Service) AssignFrontlineRoutes

```go
func (s *Service) AssignFrontlineRoutes(ctx restate.ObjectContext, req *hydrav1.AssignFrontlineRoutesRequest) (*hydrav1.AssignFrontlineRoutesResponse, error)
```

AssignFrontlineRoutes reassigns a set of frontline routes to a new deployment.

Each route in FrontlineRouteIds is updated to point at DeploymentId. The updates are executed sequentially, with each wrapped in \[restate.Run] for durability. If any update fails, the operation returns an error and Restate will retry the entire handler from the last successful checkpoint.

Returns an empty response on success. Database errors from the route updates propagate directly to the caller.

