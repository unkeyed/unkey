package reconciler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NamespaceSentinel  = "sentinel"
	NamespaceUntrusted = "untrusted"
)

// ensureNamespaceExists creates the namespace if it doesn't already exist.
// AlreadyExists errors are treated as success.
func (r *Reconciler) ensureNamespaceExists(ctx context.Context, namespace string) error {
	_, err := r.clientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
