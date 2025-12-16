package v1

import "k8s.io/apimachinery/pkg/runtime"

func (in *Deployment) DeepCopyInto(out *Deployment) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = DeploymentSpec{
		WorkspaceID:   in.Spec.WorkspaceID,
		ProjectID:     in.Spec.ProjectID,
		EnvironmentID: in.Spec.EnvironmentID,
		DeploymentID:  in.Spec.DeploymentID,
		Image:         in.Spec.Image,
		Replicas:      in.Spec.Replicas,
		CpuMillicores: in.Spec.CpuMillicores,
		MemoryMib:     in.Spec.MemoryMib,
	}
}

func (in *Deployment) DeepCopyObject() runtime.Object {
	out := &Deployment{} // nolint:exhaustruct
	in.DeepCopyInto(out)
	return out
}

func (in *DeploymentList) DeepCopyObject() runtime.Object {
	out := &DeploymentList{
		TypeMeta: in.TypeMeta,
		ListMeta: in.ListMeta,
		Items:    nil,
	}
	if in.Items != nil {
		out.Items = make([]Deployment, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return out
}
