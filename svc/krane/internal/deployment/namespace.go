package deployment

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/ptr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ensureNamespaceExists creates the namespace if it doesn't already exist,
// then locks down the default ServiceAccount so nothing in the namespace
// can access the K8s API unless explicitly granted via a per-deployment SA.
//
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

	if err := c.lockdownDefaultServiceAccount(ctx, namespace); err != nil {
		return fmt.Errorf("failed to lock down default service account: %w", err)
	}

	return nil
}

// lockdownDefaultServiceAccount disables automount on the default SA in the
// namespace. This ensures that any pod without an explicit SA assignment
// (which should never happen for krane-managed pods) gets no K8s API token.
func (c *Controller) lockdownDefaultServiceAccount(ctx context.Context, namespace string) error {
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: namespace,
		},
		AutomountServiceAccountToken: ptr.P(false),
	}
	return serverSideApply(ctx, c, "serviceaccounts", namespace, "default", sa)
}
