package deploymentcontroller

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	v1 "github.com/unkeyed/unkey/go/apps/krane/deployment_controller/api/v1"
	"github.com/unkeyed/unkey/go/apps/krane/pkg/k8s"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ApplyDeployment creates or updates an UnkeyDeployment CRD based on the provided request.
// The controller-runtime reconciler will handle the actual Kubernetes Deployment creation.
func (c *DeploymentController) ApplyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) error {

	err := assert.All(
		assert.NotEmpty(req.GetWorkspaceId(), "Workspace ID is required"),
		assert.NotEmpty(req.GetProjectId(), "Project ID is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "Environment ID is required"),
		assert.NotEmpty(req.GetDeploymentId(), "Deployment ID is required"),
		assert.NotEmpty(req.GetNamespace(), "Namespace is required"),
		assert.NotEmpty(req.GetImage(), "Image is required"),
		assert.Greater(req.GetCpuMillicores(), uint32(0), "CPU millicores must be greater than 0"),
		assert.Greater(req.GetMemorySizeMib(), uint32(0), "Memory size in MiB must be greater than 0"),
		assert.Greater(req.GetReplicas(), uint32(0), "Replicas must be greater than 0"),
	)
	if err != nil {
		return err
	}

	// Ensure the namespace exists
	if err := c.ensureNamespaceExists(ctx, req.GetNamespace()); err != nil {
		return err
	}

	// Generate CRD name using the hash-based approach
	crdName := generateCRDName(req.GetDeploymentId())

	// Define labels for resource selection
	usedLabels := k8s.NewLabels().
		WorkspaceID(req.GetWorkspaceId()).
		ProjectID(req.GetProjectId()).
		EnvironmentID(req.GetEnvironmentId()).
		DeploymentID(req.GetDeploymentId()).
		ManagedByKrane().
		ToMap()

	obj := &v1.UnkeyDeployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "UnkeyDeployment",
			APIVersion: fmt.Sprintf("%s/%s", v1.GroupName, v1.GroupVersion),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.GetNamespace(),
			Name:      crdName,
			Labels:    usedLabels,
		},
		Spec: v1.UnkeyDeploymentSpec{
			WorkspaceId:   req.GetWorkspaceId(),
			ProjectId:     req.GetProjectId(),
			EnvironmentId: req.GetEnvironmentId(),
			DeploymentId:  req.GetDeploymentId(),
			Image:         req.GetImage(),
			Replicas:      int32(req.GetReplicas()),
			CpuMillicores: int64(req.GetCpuMillicores()),
			MemoryMib:     int64(req.GetMemorySizeMib()),
		},
	}

	c.logger.Info("Applying UnkeyDeployment", "name", crdName, "namespace", req.GetNamespace())

	existing := v1.UnkeyDeployment{} // nolint:exhaustruct
	err = c.manager.GetClient().Get(ctx, types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, &existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = c.manager.GetClient().Create(ctx, obj)
			if err != nil {
				c.logger.Error("failed to create UnkeyDeployment", "deployment_id", req.GetDeploymentId(), "error", err)
				return err
			}
			c.logger.Info("created UnkeyDeployment",
				"deployment_id", req.GetDeploymentId(),
				"name", crdName,
			)
			return nil
		}
		c.logger.Error("failed to get existing UnkeyDeployment", "deployment_id", req.GetDeploymentId(), "error", err)
		return err
	}

	// Update existing
	// nolint:staticcheck
	existing.ObjectMeta.Labels = usedLabels
	existing.Spec = obj.Spec

	err = c.manager.GetClient().Update(ctx, &existing)
	if err != nil {
		c.logger.Error("failed to update UnkeyDeployment", "deployment_id", req.GetDeploymentId(), "error", err)
		return err
	}
	c.logger.Info("updated UnkeyDeployment",
		"deployment_id", req.GetDeploymentId(),
		"name", crdName,
	)

	return nil
}

// generateCRDName creates a Kubernetes-compliant name for the UnkeyDeployment CRD.
// Format: dep-{lowercase_deployment_id_with_hyphens}-{first_8_of_hash}
func generateCRDName(deploymentID string) string {
	// Convert to lowercase and replace underscores with hyphens
	lower := strings.ToLower(deploymentID)
	lower = strings.ReplaceAll(lower, "_", "-")

	// Generate hash of original ID
	hash := sha256.Sum256([]byte(deploymentID))
	hashStr := fmt.Sprintf("%x", hash)[:8]

	// Combine with prefix and hash
	return fmt.Sprintf("%s-%s", lower, hashStr)

}
