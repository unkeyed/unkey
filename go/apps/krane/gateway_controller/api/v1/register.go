package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	GroupName    = "kubernetes.unkey.com"
	GroupVersion = "v1"
	Plural       = "gateways"
)

var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Gateway{},     // nolint:exhaustruct
		&GatewayList{}, // nolint:exhaustruct
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
