package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type SentinelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Sentinel `json:"items"`
}

type Sentinel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              SentinelSpec `json:"spec"`

	Status SentinelStatus `json:"status,omitzero"`
}

type SentinelSpec struct {
	WorkspaceID   string `json:"workspaceId"`
	ProjectID     string `json:"projectId"`
	EnvironmentID string `json:"environmentId"`
	SentinelID    string `json:"sentinelId"`
	Replicas      int32  `json:"replicas"`
	Image         string `json:"image"`
	CpuMillicores int64  `json:"cpuMillicores"`
	MemoryMib     int64  `json:"memoryMib"`
}

// SentinelStatus defines the observed state of Sentinel.
type SentinelStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the Sentinel resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
