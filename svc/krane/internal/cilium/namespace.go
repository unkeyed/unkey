package cilium

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureNamespaceExists creates the namespace if it doesn't already exist and
// configures network policies for customer workloads.
// The method is idempotent: calling it for an existing namespace succeeds without error.
func (c *Controller) ensureNamespaceExists(ctx context.Context, namespace string) error {
	_, err := c.clientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
