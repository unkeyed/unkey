---
title: ctrl
description: "provides administrative and health check services"
---

Package ctrl provides administrative and health check services.

This service implements control plane management operations including liveness checks, system status monitoring, and basic administrative functions. It provides the core health and management endpoints needed for operational control of the unkey platform.

### Key Operations

\[Liveness]: Health check endpoint for load balancers and monitoring

	Used to verify control plane is running and responsive

The service uses a simple instance-based identification system to track multiple control plane instances in a distributed deployment.

### Configuration

The service requires an instance identifier for proper logging and correlation in distributed environments. It operates directly with the database for persistence.

### Usage

Creating control service:

	ctrlSvc := ctrl.New("ctrl-instance-001", database)

	// Register with Connect server
	mux.Handle(ctrlv1connect.NewCtrlServiceHandler(ctrlSvc))

### Error Handling

Uses standard Connect error codes for proper error transmission and client handling.

## Types

### type Service

```go
type Service struct {
	ctrlv1connect.UnimplementedCtrlServiceHandler
	instanceID string
	db         db.Database
}
```

#### func New

```go
func New(instanceID string, database db.Database) *Service
```

#### func (Service) Liveness

```go
func (s *Service) Liveness(
	ctx context.Context,
	req *connect.Request[ctrlv1.LivenessRequest],
) (*connect.Response[ctrlv1.LivenessResponse], error)
```

