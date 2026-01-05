// Package labels provides standardized label management for krane Kubernetes resources.
//
// This package defines a consistent labeling scheme for all krane-managed resources
// using both Kubernetes standard labels and unkey.com organization-specific labels.
// Labels are used for resource selection, identification, and organization.
//
// # Label Conventions
//
// The package follows these labeling conventions:
//   - Kubernetes standard labels use "app.kubernetes.io/" prefix
//   - Organization-specific labels use "unkey.com/" prefix
//   - Component type is identified by "app.kubernetes.io/component" label
//   - Resource ownership is identified by "unkey.com/[resource].id" labels
//
// # Label Types
//
// Resource identification labels:
//   - unkey.com/workspace.id - Workspace identifier
//   - unkey.com/project.id - Project identifier
//   - unkey.com/environment.id - Environment identifier
//   - unkey.com/deployment.id - Deployment identifier
//   - unkey.com/sentinel.id - Sentinel identifier
//
// Component and management labels:
//   - app.kubernetes.io/managed-by - Always "krane" for managed resources
//   - app.kubernetes.io/component - Either "deployment" or "sentinel"
//
// # Key Types
//
// [Labels]: Map type for building and manipulating label sets
//
// # Usage
//
// Building labels for deployment:
//
//	labels := labels.New().
//		WithWorkspaceID("ws-123").
//		WithProjectID("proj-456").
//		WithEnvironmentID("env-789").
//		WithDeploymentID("deploy-001").
//		WithManagedByKrane().
//		WithComponentDeployment()
//
// Converting to Kubernetes selector:
//
//	selector := labels.ToString()
//	// Result: "unkey.com/workspace.id=ws-123,unkey.com/project.id=proj-456,..."
//
// Extracting specific IDs from existing labels:
//
//	deploymentID, found := labels.GetDeploymentID(existingLabels)
//	if !found {
//		// Handle missing deployment ID
//	}
package labels
