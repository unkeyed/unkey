package gatewaycontroller

import (
	"context"
	"fmt"

	gatewayv1 "github.com/unkeyed/unkey/go/apps/krane/gateway_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (c *GatewayController) ApplyGateway(ctx context.Context, req *ctrlv1.ApplyGateway) error {

	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetGatewayId(), "Gateway ID is required"),
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
		GatewayID(req.GetGatewayId()).
		ManagedByKrane().
		ToMap()

	obj := &gatewayv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Gateway",
			APIVersion: fmt.Sprintf("%s/%s", gatewayv1.GroupName, gatewayv1.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.GetNamespace(),
			Name:      req.GetK8SCrdName(),
			Labels:    usedLabels,
		},
		Spec: gatewayv1.GatewaySpec{
			WorkspaceId:   req.GetWorkspaceId(),
			ProjectId:     req.GetProjectId(),
			EnvironmentId: req.GetEnvironmentId(),
			GatewayId:     req.GetGatewayId(),
			Image:         req.GetImage(),
			Replicas:      int32(req.GetReplicas()),
			CpuMillicores: int64(req.GetCpuMillicores()),
			MemoryMib:     int64(req.GetMemorySizeMib()),
		},
	}
	c.logger.Info("ctrlruntime.CreateOrUpdate", obj, obj)

	existing := gatewayv1.Gateway{}
	err = c.mgr.GetClient().Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, &existing)
	if err != nil {

		if apierrors.IsNotFound(err) {
			err = c.mgr.GetClient().Create(ctx, obj)
			if err != nil {
				c.logger.Error("failed to apply gateway", "gateway_id", req.GetGatewayId(), "error", err)
				return err
			}
		}
		c.logger.Error("failed to get existing gateway", "gateway_id", req.GetGatewayId(), "error", err)
		return err
	}

	existing.ObjectMeta.Labels = usedLabels
	existing.Spec = obj.Spec

	err = c.mgr.GetClient().Update(ctx, &existing)
	if err != nil {
		c.logger.Error("failed to apply gateway", "gateway_id", req.GetGatewayId(), "error", err)
		return err
	}
	c.logger.Info("applied gateway",
		"gateway_id", req.GetGatewayId(),
	)

	return nil
}
