// Package sync provides bidirectional synchronization between krane agents
// and the central control plane.
//
// This package implements the core synchronization engine that maintains
// consistency between desired state from the control plane and actual state
// in Kubernetes clusters. It uses streaming gRPC connections for real-time
// communication and includes resilience mechanisms for handling network failures.
//
// # Architecture
//
// The sync engine operates as a bidirectional bridge:
//   - Pull: Continuously streams desired state updates from control plane
//   - Push: Sends status updates and telemetry back to control plane
//   - Watch: Monitors Kubernetes resources for local changes
//   - Reconcile: Coordinates state changes between controllers and control plane
//
// # Resilience and Error Handling
//
// The system includes several resilience mechanisms:
//   - Circuit breakers prevent cascade failures during control plane outages
//   - Exponential backoff for retry operations with jitter to prevent thundering herd
//   - Local buffering to handle temporary network interruptions
//   - Graceful degradation when connectivity is limited
//
// # Event Flow
//
// 1. Control plane sends desired state events via streaming gRPC
// 2. Sync engine routes events to appropriate controllers
// 3. Controllers apply changes to Kubernetes resources
// 4. Status updates are buffered and sent back to control plane
// 5. Local resource changes are detected and reported
//
// # Integration
//
// The sync engine integrates with:
//   - DeploymentController: Handles application deployment events
//   - SentinelController: Manages sentinel component events
//   - Control Plane: gRPC client for bidirectional communication
//   - Kubernetes: Resource monitoring and status reporting
//
// # Usage
//
// The sync engine is created by the main krane agent and runs continuously:
//
//	cfg := sync.Config{
//		Logger:               logger,
//		Region:               "us-west-2",
//		Shard:                "production",
//		InstanceID:           "krane-001",
//		ControlPlaneURL:      "https://control-plane.example.com",
//		ControlPlaneBearer:   "bearer-token",
//		DeploymentController: deploymentCtrl,
//		SentinelController:   sentinelCtrl,
//	}
//	engine, err := sync.New(cfg)
//	if err != nil {
//		return err
//	}
//	// Engine starts automatically and runs until context cancellation
package sync
