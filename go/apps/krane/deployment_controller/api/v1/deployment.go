package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type UnkeyDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UnkeyDeployment `json:"items"`
}

type UnkeyDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              UnkeyDeploymentSpec `json:"spec"`
}

type UnkeyDeploymentSpec struct {
	WorkspaceId   string `json:"workspaceId"`
	ProjectId     string `json:"projectId"`
	EnvironmentId string `json:"environmentId"`
	DeploymentId  string `json:"deploymentId"`
	Image         string `json:"image"`
	Replicas      int32  `json:"replicas"`
	CpuMillicores int64  `json:"cpuMillicores"`
	MemoryMib     int64  `json:"memoryMib"`
}
