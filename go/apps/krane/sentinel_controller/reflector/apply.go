package reflector

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

// applySentinel creates or updates a Sentinel CRD based on the provided request.
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
func (r *Reflector) applySentinel(ctx context.Context, req *ctrlv1.ApplySentinel) error {

	r.logger.Info("applying sentinel",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
		"sentinel_id", req.GetSentinelId(),
	)
	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetSentinelId(), "Sentinel ID is required"),
		assert.NotEmpty(req.GetK8SNamespace(), "Namespace is required"),
		assert.NotEmpty(req.GetK8SName(), "K8s CRD name is required"),
		assert.NotEmpty(req.GetImage(), "Image is required"),
		assert.Greater(req.GetCpuMillicores(), int64(0), "CPU millicores must be greater than 0"),
		assert.Greater(req.GetMemoryMib(), int64(0), "MemoryMib must be greater than 0"),
	)
	if err != nil {
		return err
	}

	if err := r.ensureNamespaceExists(ctx, req.GetK8SNamespace()); err != nil {
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

	obj := &sentinelv1.Sentinel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Sentinel",
			APIVersion: fmt.Sprintf("%s/%s", sentinelv1.GroupName, sentinelv1.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.GetK8SNamespace(),
			Name:      req.GetK8SName(),
			Labels:    usedLabels,
		},
		Spec: sentinelv1.SentinelSpec{
			WorkspaceID:   req.GetWorkspaceId(),
			ProjectID:     req.GetProjectId(),
			EnvironmentID: req.GetEnvironmentId(),
			SentinelID:    req.GetSentinelId(),
			Image:         req.GetImage(),
			Replicas:      req.GetReplicas(),
			CpuMillicores: req.GetCpuMillicores(),
			MemoryMib:     req.GetMemoryMib(),
		},
		Status: sentinelv1.SentinelStatus{
			Conditions: []metav1.Condition{},
		},
	}

	existing := sentinelv1.Sentinel{} // nolint:exhaustruct
	err = r.client.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.client.Create(ctx, obj)
		}
		return err
	}
	if existing.Spec.Hash() == obj.Spec.Hash() {
		// nothing to do, we're in sync
		return nil
	}

	existing.Spec = obj.Spec

	err = r.client.Update(ctx, &existing)
	if err != nil {
		r.logger.Error("failed to apply sentinel", "sentinel_id", req.GetSentinelId(), "error", err)
		return err
	}

	return nil
}
