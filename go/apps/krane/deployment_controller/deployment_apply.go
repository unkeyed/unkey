package deploymentcontroller

import (
	"context"
	"fmt"

	deploymentv1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ApplyDeployment creates or updates a Deployment CRD based on the provided request.
//
// This method validates the request, ensures the namespace exists, and creates
// or updates the Deployment custom resource. The controller-runtime reconciler
// will handle the actual Kubernetes resource creation based on this CRD.
//
// Parameters:
//   - ctx: Context for the operation
//   - req: Deployment application request with specifications
//
// Returns an error if validation fails, namespace creation fails,
// or CRD creation/update encounters problems.
func (c *DeploymentController) ApplyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) error {

	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetDeploymentId(), "Deployment ID is required"),
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
		DeploymentID(req.GetDeploymentId()).
		ManagedByKrane().
		ToMap()

	c.logger.Info("creating deployment", "req", req)

	obj := &deploymentv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: fmt.Sprintf("%s/%s", deploymentv1.GroupName, deploymentv1.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.GetNamespace(),
			Name:      req.GetK8SCrdName(),
			Labels:    usedLabels,
		},
		Spec: deploymentv1.DeploymentSpec{
			WorkspaceID:   req.GetWorkspaceId(),
			ProjectID:     req.GetProjectId(),
			EnvironmentID: req.GetEnvironmentId(),
			DeploymentID:  req.GetDeploymentId(),
			Image:         req.GetImage(),
			Replicas:      int32(req.GetReplicas()),
			CpuMillicores: int64(req.GetCpuMillicores()),
			MemoryMib:     int64(req.GetMemorySizeMib()),
		},
	}
	c.logger.Info("ctrlruntime.CreateOrUpdate", "obj", obj)

	existing := deploymentv1.Deployment{} // nolint:exhaustruct
	err = c.client.Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.client.Create(ctx, obj)
		}
		return err
	}

	// nolint:staticcheck
	existing.ObjectMeta.Labels = usedLabels
	existing.Spec = obj.Spec

	c.logger.Info("updating deployment", "existing", existing)
	err = c.client.Update(ctx, &existing)
	if err != nil {
		c.logger.Error("failed to apply deployment", "deployment_id", req.GetDeploymentId(), "error", err)
		return err
	}
	c.logger.Info("applied deployment",
		"deployment_id", req.GetDeploymentId(),
	)

	return nil
}
