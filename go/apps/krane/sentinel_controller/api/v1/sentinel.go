package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// SentinelList contains a list of Sentinel resources.
type SentinelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// Items is the list of Sentinel resources.
	Items []Sentinel `json:"items"`
}

// Sentinel represents a sentinel deployment for monitoring and routing.
//
// This custom resource defines the desired state for sentinel components
// which provide edge functionality like traffic routing, monitoring, and
// security for Unkey deployments.
type Sentinel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	// Spec defines the desired state of the sentinel.
	Spec SentinelSpec `json:"spec"`
	// Status defines the observed state of the sentinel.
	Status SentinelStatus `json:"status,omitzero"`
}

// SentinelSpec defines the specification for a Sentinel deployment.
//
// This struct contains all parameters needed to create and manage
// sentinel components that provide edge functionality.
type SentinelSpec struct {
	// WorkspaceID identifies the workspace this sentinel belongs to.
	WorkspaceID string `json:"workspaceId"`
	// ProjectID identifies the project within the workspace.
	ProjectID string `json:"projectId"`
	// EnvironmentID identifies the environment (e.g., staging, production).
	EnvironmentID string `json:"environmentId"`
	// SentinelID is the unique identifier for this sentinel instance.
	SentinelID string `json:"sentinelId"`

	Image         string `json:"image"`
	Replicas      int32  `json:"replicas"`
	CpuMillicores int64  `json:"cpuMillicores"`
	MemoryMib     int64  `json:"memoryMib"`
}

// SentinelStatus defines the observed state of Sentinel.
//
// This struct contains the current status information about the sentinel
// resource, including conditions that reflect its operational state.
type SentinelStatus struct {
	// Conditions represent the current state of the Sentinel resource.
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
