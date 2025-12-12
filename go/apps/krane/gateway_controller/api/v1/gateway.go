package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type GatewayList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gateway `json:"items"`
}

type Gateway struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              GatewaySpec `json:"spec"`
}

type GatewaySpec struct {
	WorkspaceId   string `json:"workspaceId"`
	ProjectId     string `json:"projectId"`
	EnvironmentId string `json:"environmentId"`
	GatewayId     string `json:"gatewayId"`
	Image         string `json:"image"`
	Replicas      int32  `json:"replicas"`
	CpuMillicores int64  `json:"cpuMillicores"`
	MemoryMib     int64  `json:"memoryMib"`
}
