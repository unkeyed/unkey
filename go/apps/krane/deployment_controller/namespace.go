package deploymentcontroller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *DeploymentController) ensureNamespaceExists(ctx context.Context, namespace string) error {

	err := c.client.Create(ctx, &corev1.Namespace{

		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
