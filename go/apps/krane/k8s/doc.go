// Package k8s provides utilities for interacting with Kubernetes resources.
//
// This package contains helper functions and types for managing Kubernetes
// resources, particularly focusing on label management for Krane-managed
// deployments. The label utilities follow Kubernetes best practices and
// conventions for resource labeling.
//
// # Label Management
//
// The primary functionality uses a builder pattern for creating and managing
// Kubernetes labels. This approach provides a fluent interface for composing
// labels while maintaining immutability and type safety.
//
// Basic usage for creating new label maps:
//
//	labels := k8s.NewLabels().
//	    WorkspaceID("ws_123").
//	    DeploymentID("dep_456").
//	    ManagedByKrane().
//	    ToMap()
//	// labels now contains:
//	// {
//	//     "unkey.com/workspace.id": "ws_123",
//	//     "unkey.com/deployment.id": "dep_456",
//	//     "app.kubernetes.io/managed-by": "krane"
//	// }
//
// Starting from existing labels:
//
//	existing := map[string]string{"env": "production"}
//	labels := k8s.From(existing).
//	    Version("v1.2.3").
//	    Component("backend").
//	    ToMap()
//	// Original 'existing' map is unchanged
//	// 'labels' contains all labels including the original ones
//
// Using the Merge escape hatch for existing maps:
//
//	customLabels := map[string]string{"team": "platform", "tier": "api"}
//	labels := k8s.NewLabels().
//	    WorkspaceID("ws_123").
//	    Merge(customLabels).
//	    ManagedByKrane().
//	    ToMap()
//
// For label selectors, use the provided constants:
//
//	selector := fmt.Sprintf("%s=%s", k8s.LabelKeyGatewayID, gatewayID)
//
// Reading labels from existing resources:
//
//	// Get specific label values
//	workspaceID := k8s.GetWorkspaceID(pod.Labels)
//	if k8s.IsManagedByKrane(pod.Labels) {
//	    // Handle Krane-managed resources
//	}
//
//	// Generic label access
//	if value, exists := k8s.GetLabel(pod.Labels, "custom.io/field"); exists {
//	    // Use the custom label value
//	}
//
//	// Check if labels match expected values
//	expected := map[string]string{
//	    k8s.LabelKeyWorkspaceID: "ws_123",
//	    k8s.LabelKeyComponent: "api",
//	}
//	if k8s.MatchesLabels(pod.Labels, expected) {
//	    // Labels match
//	}
//
// # Label Conventions
//
// This package follows Kubernetes labeling conventions:
//   - Uses "unkey.com/" prefix for organization-specific labels
//   - Follows "app.kubernetes.io/" prefix for standard Kubernetes labels
//   - All label values are strings and should be kept under 63 characters
//   - Empty strings are ignored to prevent creating invalid labels
//
// # Design Rationale
//
// We use the builder pattern for several reasons:
//   - Fluent interface provides excellent developer experience
//   - Immutability ensures the original maps are never modified
//   - Method chaining makes label composition clear and readable
//   - Empty value checking prevents invalid labels
//   - The Merge method provides an escape hatch for existing label maps
//   - Label key constants enable use in selectors and direct comparisons
//
// The builder creates a copy when using From() or ToMap() to ensure that
// modifications don't affect the original data structures.
package k8s
