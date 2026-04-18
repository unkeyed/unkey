package cilium

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/conc"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// runResyncLoop periodically reconciles all Cilium network policies with their
// desired state from the control plane.
//
// The loop runs every minute as a consistency safety net. While
// [Controller.runDesiredStateApplyLoop] handles streaming updates, it can miss
// events during network partitions, controller restarts, or stream errors.
// This resync loop guarantees eventual consistency by querying the control plane
// for each existing CiliumNetworkPolicy and applying any drift.
//
// Resources are processed concurrently via [conc.ForEach] so that a slow RPC
// for one policy does not block reconciliation of others.
func (c *Controller) runResyncLoop(ctx context.Context) {
	repeat.Every(1*time.Minute, func() {
		logger.Info("running periodic resync")

		cursor := ""
		for {
			policies, err := c.listCiliumNetworkPolicies(ctx, cursor)
			if err != nil {
				logger.Error("unable to list cilium network policies", "error", err.Error())
				return
			}

			conc.ForEach(ctx, policies.Items, func(ctx context.Context, policy *unstructured.Unstructured) {
				c.resyncCiliumNetworkPolicy(ctx, policy)
			})

			cursor = policies.GetContinue()
			if cursor == "" {
				break
			}
		}
	})
}

// resyncCiliumNetworkPolicy reconciles a single CiliumNetworkPolicy against the
// control plane's desired state.
func (c *Controller) resyncCiliumNetworkPolicy(ctx context.Context, policy *unstructured.Unstructured) {
	policyID, ok := labels.GetCiliumNetworkPolicyID(policy.GetLabels())
	if !ok {
		logger.Error("unable to get cilium network policy id", "policy", policy.GetName())
		return
	}

	res, err := c.cluster.GetDesiredCiliumNetworkPolicyState(ctx, &ctrlv1.GetDesiredCiliumNetworkPolicyStateRequest{
		Region:                c.regionKey(),
		CiliumNetworkPolicyId: policyID,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			if err := c.DeleteCiliumNetworkPolicy(ctx, &ctrlv1.DeleteCiliumNetworkPolicy{
				K8SNamespace: policy.GetNamespace(),
				K8SName:      policy.GetName(),
			}); err != nil {
				logger.Error("unable to delete cilium network policy", "error", err.Error(), "policy_id", policyID)
			}

			return
		}

		logger.Error("unable to get desired cilium network policy state", "error", err.Error(), "policy_id", policyID)
		return
	}

	switch res.GetState().(type) {
	case *ctrlv1.CiliumNetworkPolicyState_Apply:
		if err := c.ApplyCiliumNetworkPolicy(ctx, res.GetApply()); err != nil {
			logger.Error("unable to apply cilium network policy", "error", err.Error(), "policy_id", policyID)
		}
	case *ctrlv1.CiliumNetworkPolicyState_Delete:
		if err := c.DeleteCiliumNetworkPolicy(ctx, res.GetDelete()); err != nil {
			logger.Error("unable to delete cilium network policy", "error", err.Error(), "policy_id", policyID)
		}
	}
}
