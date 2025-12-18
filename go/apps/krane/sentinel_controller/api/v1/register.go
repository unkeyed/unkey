// Package v1 contains the Kubernetes API types for Sentinel resources.
//
// This package defines the Custom Resource Definition (CRD) types used by
// the sentinel controller to manage sentinel deployments. The types follow
// Kubernetes API conventions and support standard operations like create,
// read, update, delete (CRUD) and watch.
//
// The API version "kubernetes.unkey.com/v1" represents the first stable
// version of the Sentinel CRD and is backward compatible within the v1
// series.
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// GroupName is the API group name for Sentinel resources.
	GroupName = "kubernetes.unkey.com"
	// GroupVersion is the API version for this package.
	GroupVersion = "v1"
	// Plural is the plural name for Sentinel resources in the API.
	Plural = "sentinels"
)

// SchemeGroupVersion is the group version used to register these objects.
var SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}

var (
	// SchemeBuilder builds a new runtime.Scheme with the Sentinel types.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme adds all types in this package to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// addKnownTypes registers the Sentinel API types with the provided scheme.
//
// This function is called during controller initialization to ensure the
// Kubernetes client can serialize and deserialize Sentinel resources.
// It also adds the standard metav1 types for this group version.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Sentinel{},     // nolint:exhaustruct
		&SentinelList{}, // nolint:exhaustruct
	)

	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
