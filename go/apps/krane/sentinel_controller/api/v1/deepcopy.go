package v1

import "k8s.io/apimachinery/pkg/runtime"

func (in *Sentinel) DeepCopyInto(out *Sentinel) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = SentinelSpec{
		WorkspaceID:   in.Spec.WorkspaceID,
		ProjectID:     in.Spec.ProjectID,
		EnvironmentID: in.Spec.EnvironmentID,
		SentinelID:    in.Spec.SentinelID,
		Image:         in.Spec.Image,
		Replicas:      in.Spec.Replicas,
		CpuMillicores: in.Spec.CpuMillicores,
		MemoryMib:     in.Spec.MemoryMib,
	}
}

func (in *Sentinel) DeepCopyObject() runtime.Object {
	out := &Sentinel{} // nolint:exhaustruct
	in.DeepCopyInto(out)
	return out
}

func (in *SentinelList) DeepCopyObject() runtime.Object {
	out := &SentinelList{
		TypeMeta: in.TypeMeta,
		ListMeta: in.ListMeta,
		Items:    nil,
	}
	if in.Items != nil {
		out.Items = make([]Sentinel, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return out
}
