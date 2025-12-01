// Package docker implements idempotent container orchestration using Docker Engine API.
//
// This backend provides single-node container deployment and management through
// direct Docker daemon interaction. All operations are fully idempotent, allowing
// safe retries and recovery from failures without creating duplicate resources.
//
// # Design Philosophy
//
// The Docker backend implements the [backend.Backend] interface with a focus on:
//   - Idempotent operations that can be safely retried
//   - Label-based resource management for reliable identification
//   - Graceful handling of existing resources
//   - Force removal to ensure cleanup succeeds
//
// # Idempotency Implementation
//
// Apply operations (ApplyDeployment, ApplyGateway):
//   - List existing containers by label before creation
//   - Skip creation if containers already exist
//   - Handle name conflicts gracefully (race conditions)
//   - Continue operation even if some containers exist
//
// Delete operations (DeleteDeployment, DeleteGateway):
//   - Return success if no containers exist (already deleted)
//   - Continue deletion even if some containers are missing
//   - Use force removal to handle stuck containers
//
// This approach ensures operations can be retried indefinitely without errors
// or resource duplication.
//
// # Resource Identification
//
// Containers are identified using Docker labels:
//   - Deployments: "unkey.deployment.id" label
//   - Gateways: "unkey.gateway.id" label
//   - All resources: "unkey.managed.by=krane" label
//
// Using labels instead of container names provides flexibility and allows
// multiple containers per deployment (for replicas).
//
// # Container Lifecycle
//
// Container creation follows this pattern:
//  1. Check if image exists locally (pull if needed)
//  2. List existing containers with matching labels
//  3. Create only missing containers
//  4. Start containers with restart policy
//
// Container deletion follows this pattern:
//  1. List all containers with matching labels
//  2. Force remove each container
//  3. Continue even if containers don't exist
//
// # Configuration
//
// The Docker backend requires:
//   - Docker daemon socket path (typically /var/run/docker.sock)
//   - Optional registry credentials for private images
//   - Logger for operational visibility
//
// Example configuration:
//
//	cfg := docker.Config{
//	    Logger:     logger,
//	    SocketPath: "/var/run/docker.sock",
//	    RegistryURL: "registry.example.com",
//	    RegistryUsername: "user",
//	    RegistryPassword: "pass",
//	}
//	backend, err := docker.New(cfg)
//
// # Limitations
//
// The Docker backend is designed for single-node deployments:
//   - No cross-node networking
//   - No distributed storage
//   - No automatic failover
//   - Limited to single Docker daemon
//
// For multi-node deployments, use the Kubernetes backend.
//
// # Error Handling
//
// Errors are handled gracefully:
//   - "No such container" errors are ignored (idempotency)
//   - "Conflict" errors trigger retry logic
//   - "Already started" errors are ignored
//   - Network errors are propagated for client retry
//
// # Performance Considerations
//
// Operations have these performance characteristics:
//   - Apply: O(n) where n is number of replicas
//   - Delete: O(n) where n is number of existing containers
//   - Get: O(n) container list operation
//
// The backend uses Docker's filtering to minimize data transfer.
package docker
