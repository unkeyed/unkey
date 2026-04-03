package deployment

import (
	"context"
	"encoding/json"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// ensureVPAExists creates or updates a VerticalPodAutoscaler that right-sizes the
// deployment's ReplicaSet using the vertical autoscaling policy from the control plane.
// The VPA is owned by the ReplicaSet for automatic garbage collection.
//
// If the VPA CRD is not installed in the cluster, the error is logged and skipped
// to avoid breaking deployments on clusters without VPA support.
func (c *Controller) ensureVPAExists(ctx context.Context, req *ctrlv1.ApplyDeployment, rs *appsv1.ReplicaSet) error {
	vpaPolicy := req.GetVerticalAutoscaling()
	if vpaPolicy == nil {
		return nil
	}

	updateMode := vpaUpdateModeString(vpaPolicy.GetUpdateMode())
	controlledResources := vpaControlledResourcesList(vpaPolicy.GetControlledResources())
	controlledValues := vpaControlledValuesString(vpaPolicy.GetControlledValues())

	containerPolicy := map[string]any{
		"containerName":       "deployment",
		"controlledResources": controlledResources,
		"controlledValues":    controlledValues,
	}

	minAllowed := map[string]any{}
	maxAllowed := map[string]any{}

	if vpaPolicy.CpuMinMillicores != nil {
		minAllowed["cpu"] = fmt.Sprintf("%dm", vpaPolicy.GetCpuMinMillicores())
	}
	if vpaPolicy.CpuMaxMillicores != nil {
		maxAllowed["cpu"] = fmt.Sprintf("%dm", vpaPolicy.GetCpuMaxMillicores())
	}
	if vpaPolicy.MemoryMinMib != nil {
		minAllowed["memory"] = fmt.Sprintf("%dMi", vpaPolicy.GetMemoryMinMib())
	}
	if vpaPolicy.MemoryMaxMib != nil {
		maxAllowed["memory"] = fmt.Sprintf("%dMi", vpaPolicy.GetMemoryMaxMib())
	}

	if len(minAllowed) > 0 {
		containerPolicy["minAllowed"] = minAllowed
	}
	if len(maxAllowed) > 0 {
		containerPolicy["maxAllowed"] = maxAllowed
	}

	vpaObj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": vpaGroup + "/" + vpaVersion,
			"kind":       "VerticalPodAutoscaler",
			"metadata": map[string]any{
				"name":      req.GetK8SName(),
				"namespace": req.GetK8SNamespace(),
				"labels": labels.New().
					WorkspaceID(req.GetWorkspaceId()).
					ProjectID(req.GetProjectId()).
					AppID(req.GetAppId()).
					EnvironmentID(req.GetEnvironmentId()).
					DeploymentID(req.GetDeploymentId()).
					ManagedByKrane().
					ComponentDeployment(),
				"ownerReferences": []map[string]any{
					{
						"apiVersion":         "apps/v1",
						"kind":               "ReplicaSet",
						"name":               rs.Name,
						"uid":                string(rs.UID),
						"controller":         true,
						"blockOwnerDeletion": true,
					},
				},
			},
			"spec": map[string]any{
				"targetRef": map[string]any{
					"apiVersion": "apps/v1",
					"kind":       "ReplicaSet",
					"name":       req.GetK8SName(),
				},
				"updatePolicy": map[string]any{
					"updateMode": updateMode,
				},
				"resourcePolicy": map[string]any{
					"containerPolicies": []any{containerPolicy},
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group:    vpaGroup,
		Version:  vpaVersion,
		Resource: vpaResource,
	}

	patch, err := json.Marshal(vpaObj.Object)
	if err != nil {
		return fmt.Errorf("failed to marshal VPA: %w", err)
	}

	_, err = c.dynamicClient.Resource(gvr).Namespace(req.GetK8SNamespace()).Patch(
		ctx,
		req.GetK8SName(),
		types.ApplyPatchType,
		patch,
		metav1.PatchOptions{FieldManager: fieldManagerKrane},
	)
	if err != nil {
		logger.Warn("failed to apply VPA (CRD may not be installed)",
			"namespace", req.GetK8SNamespace(),
			"name", req.GetK8SName(),
			"error", err,
		)
		return fmt.Errorf("failed to apply VPA: %w", err)
	}

	return nil
}

// vpaUpdateModeString converts the proto enum to the VPA CRD string value.
func vpaUpdateModeString(mode ctrlv1.VerticalAutoscalingPolicy_UpdateMode) string {
	switch mode {
	case ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_UNSPECIFIED:
		return "Off"
	case ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_OFF:
		return "Off"
	case ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_INITIAL:
		return "Initial"
	case ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_RECREATE:
		return "Recreate"
	case ctrlv1.VerticalAutoscalingPolicy_UPDATE_MODE_IN_PLACE_OR_RECREATE:
		return "InPlaceOrRecreate"
	default:
		return "Off"
	}
}

// vpaControlledResourcesList converts the proto enum to the VPA CRD resource list.
func vpaControlledResourcesList(cr ctrlv1.VerticalAutoscalingPolicy_ControlledResources) []string {
	switch cr {
	case ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_RESOURCES_UNSPECIFIED:
		return []string{"cpu", "memory"}
	case ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_RESOURCES_CPU:
		return []string{"cpu"}
	case ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_RESOURCES_MEMORY:
		return []string{"memory"}
	case ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_RESOURCES_BOTH:
		return []string{"cpu", "memory"}
	default:
		return []string{"cpu", "memory"}
	}
}

// vpaControlledValuesString converts the proto enum to the VPA CRD string value.
func vpaControlledValuesString(cv ctrlv1.VerticalAutoscalingPolicy_ControlledValues) string {
	switch cv {
	case ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_VALUES_UNSPECIFIED:
		return "RequestsOnly"
	case ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_VALUES_REQUESTS:
		return "RequestsOnly"
	case ctrlv1.VerticalAutoscalingPolicy_CONTROLLED_VALUES_REQUESTS_AND_LIMITS:
		return "RequestsAndLimits"
	default:
		return "RequestsOnly"
	}
}
