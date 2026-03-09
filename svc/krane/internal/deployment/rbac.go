package deployment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// ensureDeploymentServiceAccount creates a ServiceAccount for the deployment.
// The SA is referenced by podSpec.ServiceAccountName so the pod doesn't use the
// namespace default. automountServiceAccountToken is false — no API access is needed.
// The SA is owned by the ReplicaSet via ownerRef for automatic GC.
func (c *Controller) ensureDeploymentServiceAccount(ctx context.Context, namespace, deploymentID string, ownerRef metav1.OwnerReference) error {
	saName := deploymentResourcePrefix(deploymentID)
	commonLabels := labels.New().DeploymentID(deploymentID).ManagedByKrane()

	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            saName,
			Namespace:       namespace,
			Labels:          commonLabels,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		AutomountServiceAccountToken: ptr.P(false),
	}
	if err := serverSideApplyResource(ctx, c.clientSet.CoreV1().RESTClient(), "serviceaccounts", namespace, saName, sa); err != nil {
		return fmt.Errorf("failed to apply service account: %w", err)
	}

	return nil
}

func serverSideApplyResource(ctx context.Context, restClient rest.Interface, resource, namespace, name string, obj any) error {
	patch, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = restClient.Patch(types.ApplyPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Param("fieldManager", fieldManagerKrane).
		Body(patch).
		Do(ctx).
		Get()
	return err
}
