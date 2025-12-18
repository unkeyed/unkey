package reflector

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureNamespaceExists creates the namespace if it doesn't already exist.
//
// This helper function ensures that the target namespace for sentinel
// resources exists before attempting to create sentinel resources.
// If the namespace already exists, the function returns nil without error.
//
// This approach allows the reflector to automatically create required
// namespaces while gracefully handling cases where they already exist.
func (r *Reflector) ensureNamespaceExists(ctx context.Context, namespace string) error {
	err := r.client.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
