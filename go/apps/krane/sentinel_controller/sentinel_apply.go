package sentinelcontroller

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	sentinelv1 "github.com/unkeyed/unkey/go/apps/krane/sentinel_controller/api/v1"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ApplySentinel creates or updates a Sentinel CRD based on the provided request.
//
// This method validates the request, ensures the namespace exists, and creates
// or updates the Sentinel custom resource. The controller-runtime reconciler
// will handle the actual Kubernetes resource creation based on this CRD.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: Sentinel application request with specifications
//
// Returns an error if validation fails, namespace creation fails,
// or CRD creation/update encounters problems.
func (c *SentinelController) ApplySentinel(ctx context.Context, req *ctrlv1.ApplySentinel) error {

	c.logger.Info("Applying Sentinel",
		"req", req,
	)
	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetSentinelId(), "Sentinel ID is required"),
		assert.NotEmpty(req.GetNamespace(), "Namespace is required"),
		assert.NotEmpty(req.GetK8SCrdName(), "K8s CRD name is required"),
		assert.NotEmpty(req.GetHash(), "Hash is required"),
	)
	if err != nil {
		return err
	}

	if err := c.ensureNamespaceExists(ctx, req.GetNamespace()); err != nil {
		return err
	}

	// Define labels for resource selection
	usedLabels := k8s.NewLabels().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		EnvironmentID(req.GetEnvironmentId()).
		SentinelID(req.GetSentinelId()).
		ManagedByKrane().
		ToMap()

	c.logger.Info("creating sentinel", "req", req)
	obj := &sentinelv1.Sentinel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Sentinel",
			APIVersion: fmt.Sprintf("%s/%s", sentinelv1.GroupName, sentinelv1.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.GetNamespace(),
			Name:      req.GetK8SCrdName(),
			Labels:    usedLabels,
		},
		Spec: sentinelv1.SentinelSpec{
			WorkspaceID:   req.GetWorkspaceId(),
			ProjectID:     req.GetProjectId(),
			EnvironmentID: req.GetEnvironmentId(),
			SentinelID:    req.GetSentinelId(),
			Hash:          req.GetHash(),
		},
		Status: sentinelv1.SentinelStatus{
			Conditions: []metav1.Condition{},
		},
	}

	existing := sentinelv1.Sentinel{} // nolint:exhaustruct
	err = c.client.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.client.Create(ctx, obj)
		}
		return err
	}

	existing.Spec = obj.Spec

	err = c.client.Update(ctx, &existing)
	if err != nil {
		c.logger.Error("failed to apply sentinel", "sentinel_id", req.GetSentinelId(), "error", err)
		return err
	}

	return nil
}
