// Package labels defines Kubernetes label keys for Krane resource management.
//
// This package provides standardized label keys used across all Kubernetes
// resources managed by Krane. Consistent labeling enables reliable resource
// selection, filtering, and lifecycle management.
//
// # Label Strategy
//
// All Krane-managed resources use labels for:
//   - Resource identification (deployment-id, gateway-id)
//   - Management tracking (managed-by=krane)
//   - Hierarchical organization (workspace, project, environment)
//   - Operational metadata (version, component)
//
// # Label Naming Convention
//
// Labels follow Kubernetes best practices:
//   - Use lowercase letters, numbers, and hyphens
//   - No dots in keys (reserved for Kubernetes)
//   - Descriptive but concise names
//   - Consistent across all resource types
//
// # Usage Example
//
//	import "github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes/labels"
//
//	// Creating resources with standard labels
//	deployment := &appsv1.Deployment{
//	    ObjectMeta: metav1.ObjectMeta{
//	        Labels: map[string]string{
//	            labels.DeploymentID: "web-123",
//	            labels.ManagedBy:    "krane",
//	            labels.WorkspaceID:  "ws-456",
//	        },
//	    },
//	}
//
//	// Selecting resources by labels
//	selector := fmt.Sprintf("%s=%s", labels.DeploymentID, deploymentID)
//	list, err := client.List(ctx, metav1.ListOptions{
//	    LabelSelector: selector,
//	})
//
// # Important Considerations
//
// Label values must comply with Kubernetes restrictions:
//   - Maximum 63 characters
//   - Must start and end with alphanumeric
//   - Can contain dashes, underscores, dots
//
// Once applied, labels should be treated as immutable for resource
// identification. Changing labels may break resource selection and
// cause orphaned resources.
package labels
