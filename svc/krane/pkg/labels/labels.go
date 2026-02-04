package labels

import (
	"fmt"
	"strings"
)

// Label key constants for krane resources.
// These are the single source of truth for label keys used across the codebase.
const (
	LabelKeyWorkspaceID           = "unkey.com/workspace.id"
	LabelKeyProjectID             = "unkey.com/project.id"
	LabelKeyEnvironmentID         = "unkey.com/environment.id"
	LabelKeyDeploymentID          = "unkey.com/deployment.id"
	LabelKeyBuildID               = "unkey.com/build.id"
	LabelKeySentinelID            = "unkey.com/sentinel.id"
	LabelKeyNetworkPolicyID       = "unkey.com/networkpolicy.id"
	LabelKeyCiliumNetworkPolicyID = "unkey.com/cilium.network-policy.id"
	LabelKeyInject                = "unkey.com/inject"
	LabelKeyManagedBy             = "app.kubernetes.io/managed-by"
	LabelKeyComponent             = "app.kubernetes.io/component"
	LabelKeyNamespace             = "io.kubernetes.pod.namespace"
)

// Labels represents a map of Kubernetes labels for krane resources.
//
// This type provides fluent methods for building standardized label sets
// that follow krane's labeling conventions. It implements method chaining
// for easy label construction.
type Labels map[string]string

// New creates an empty Labels map for building label sets.
//
// This function returns a new Labels instance that can be populated
// using the fluent With methods for adding specific labels.
// Returns an empty label map ready for method chaining.
func New() Labels {
	return Labels{}
}

// Namespace adds namespace label to the label set.
//
// This method sets the "io.kubernetes.pod.namespace" label to specify
// the Kubernetes namespace of the resource. Returns the same Labels
// instance for method chaining.
func (l Labels) Namespace(namespace string) Labels {
	l[LabelKeyNamespace] = namespace
	return l
}

// WorkspaceID adds workspace ID label to the label set.
//
// This method sets the "unkey.com/workspace.id" label for identifying
// the workspace that owns the resource. Returns the same Labels instance
// for method chaining.
func (l Labels) WorkspaceID(id string) Labels {
	l[LabelKeyWorkspaceID] = id
	return l
}

// DeploymentID adds deployment ID label to the label set.
//
// This method sets the "unkey.com/deployment.id" label for identifying
// the specific deployment that owns this resource. Returns the same
// Labels instance for method chaining.
func (l Labels) DeploymentID(id string) Labels {
	l[LabelKeyDeploymentID] = id
	return l
}

// ManagedByKrane adds the standard Kubernetes managed-by label.
//
// This method sets the "app.kubernetes.io/managed-by" label to "krane"
// to indicate that the resource is managed by the krane system.
// Returns the same Labels instance for method chaining.
func (l Labels) ManagedByKrane() Labels {
	l[LabelKeyManagedBy] = "krane"
	return l
}

// SentinelID adds sentinel ID label to the label set.
//
// This method sets the "unkey.com/sentinel.id" label for identifying
// the specific sentinel that owns this resource. Returns the same
// Labels instance for method chaining.
func (l Labels) SentinelID(id string) Labels {
	l[LabelKeySentinelID] = id
	return l
}

// NetworkPolicyID adds network policy ID label to the label set.
//
// This method sets the "unkey.com/networkpolicy.id" label for identifying
// the specific sentinel that owns this resource. Returns the same
// Labels instance for method chaining.
func (l Labels) NetworkPolicyID(id string) Labels {
	l[LabelKeyNetworkPolicyID] = id
	return l
}

// ComponentSentinel adds component label for sentinel resources.
//
// This method sets "app.kubernetes.io/component" label to "sentinel"
// to identify resource as a sentinel component. Returns the same
// Labels instance for method chaining.
func (l Labels) ComponentSentinel() Labels {
	l[LabelKeyComponent] = "sentinel"
	return l
}

