package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type SentinelList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Sentinel `json:"items"`
}

type Sentinel struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SentinelSpec `json:"spec"`
}

type SentinelSpec struct {
	WorkspaceId   string `json:"workspaceId"`
	ProjectId     string `json:"projectId"`
	EnvironmentId string `json:"environmentId"`
	SentinelId    string `json:"sentinelId"`
	Image         string `json:"image"`
	Replicas      int32  `json:"replicas"`
	CpuMillicores int64  `json:"cpuMillicores"`
	MemoryMib     int64  `json:"memoryMib"`
}
