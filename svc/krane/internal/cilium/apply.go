package cilium

import (
	"context"
	"encoding/json"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ApplyCiliumNetworkPolicy creates or updates a CiliumNetworkPolicy in the cluster.
//
// The method uses server-side apply with the dynamic client, enabling
// concurrent modifications from different sources without conflicts.
//
// ApplyCiliumNetworkPolicy validates required fields and returns an error if any are missing
// or invalid: K8sNamespace, K8sName, CiliumNetworkPolicyId, and Policy must be non-empty.
func (c *Controller) ApplyCiliumNetworkPolicy(ctx context.Context, req *ctrlv1.ApplyCiliumNetworkPolicy) error {
	logger.Info("applying cilium network policy",
		"namespace", req.GetK8SNamespace(),
		"name", req.GetK8SName(),
		"policy_id", req.GetCiliumNetworkPolicyId(),
	)

	err := assert.All(
		assert.NotEmpty(req.GetCiliumNetworkPolicyId(), "Cilium network policy ID is required"),
		assert.NotEmpty(req.GetK8SNamespace(), "Namespace is required"),
		assert.NotEmpty(req.GetK8SName(), "K8s CRD name is required"),
		assert.NotEmpty(req.GetPolicy(), "Policy is required"),
	)
	if err != nil {
		return err
	}

	err = c.ensureNamespaceExists(ctx, req.GetK8SNamespace())
	if err != nil {
		return err
	}

	// nolint:exhaustruct
	policy := unstructured.Unstructured{}

	if err := json.Unmarshal(req.GetPolicy(), &policy.Object); err != nil {
		return fmt.Errorf("failed to unmarshal cilium policy: %w", err)
	}

	policy.SetNamespace(req.GetK8SNamespace())
	policy.SetName(req.GetK8SName())
	policy.SetLabels(labels.New().ManagedByKrane().ComponentCiliumNetworkPolicy().NetworkPolicyID(req.GetCiliumNetworkPolicyId()))

	gvr := schema.GroupVersionResource{
		Group:    "cilium.io",
		Version:  "v2",
		Resource: "ciliumnetworkpolicies",
	}

	_, err = c.dynamicClient.Resource(gvr).Namespace(req.GetK8SNamespace()).Apply(
		ctx,
		req.GetK8SName(),
		&policy,
		metav1.ApplyOptions{FieldManager: "krane"},
	)
	if err != nil {
		return fmt.Errorf("failed to apply cilium network policy: %w", err)
	}

	return nil
}
