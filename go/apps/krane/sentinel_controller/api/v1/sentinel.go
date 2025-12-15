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
}

type SentinelSpec struct {
	WorkspaceID   string `json:"workspaceId"`
	ProjectID     string `json:"projectId"`
	EnvironmentID string `json:"environmentId"`
	SentinelID    string `json:"sentinelId"`
	Image         string `json:"image"`
	Replicas      int32  `json:"replicas"`
	CpuMillicores int64  `json:"cpuMillicores"`
	MemoryMib     int64  `json:"memoryMib"`
}
