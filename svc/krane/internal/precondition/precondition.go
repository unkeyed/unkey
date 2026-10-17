// Package precondition builds Kubernetes write preconditions.
package precondition

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Unchanged deletes object only if it is still the observed version, so a
// concurrent replacement or update is never deleted by mistake.
func Unchanged(object metav1.Object) metav1.DeleteOptions {
	uid := object.GetUID()
	version := object.GetResourceVersion()
	return metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid, ResourceVersion: &version}}
}
