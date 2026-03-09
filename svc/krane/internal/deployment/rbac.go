package deployment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// ensureDeploymentRBAC creates a ServiceAccount, Role, and RoleBinding for the
// deployment. The Role only allows reading the specific deployment secret.
// All resources are owned by the ReplicaSet via ownerRef for automatic GC.
func (c *Controller) ensureDeploymentRBAC(ctx context.Context, namespace, deploymentID, secretName string, ownerRef metav1.OwnerReference) error {
	prefix := deploymentResourcePrefix(deploymentID)
	saName := prefix
	roleName := prefix + "-secret-reader"
	commonLabels := labels.New().DeploymentID(deploymentID).ManagedByKrane()

	// ServiceAccount
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

	// Role: can only get this one specific secret
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            roleName,
			Namespace:       namespace,
			Labels:          commonLabels,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups:       []string{""},
			Resources:       []string{"secrets"},
			ResourceNames:   []string{secretName},
			Verbs:           []string{"get"},
			NonResourceURLs: nil,
		}},
	}
	if err := serverSideApplyResource(ctx, c.clientSet.RbacV1().RESTClient(), "roles", namespace, roleName, role); err != nil {
		return fmt.Errorf("failed to apply role: %w", err)
	}

	// RoleBinding
	rb := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            roleName,
			Namespace:       namespace,
			Labels:          commonLabels,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      saName,
			Namespace: namespace,
			APIGroup:  "",
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     roleName,
		},
	}
	if err := serverSideApplyResource(ctx, c.clientSet.RbacV1().RESTClient(), "rolebindings", namespace, roleName, rb); err != nil {
		return fmt.Errorf("failed to apply role binding: %w", err)
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