// ComponentDeployment adds component label for deployment resources.
//
// This method sets "app.kubernetes.io/component" label to "deployment"
// to identify resource as a deployment component. Returns the same
// Labels instance for method chaining.
func (l Labels) ComponentDeployment() Labels {
	l[LabelKeyComponent] = "deployment"
	return l
}

// ComponentCiliumNetworkPolicy adds component label for deployment resources.
//
// This method sets "app.kubernetes.io/component" label to "ciliumnetworkpolicy"
// to identify resource as a deployment component. Returns the same
// Labels instance for method chaining.
func (l Labels) ComponentCiliumNetworkPolicy() Labels {
	l[LabelKeyComponent] = "ciliumnetworkpolicy"
	return l
}

// ProjectID adds project ID label to the label set.
//
// This method sets the "unkey.com/project.id" label for identifying
// the project that owns the resource. Returns the same Labels
// instance for method chaining.
func (l Labels) ProjectID(id string) Labels {
	l[LabelKeyProjectID] = id
	return l
}

// EnvironmentID adds environment ID label to the label set.
//
// This method sets the "unkey.com/environment.id" label to identify the
// environment that owns the resource. Returns the same Labels instance
// for method chaining.
func (l Labels) EnvironmentID(id string) Labels {
	l[LabelKeyEnvironmentID] = id
	return l
}

// Inject adds the inject label to the label set.
//
// This method sets the "unkey.com/inject" label to "true" to indicate
// that the resource should be injected. Returns the same Labels instance
// for method chaining.
func (l Labels) Inject() Labels {
	l[LabelKeyInject] = "true"
	return l
}

// BuildID adds build ID label to the label set.
//
// This method sets the "unkey.com/build.id" label for identifying
// the build that produced the container image. Returns the same Labels
// instance for method chaining.
func (l Labels) BuildID(id string) Labels {
	l[LabelKeyBuildID] = id
	return l
}

// ToString converts Labels map to Kubernetes label selector string.
//
// This method formats the labels as a comma-separated list of key=value pairs
// suitable for use with Kubernetes API selectors. The output format follows
// Kubernetes label selector conventions.
//
// Returns an empty string for empty label maps.
func (l Labels) ToString() string {
	s := ""
	for k, v := range l {
		s += fmt.Sprintf("%s=%s,", k, v)
	}
	return strings.TrimSuffix(s, ",")
}

// GetSentinelID extracts sentinel ID from Kubernetes label map.
//
// This helper function retrieves the "unkey.com/sentinel.id" label from
// a Kubernetes resource's labels. Returns the ID and a boolean indicating
// whether the label was found.
func GetSentinelID(l map[string]string) (string, bool) {
	v, ok := l[LabelKeySentinelID]
	return v, ok
}

// GetDeploymentID extracts deployment ID from Kubernetes label map.
//
// This helper function retrieves the "unkey.com/deployment.id" label from
// a Kubernetes resource's labels. Returns ID and a boolean indicating
// whether the label was found.
func GetDeploymentID(l map[string]string) (string, bool) {
	v, ok := l[LabelKeyDeploymentID]
	return v, ok
}

// GetEnvironmentID extracts environment ID from Kubernetes label map.
//
// This helper function retrieves the "unkey.com/environment.id" label from
// a Kubernetes resource's labels. Returns ID and a boolean indicating
// whether the label was found.
func GetEnvironmentID(l map[string]string) (string, bool) {
	v, ok := l[LabelKeyEnvironmentID]
	return v, ok
}

// GetCiliumNetworkPolicyID extracts cilium network policy ID from Kubernetes label map.
//
// This helper function retrieves the "unkey.com/cilium.network-policy.id" label from
// a Kubernetes resource's labels. Returns ID and a boolean indicating whether the label was found.
func GetCiliumNetworkPolicyID(l map[string]string) (string, bool) {
	v, ok := l[LabelKeyCiliumNetworkPolicyID]
	return v, ok
}
