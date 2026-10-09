package deployment

import (
	"context"
	"fmt"
	"strconv"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/precondition"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const frontlinePolicySuffix = "-frontline-ingress"

func (c *Controller) ensureCiliumNetworkPolicy(ctx context.Context, req *ctrlv1.ApplyDeployment, rs *appsv1.ReplicaSet) error {
	policyName := req.GetK8SName() + frontlinePolicySuffix

	metadata := map[string]interface{}{
		"name":      policyName,
		"namespace": req.GetK8SNamespace(),
		"labels": labels.New().
			WorkspaceID(req.GetWorkspaceId()).
			ProjectID(req.GetProjectId()).
			AppID(req.GetAppId()).
			EnvironmentID(req.GetEnvironmentId()).
			EnvironmentKind(req.GetEnvironmentKind()).
			DeploymentID(req.GetDeploymentId()).
			ManagedByKrane().
			ComponentCiliumNetworkPolicy(),
	}
	if rs != nil {
		metadata["ownerReferences"] = []interface{}{
			map[string]interface{}{
				"apiVersion":         "apps/v1",
				"kind":               "ReplicaSet",
				"name":               rs.Name,
				"uid":                string(rs.UID),
				"controller":         true,
				"blockOwnerDeletion": true,
			},
		}
	}

	ingress := []interface{}{
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
	}

	spec := map[string]interface{}{
		"endpointSelector": map[string]interface{}{
			"matchLabels": map[string]interface{}{
				labels.LabelKeyWorkspaceID:  req.GetWorkspaceId(),
				labels.LabelKeyProjectID:    req.GetProjectId(),
				labels.LabelKeyAppID:        req.GetAppId(),
				labels.LabelKeyDeploymentID: req.GetDeploymentId(),
			},
		},
		"ingress": ingress,
	}
	if c.privateNetworkEnabled(req) {
		self := cilium.DeploymentEndpoint(labels.New().
			WorkspaceID(req.GetWorkspaceId()).
			ProjectID(req.GetProjectId()).
			AppID(req.GetAppId()).
			DeploymentID(req.GetDeploymentId()))
		ports := cilium.UnicastPorts()
		spec["enableDefaultDeny"] = map[string]interface{}{"ingress": true, "egress": false}
		spec["ingress"] = append(ingress, map[string]interface{}{"fromEndpoints": []interface{}{self}, "toPorts": ports})
		spec["egress"] = []interface{}{map[string]interface{}{"toEndpoints": []interface{}{self}, "toPorts": ports}}
	}

	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata":   metadata,
			"spec":       spec,
		},
	}

	client := c.dynamicClient.Resource(cilium.NetworkPolicyResource).Namespace(req.GetK8SNamespace())
	if rs == nil {
		existing, err := client.Get(ctx, policyName, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to read cilium network policy: %w", err)
		}
		if err == nil {
			if len(existing.GetOwnerReferences()) != 0 || existing.GetDeletionTimestamp() != nil {
				return fmt.Errorf("cilium network policy %s is awaiting garbage collection", policyName)
			}
			policy.SetResourceVersion(existing.GetResourceVersion())
		}
	}

	_, err := client.Apply(
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

func (c *Controller) deleteUnownedCiliumPolicy(ctx context.Context, namespace, name string) error {
	client := c.dynamicClient.Resource(cilium.NetworkPolicyResource).Namespace(namespace)
	policy, err := client.Get(ctx, name+frontlinePolicySuffix, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(policy.GetOwnerReferences()) != 0 {
		return nil
	}
	deploymentID, ok := labels.GetDeploymentID(policy.GetLabels())
	if !ok || deploymentID == "" || !labels.New().ManagedByKrane().ComponentCiliumNetworkPolicy().Matches(policy.GetLabels()) {
		return fmt.Errorf("cilium network policy %s is not a managed deployment policy", policy.GetName())
	}
	_, err = c.clientSet.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}
	pods, err := c.clientSet.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labels.New().DeploymentID(deploymentID).ToString(), Limit: 1,
	})
	if err != nil {
		return err
	}
	if len(pods.Items) != 0 {
		return nil
	}
	err = client.Delete(ctx, policy.GetName(), precondition.Unchanged(policy))
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
