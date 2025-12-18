package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SentinelList contains a list of Sentinel resources.
//
// This type implements the standard Kubernetes list pattern and is used
// by the Kubernetes API server when returning collections of Sentinel
// resources through list operations and watch streams.
type SentinelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// Items is the list of Sentinel resources contained in this list.
	Items []Sentinel `json:"items"`
}

// Sentinel represents a sentinel deployment for monitoring, routing, and security.
//
// This custom resource defines the desired state for sentinel components
// which provide edge functionality like traffic routing, request monitoring,
// and security enforcement for Unkey deployments. Sentinels operate as
// independent services alongside application workloads.
//
// The controller ensures that the actual Kubernetes resources (Deployments,
// Services) match the desired state specified in this resource.
type Sentinel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// Spec defines the desired state of the sentinel deployment.
	// All fields in Spec are required and immutable after creation.
	// Changes to Spec trigger reconciliation to update the underlying resources.
	Spec SentinelSpec `json:"spec"`

	// Status defines the observed state of the sentinel.
	// This field is managed by the controller and should not be modified by users.
	// Status reflects the current operational state including deployment readiness
	// and any conditions that indicate problems or progress.
	Status SentinelStatus `json:"status,omitzero"`
}

// SentinelSpec defines the specification for a Sentinel deployment.
//
// This struct contains all configuration parameters needed to create and manage
// sentinel components. All fields are required because sentinels need complete
// information to function properly in the Unkey infrastructure.
type SentinelSpec struct {
	// WorkspaceID identifies the workspace this sentinel belongs to.
	// This is used for multi-tenant isolation and billing purposes.
	WorkspaceID string `json:"workspaceId"`

	// ProjectID identifies the project within the workspace.
	// Projects group related sentinels and applications together.
	ProjectID string `json:"projectId"`

	// EnvironmentID identifies the deployment environment (e.g., staging, production).
	// This enables different configurations and policies per environment.
	EnvironmentID string `json:"environmentId"`

	// SentinelID is the unique identifier for this sentinel instance.
	// This ID is globally unique within the workspace and used for
	// tracking and referencing the sentinel across systems.
	SentinelID string `json:"sentinelId"`

	// Image specifies the container image to run for the sentinel pods.
	// Must be a fully qualified container image name including the registry.
	Image string `json:"image"`

	// Replicas specifies the number of sentinel pods to run.
	// Minimum value is 1 for high availability. The controller ensures
	// this many pods are healthy and ready before marking the resource available.
	Replicas int32 `json:"replicas"`

	// CpuMillicores specifies the CPU allocation for each sentinel pod.
	// Value is in millicores (1000 millicores = 1 CPU core).
	// This affects scheduling and performance characteristics.
	CpuMillicores int64 `json:"cpuMillicores"`

	// MemoryMib specifies the memory allocation for each sentinel pod.
	// Value is in mebibytes (MiB). The sentinel must fit within this limit.
	MemoryMib int64 `json:"memoryMib"`
}

// Hash returns a hash of the sentinel spec used to quickly check for changes.
func (s SentinelSpec) Hash() string {
	return fmt.Sprintf("%s%s%s%s%s%d%d%d",
		s.WorkspaceID,
		s.ProjectID,
		s.EnvironmentID,
		s.SentinelID,
		s.Image,
		s.Replicas,
		s.CpuMillicores,
		s.MemoryMib,
	)
}

// SentinelStatus defines the observed state of a Sentinel resource.
//
// This struct contains runtime information about the sentinel's current
// operational state. It is managed entirely by the controller and provides
// visibility into the health and progress of sentinel deployments.
//
// Status conditions follow Kubernetes conventions with types, statuses,
// reasons, and messages that indicate specific aspects of the resource state.
type SentinelStatus struct {
	// Conditions represent the current state of the Sentinel resource.
	// Each condition has a unique type and reflects the status of a specific aspect.
	//
	// Standard condition types include:
	// - "Available": the sentinel deployment is fully functional and serving traffic
	// - "Progressing": the sentinel is being created, updated, or scaled
	// - "Degraded": the sentinel failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown:
	// - True: the condition is in the desired state
	// - False: the condition is not in the desired state (problem detected)
	// - Unknown: the state cannot be determined (typically during startup)
	//
	// Conditions include transition timestamps and detailed messages for debugging.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
