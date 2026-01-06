package labels

import (
	"fmt"
	"strings"
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

// WorkspaceID adds workspace ID label to the label set.
//
// This method sets the "unkey.com/workspace.id" label for identifying
// the workspace that owns the resource. Returns the same Labels instance
// for method chaining.
func (l Labels) WorkspaceID(id string) Labels {
	(l)["unkey.com/workspace.id"] = id
	return l
}

// DeploymentID adds deployment ID label to the label set.
//
// This method sets the "unkey.com/deployment.id" label for identifying
// the specific deployment that owns this resource. Returns the same
// Labels instance for method chaining.
func (l Labels) DeploymentID(id string) Labels {
	(l)["unkey.com/deployment.id"] = id
	return l
}

// ManagedByKrane adds the standard Kubernetes managed-by label.
//
// This method sets the "app.kubernetes.io/managed-by" label to "krane"
// to indicate that the resource is managed by the krane system.
// Returns the same Labels instance for method chaining.
func (l Labels) ManagedByKrane() Labels {
	(l)["app.kubernetes.io/managed-by"] = "krane"
	return l
}

// SentinelID adds sentinel ID label to the label set.
//
// This method sets the "unkey.com/sentinel.id" label for identifying
// the specific sentinel that owns this resource. Returns the same
// Labels instance for method chaining.
func (l Labels) SentinelID(id string) Labels {
	(l)["unkey.com/sentinel.id"] = id
	return l
}

// ComponentSentinel adds component label for sentinel resources.
//
// This method sets "app.kubernetes.io/component" label to "sentinel"
// to identify resource as a sentinel component. Returns the same
// Labels instance for method chaining.
func (l Labels) ComponentSentinel() Labels {
	(l)["app.kubernetes.io/component"] = "sentinel"
	return l
}

// ComponentDeployment adds component label for deployment resources.
//
// This method sets "app.kubernetes.io/component" label to "deployment"
// to identify resource as a deployment component. Returns the same
// Labels instance for method chaining.
func (l Labels) ComponentDeployment() Labels {
	(l)["app.kubernetes.io/component"] = "deployment"
	return l
}

// ProjectID adds project ID label to the label set.
//
// This method sets the "unkey.com/project.id" label for identifying
// the project that owns the resource. Returns the same Labels
// instance for method chaining.
func (l Labels) ProjectID(id string) Labels {
	(l)["unkey.com/project.id"] = id
	return l
}

// EnvironmentID adds environment ID label to the label set.
//
// This method sets the "unkey.com/environment.id" label to identify the
// environment that owns the resource. Returns the same Labels instance
// for method chaining.
func (l Labels) EnvironmentID(id string) Labels {
	(l)["unkey.com/environment.id"] = id
	return l
}

// Inject adds the inject label to the label set.
//
// This method sets the "unkey.com/inject" label to "true" to indicate
// that the resource should be injected. Returns the same Labels instance
// for method chaining.
func (l Labels) Inject() Labels {
	(l)["unkey.com/inject"] = "true"
	return l
}

// BuildID adds build ID label to the label set.
//
// This method sets the "unkey.com/build.id" label for identifying
// the build that produced the container image. Returns the same Labels
// instance for method chaining.
func (l Labels) BuildID(id string) Labels {
	(l)["unkey.com/build.id"] = id
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
	v, ok := l["unkey.com/sentinel.id"]
	return v, ok
}

// GetDeploymentID extracts deployment ID from Kubernetes label map.
//
// This helper function retrieves the "unkey.com/deployment.id" label from
// a Kubernetes resource's labels. Returns ID and a boolean indicating
// whether the label was found.
func GetDeploymentID(l map[string]string) (string, bool) {
	v, ok := l["unkey.com/deployment.id"]
	return v, ok
}
