// Package ctrl provides administrative and health check services.
//
// This service implements control plane management operations including
// liveness checks, system status monitoring, and basic
// administrative functions. It provides the core health
// and management endpoints needed for operational control
// of the unkey platform.
//
// # Key Operations
//
// [Liveness]: Health check endpoint for load balancers and monitoring
//
//	Used to verify control plane is running and responsive
//
// The service uses a simple instance-based identification
// system to track multiple control plane instances in a
// distributed deployment.
//
// # Configuration
//
// The service requires an instance identifier for proper
// logging and correlation in distributed environments.
// It operates directly with the database for persistence.
//
// # Usage
//
// Creating control service:
//
//	ctrlSvc := ctrl.New("ctrl-instance-001", database)
//
//	// Register with Connect server
//	mux.Handle(ctrlv1connect.NewCtrlServiceHandler(ctrlSvc))
//
// # Error Handling
//
// Uses standard Connect error codes for proper
// error transmission and client handling.
package ctrl
