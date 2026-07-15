package deployment

import (
	"context"
	"fmt"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ensureCiliumNetworkPolicy creates or updates a CiliumNetworkPolicy that
// permits frontline pods to reach this deployment on its container port.
// Cilium's default-deny kicks in for any endpoint selected by a CNP, so
// installing this policy is what unblocks ingress; without it, the
// customer pods are reachable only from inside their own namespace.
//
// The policy is namespaced to the deployment's namespace and owned by the
// ReplicaSet so it is garbage-collected automatically when the deployment
// is deleted.
func (c *Controller) ensureCiliumNetworkPolicy(ctx context.Context, req *ctrlv1.ApplyDeployment, rs *appsv1.ReplicaSet) error {
	policyName := fmt.Sprintf("%s-frontline-ingress", req.GetK8SName())

	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      policyName,
				"namespace": req.GetK8SNamespace(),
				"labels": labels.New().
					WorkspaceID(req.GetWorkspaceId()).
					ProjectID(req.GetProjectId()).
					AppID(req.GetAppId()).
					EnvironmentID(req.GetEnvironmentId()).
					DeploymentID(req.GetDeploymentId()).
					ManagedByKrane().
					ComponentCiliumNetworkPolicy(),
				"ownerReferences": []interface{}{
					map[string]interface{}{
						"apiVersion":         "apps/v1",
						"kind":               "ReplicaSet",
						"name":               rs.Name,
						"uid":                string(rs.UID),
						"controller":         true,
						"blockOwnerDeletion": true,
					},
				},
			},
			"spec": map[string]interface{}{
				"endpointSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						labels.LabelKeyDeploymentID: req.GetDeploymentId(),
					},
				},
				"ingress": []interface{}{
					map[string]interface{}{
						"fromEndpoints": []interface{}{
							map[string]interface{}{
								"matchLabels": map[string]interface{}{
									labels.LabelKeyNamespace: frontlineNamespace,
								},
							},
						},
						"toPorts": []interface{}{
							map[string]interface{}{
								"ports": []interface{}{
									map[string]interface{}{
										"port":     strconv.Itoa(int(req.GetPort())),
										"protocol": "TCP",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumnetworkpolicies",
	}

	// Server-side apply so concurrent reconciles converge instead of
	// fighting over field ownership.
	_, err := c.dynamicClient.Resource(gvr).Namespace(req.GetK8SNamespace()).Apply(
		ctx,
		policyName,
		policy,
		metav1.ApplyOptions{FieldManager: fieldManagerKrane},
	)
	if err != nil {
		return fmt.Errorf("failed to apply cilium network policy: %w", err)
	}

	return nil
}
