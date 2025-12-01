// Package kubernetes implements idempotent container orchestration using Kubernetes API.
//
// This backend provides multi-node container deployment and management through
// the Kubernetes control plane. All operations are fully idempotent, leveraging
// Kubernetes' declarative model while adding additional safeguards for reliability.
//
// # Design Philosophy
//
// The Kubernetes backend implements the [backend.Backend] interface with:
//   - Idempotent apply operations that mirror kubectl apply semantics
//   - Label-based resource selection for consistent management
//   - StatefulSet-based deployments for stable network identities
//   - Deployment-based gateways for stateless scalability
//
// # Idempotency Implementation
//
// Apply operations (ApplyDeployment, ApplyGateway):
//   - List existing resources by label selector
//   - Reuse existing Services and StatefulSets/Deployments
//   - Only create missing resources
//   - Update owner references for proper cleanup
//
// Delete operations (DeleteDeployment, DeleteGateway):
//   - Use label selectors to find all related resources
//   - Continue deletion even if resources don't exist
//   - Return success for already-deleted resources
//
// The implementation goes beyond Kubernetes' native idempotency by:
//   - Checking for existing resources before creation
//   - Handling multiple resources with the same labels gracefully
//   - Ensuring owner references are properly set
//
// # Architecture Decisions
//
// StatefulSets for Deployments:
//
// We use StatefulSets instead of Deployments for application workloads because:
//   - Stable network identities (pod-name.service.namespace.svc.cluster.local)
//   - Predictable pod names for debugging
//   - Ordered rolling updates
//   - Better alignment with microVM abstraction patterns
//
// Note: This may be reconsidered in future versions as it diverges from
// cloud-native patterns for stateless services.
//
// Deployments for Gateways:
//
// Gateways use standard Deployments because:
//   - No need for stable network identities
//   - Better horizontal scaling
//   - Faster rollouts and rollbacks
//   - Standard ingress patterns
//
// # Resource Management
//
// Resources are organized as follows:
//
// Deployments create:
//   - Service (headless, ClusterIP: None)
//   - StatefulSet with pod template
//   - Owner references for cascade deletion
//
// Gateways create:
//   - Service (ClusterIP for internal access)
//   - Deployment with pod template
//   - Owner references for cascade deletion
//
// All resources use consistent labeling:
//   - deployment-id or gateway-id for identification
//   - managed-by=krane for management tracking
//   - Additional metadata labels for filtering
//
// # Service Discovery
//
// Deployments (StatefulSets):
//
//	{pod-name}-{ordinal}.{service-name}.{namespace}.svc.cluster.local
//	Example: myapp-0.myapp.default.svc.cluster.local
//
// Gateways (Deployments):
//
//	{service-name}.{namespace}.svc.cluster.local
//	Example: gateway.default.svc.cluster.local
//
// # Configuration
//
// The Kubernetes backend can be configured with:
//   - In-cluster config (when running inside Kubernetes)
//   - Kubeconfig file (for external access)
//   - Explicit API server configuration
//
// Example:
//
//	// In-cluster configuration
//	backend, err := kubernetes.New(kubernetes.Config{
//	    Logger: logger,
//	})
//
//	// External configuration
//	backend, err := kubernetes.NewFromKubeconfig(
//	    "/path/to/kubeconfig",
//	    logger,
//	)
//
// # Error Handling
//
// The backend handles Kubernetes API errors:
//   - IsAlreadyExists: Ignored for idempotency
//   - IsNotFound: Ignored for delete operations
//   - IsConflict: May trigger retry logic
//   - Network errors: Propagated for client retry
//
// # Performance Considerations
//
// Operations interact with the Kubernetes API server:
//   - Apply: 2-4 API calls (list, create/update, set owner)
//   - Delete: 2-3 API calls per resource type
//   - Get: 2 API calls (list StatefulSet/Deployment and pods)
//
// The backend uses label selectors and field selectors to minimize data transfer.
// Consider using ResourceVersion for watch operations in high-frequency scenarios.
//
// # Best Practices
//
// When using this backend:
//   - Always specify resource requests and limits
//   - Use appropriate replica counts for availability
//   - Configure pod disruption budgets for production
//   - Set up monitoring for resource utilization
//   - Use namespaces for multi-tenancy isolation
package kubernetes
