package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DeploymentList contains a list of Deployment resources.
type DeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// Items is the list of Deployment resources.
	Items []Deployment `json:"items"`
}

// Deployment represents a Deployment Deployment for monitoring and routing.
//
// This custom resource defines the desired state for Deployment components
// which provide edge functionality like traffic routing, monitoring, and
// security for Unkey Deployments.
type Deployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	// Spec defines the desired state of the Deployment.
	Spec DeploymentSpec `json:"spec"`
	// Status defines the observed state of the Deployment.
	Status DeploymentStatus `json:"status,omitzero"`
}

// DeploymentSpec defines the specification for a Deployment Deployment.
//
// This struct contains all parameters needed to create and manage
// Deployment components that provide edge functionality.
type DeploymentSpec struct {
	// WorkspaceID identifies the workspace this Deployment belongs to.
	WorkspaceID string `json:"workspaceId"`
	// ProjectID identifies the project within the workspace.
	ProjectID string `json:"projectId"`
	// EnvironmentID identifies the environment (e.g., staging, production).
	EnvironmentID string `json:"environmentId"`
	// DeploymentID is the unique identifier for this Deployment instance.
	DeploymentID string `json:"deploymentId"`
	// Replicas specifies the number of Deployment replicas to run.
	Replicas int32 `json:"replicas"`
	// Image specifies the container image for the Deployment.
	Image string `json:"image"`
	// CpuMillicores specifies the CPU request in millicores.
	CpuMillicores int64 `json:"cpuMillicores"`
	// MemoryMib specifies the memory request in mebibytes.
	MemoryMib int64 `json:"memoryMib"`
}

// DeploymentStatus defines the observed state of Deployment.
//
// This struct contains the current status information about the Deployment
// resource, including conditions that reflect its operational state.
type DeploymentStatus struct {
	// Conditions represent the current state of the Deployment resource.
	// Each condition has a unique type and reflects the status of a specific aspect.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
