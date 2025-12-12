package v1

import "k8s.io/apimachinery/pkg/runtime"

func (in *Gateway) DeepCopyInto(out *Gateway) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = GatewaySpec{
		WorkspaceId:   in.Spec.WorkspaceId,
		ProjectId:     in.Spec.ProjectId,
		EnvironmentId: in.Spec.EnvironmentId,
		GatewayId:     in.Spec.GatewayId,
		Image:         in.Spec.Image,
		Replicas:      in.Spec.Replicas,
		CpuMillicores: in.Spec.CpuMillicores,
		MemoryMib:     in.Spec.MemoryMib,
	}
}

func (in *Gateway) DeepCopyObject() runtime.Object {
	out := &Gateway{} // nolint:exhaustruct
	in.DeepCopyInto(out)
	return out
}

func (in *GatewayList) DeepCopyObject() runtime.Object {
	out := &GatewayList{
		TypeMeta: in.TypeMeta,
		ListMeta: in.ListMeta,
		Items:    nil,
	}
	if in.Items != nil {
		out.Items = make([]Gateway, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return out
}
