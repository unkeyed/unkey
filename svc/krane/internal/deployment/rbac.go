package deployment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ensureDeploymentRBAC creates a ServiceAccount, Role, and RoleBinding for the
// deployment. The Role only allows reading the specific deployment secret.
// Returns the ServiceAccount name.
func (c *Controller) ensureDeploymentRBAC(ctx context.Context, namespace, deploymentID, secretName string) (string, error) {
	saName := fmt.Sprintf("deploy-%s", deploymentID)
	roleName := fmt.Sprintf("deploy-%s-secret-reader", deploymentID)
	commonLabels := labels.New().DeploymentID(deploymentID).ManagedByKrane()

	// ServiceAccount
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
			Labels:    commonLabels,
		},
		AutomountServiceAccountToken: ptr.P(false),
	}
	if err := serverSideApply(ctx, c, "serviceaccounts", namespace, saName, sa); err != nil {
		return "", fmt.Errorf("failed to apply service account: %w", err)
	}

	// Role: can only get this one specific secret
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
			Labels:    commonLabels,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups:       []string{""},
			Resources:       []string{"secrets"},
			ResourceNames:   []string{secretName},
			Verbs:           []string{"get"},
			NonResourceURLs: nil,
		}},
	}
	if err := serverSideApplyRBAC(ctx, c, "roles", namespace, roleName, role); err != nil {
		return "", fmt.Errorf("failed to apply role: %w", err)
	}

	// RoleBinding
	rb := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
			Labels:    commonLabels,
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
	if err := serverSideApplyRBAC(ctx, c, "rolebindings", namespace, roleName, rb); err != nil {
		return "", fmt.Errorf("failed to apply role binding: %w", err)
	}

	return saName, nil
}

func serverSideApply(ctx context.Context, c *Controller, resource, namespace, name string, obj any) error {
	patch, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = c.clientSet.CoreV1().RESTClient().Patch(types.ApplyPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Param("fieldManager", fieldManagerKrane).
		Body(patch).
		Do(ctx).
		Get()
	return err
}

func serverSideApplyRBAC(ctx context.Context, c *Controller, resource, namespace, name string, obj any) error {
	patch, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	_, err = c.clientSet.RbacV1().RESTClient().Patch(types.ApplyPatchType).
		Namespace(namespace).
		Resource(resource).
		Name(name).
		Param("fieldManager", fieldManagerKrane).
		Body(patch).
		Do(ctx).
		Get()
	return err
}

// cleanupDeploymentResources removes the Secret, ServiceAccount, Role, and
// RoleBinding associated with a deployment. Ignores NotFound errors.
func (c *Controller) cleanupDeploymentResources(ctx context.Context, namespace, deploymentID string) {
	secretName := deploymentSecretName(deploymentID)
	saName := fmt.Sprintf("deploy-%s", deploymentID)
	roleName := fmt.Sprintf("deploy-%s-secret-reader", deploymentID)

	for _, fn := range []func() error{
		func() error {
			return c.clientSet.RbacV1().RoleBindings(namespace).Delete(ctx, roleName, metav1.DeleteOptions{})
		},
		func() error {
			return c.clientSet.RbacV1().Roles(namespace).Delete(ctx, roleName, metav1.DeleteOptions{})
		},
		func() error {
			return c.clientSet.CoreV1().Secrets(namespace).Delete(ctx, secretName, metav1.DeleteOptions{})
		},
		func() error {
			return c.clientSet.CoreV1().ServiceAccounts(namespace).Delete(ctx, saName, metav1.DeleteOptions{})
		},
	} {
		if err := fn(); err != nil && !apierrors.IsNotFound(err) {
			logger.Error("failed to cleanup deployment resource",
				"namespace", namespace,
				"deployment_id", deploymentID,
				"error", err,
			)
		}
	}
}
