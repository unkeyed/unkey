package k8s

import "k8s.io/apimachinery/pkg/labels"

// Label key constants for direct access when needed (e.g., in label selectors)
const (
	// LabelKeyWorkspaceID is the label key for workspace ID
	LabelKeyWorkspaceID = "unkey.com/workspace.id"
	// LabelKeyDeploymentID is the label key for deployment ID
	LabelKeyDeploymentID = "unkey.com/deployment.id"
	// LabelKeySentinelID is the label key for sentinel ID
	LabelKeySentinelID = "unkey.com/sentinel.id"
	// LabelKeyProjectID is the label key for project ID
	LabelKeyProjectID = "unkey.com/project.id"
	// LabelKeyEnvironmentID is the label key for environment ID
	LabelKeyEnvironmentID = "unkey.com/environment.id"
	// LabelKeyVersion is the label key for version
	LabelKeyVersion = "app.kubernetes.io/version"
	// LabelKeyManagedBy is the label key for managed-by
	LabelKeyManagedBy = "app.kubernetes.io/managed-by"
	// LabelKeyComponent is the label key for component
	LabelKeyComponent = "app.kubernetes.io/component"
)

// LabelBuilder provides a fluent interface for building Kubernetes labels.
//
// The builder pattern allows chaining multiple label setters and produces
// an immutable map of labels. Each method returns the builder for chaining.
// Empty string values are ignored to prevent creating invalid labels.
type LabelBuilder struct {
	labels map[string]string
}

// NewLabels creates a new label builder for constructing Kubernetes labels.
//
// Example:
//
//	labels := k8s.NewLabels().
//	    WorkspaceID("ws_123").
//	    DeploymentID("dep_456").
//	    ManagedByKrane().
//	    ToMap()
func NewLabels() *LabelBuilder {
	return &LabelBuilder{
		labels: make(map[string]string),
	}
}

// From creates a new label builder starting with existing labels.
//
// This creates a copy of the provided map to ensure immutability.
// The original map is not modified by subsequent builder operations.
//
// Example:
//
//	existing := map[string]string{"env": "production"}
//	labels := k8s.From(existing).
//	    Version("v1.2.3").
//	    ToMap()
func From(existing map[string]string) *LabelBuilder {
	labels := make(map[string]string, len(existing))
	for k, v := range existing {
		labels[k] = v
	}
	return &LabelBuilder{labels: labels}
}

// WorkspaceID sets the workspace ID label.
//
// The workspace ID label (unkey.com/workspace.id) identifies which workspace
// owns or is associated with the Kubernetes resource. This is used for
// multi-tenancy and resource filtering.
func (b *LabelBuilder) WorkspaceID(id string) *LabelBuilder {
	b.labels[LabelKeyWorkspaceID] = id
	return b
}

// DeploymentID sets the deployment ID label.
//
// The deployment ID label (unkey.com/deployment.id) uniquely identifies
// a specific deployment managed by Krane. This enables tracking and management
// of deployment-specific resources.
func (b *LabelBuilder) DeploymentID(id string) *LabelBuilder {
	b.labels[LabelKeyDeploymentID] = id
	return b
}

// SentinelID sets the sentinel ID label.
//
// The sentinel ID label (unkey.com/sentinel.id) identifies resources associated
// with a specific API sentinel instance. This is used for sentinel-specific
// resource management and filtering.
func (b *LabelBuilder) SentinelID(id string) *LabelBuilder {
	b.labels[LabelKeySentinelID] = id
	return b
}

// ProjectID sets the project ID label.
//
// The project ID label (unkey.com/project.id) groups resources belonging
// to the same project. Projects typically represent applications or services
// within a workspace.
func (b *LabelBuilder) ProjectID(id string) *LabelBuilder {
	b.labels[LabelKeyProjectID] = id
	return b
}

