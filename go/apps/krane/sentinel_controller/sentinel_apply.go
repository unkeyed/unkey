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

func (c *SentinelController) ApplySentinel(ctx context.Context, req *ctrlv1.ApplySentinel) error {

	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetSentinelId(), "Sentinel ID is required"),
		assert.NotEmpty(req.GetNamespace(), "Namespace is required"),
		assert.NotEmpty(req.GetK8SCrdName(), "K8s CRD name is required"),
		assert.NotEmpty(req.GetImage(), "Image is required"),
		assert.Greater(req.GetCpuMillicores(), uint32(0), "CPU millicores must be greater than 0"),
		assert.Greater(req.GetMemorySizeMib(), uint32(0), "Memory size in MiB must be greater than 0"),
		assert.Greater(req.GetReplicas(), uint32(0), "Replicas must be greater than 0"),
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
			Image:         req.GetImage(),
			Replicas:      int32(req.GetReplicas()),
			CpuMillicores: int64(req.GetCpuMillicores()),
			MemoryMib:     int64(req.GetMemorySizeMib()),
		},
	}
	c.logger.Info("ctrlruntime.CreateOrUpdate", "obj", obj)

	existing := sentinelv1.Sentinel{} // nolint:exhaustruct
	err = c.manager.GetClient().Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.manager.GetClient().Create(ctx, obj)
		}
		return err
	}

	// nolint:staticcheck
	existing.ObjectMeta.Labels = usedLabels
	existing.Spec = obj.Spec

	c.logger.Info("updating sentinel", "existing", existing)
	err = c.manager.GetClient().Update(ctx, &existing)
	if err != nil {
		c.logger.Error("failed to apply sentinel", "sentinel_id", req.GetSentinelId(), "error", err)
		return err
	}
	c.logger.Info("applied sentinel",
		"sentinel_id", req.GetSentinelId(),
	)

	return nil
}
