package v1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies the receiver, writes into out. in must be non-nil.
//
// This method performs a deep copy of all Sentinel fields including metadata
// and specifications. It is used by controller-runtime and other Kubernetes
// components to ensure immutability of API objects during processing.
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

// DeepCopyObject returns a runtime.Object that is a deep copy of the receiver.
//
// This method is required by the runtime.Object interface and is used
// throughout the Kubernetes ecosystem to create immutable copies of API objects.
// The returned object can be safely modified without affecting the original.
func (in *Sentinel) DeepCopyObject() runtime.Object {
	out := &Sentinel{} // nolint:exhaustruct
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a runtime.Object that is a deep copy of the receiver.
//
// This method creates a deep copy of the SentinelList, including all
// contained Sentinel items. Each item in the list is individually deep copied
// to ensure complete isolation between the original and the copy.
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