// EnvironmentID sets the environment ID label.
//
// The environment ID label (unkey.com/environment.id) identifies which
// environment (development, staging, production, etc.) the resource belongs to.
// This enables environment-specific resource management and filtering.
func (b *LabelBuilder) EnvironmentID(id string) *LabelBuilder {
	b.labels[LabelKeyEnvironmentID] = id
	return b
}

// Version sets the version label.
//
// The version label (app.kubernetes.io/version) follows the Kubernetes
// recommended label convention for indicating the version of the application.
// This should typically be a semantic version string like "v1.2.3".
func (b *LabelBuilder) Version(version string) *LabelBuilder {
	b.labels[LabelKeyVersion] = version
	return b
}

// ManagedByKrane marks a resource as managed by Krane.
//
// The managed-by label (app.kubernetes.io/managed-by) follows the Kubernetes
// recommended label convention. Resources with this label set to "krane" are
// under Krane's management and should not be manually modified.
func (b *LabelBuilder) ManagedByKrane() *LabelBuilder {
	b.labels[LabelKeyManagedBy] = "krane"
	return b
}

// Custom adds a custom label with any key-value pair.
//
// Use this for labels that don't have dedicated builder methods.
// The key and value are used as-is without any prefix or validation.
func (b *LabelBuilder) Custom(key, value string) *LabelBuilder {
	if key != "" && value != "" {
		b.labels[key] = value
	}
	return b
}

// Merge adds all labels from the provided map to the builder.
//
// This is the escape hatch for merging existing label maps.
// Existing labels with the same keys will be overwritten.
//
// Example:
//
//	existingLabels := map[string]string{"env": "production", "tier": "frontend"}
//	labels := k8s.NewLabels().
//	    WorkspaceID("ws_123").
//	    Merge(existingLabels).
//	    Version("v1.2.3").
//	    ToMap()
func (b *LabelBuilder) Merge(labels map[string]string) *LabelBuilder {
	for k, v := range labels {
		if k != "" && v != "" {
			b.labels[k] = v
		}
	}
	return b
}

// ToMap returns the built label map.
//
// This returns a copy of the internal map to ensure immutability.
// The builder can continue to be used after calling ToMap.
func (b *LabelBuilder) ToMap() map[string]string {
	result := make(map[string]string, len(b.labels))
	for k, v := range b.labels {
		result[k] = v
	}
	return result
}

func (b *LabelBuilder) ToSelector() labels.Selector {
	return labels.SelectorFromSet(b.labels)
}

// GetWorkspaceID extracts the workspace ID from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetWorkspaceID(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeyWorkspaceID]
	return value, ok
}

// GetDeploymentID extracts the deployment ID from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetDeploymentID(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeyDeploymentID]
	return value, ok
}

// GetSentinelID extracts the sentinel ID from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetSentinelID(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeySentinelID]
	return value, ok
}

// GetProjectID extracts the project ID from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetProjectID(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeyProjectID]
	return value, ok
}

// GetEnvironmentID extracts the environment ID from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetEnvironmentID(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeyEnvironmentID]
	return value, ok
}

// GetVersion extracts the version from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetVersion(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeyVersion]
	return value, ok
}

// GetManagedBy extracts the managed-by value from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetManagedBy(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeyManagedBy]
	return value, ok
}

// GetComponent extracts the component from a label map.
// Returns the value and true if present, or empty string and false if not.
func GetComponent(labels map[string]string) (string, bool) {
	value, ok := labels[LabelKeyComponent]
	return value, ok
}

// IsManagedByKrane checks if the resource is managed by Krane.
// Returns true only if the managed-by label exists and is set to "krane".
func IsManagedByKrane(labels map[string]string) bool {
	value, ok := labels[LabelKeyManagedBy]
	return ok && value == "krane"
}

// GetLabel is a generic function to get any label value by its key.
// Returns the value and a boolean indicating if the label was found.
func GetLabel(labels map[string]string, key string) (string, bool) {
	value, exists := labels[key]
	return value, exists
}
