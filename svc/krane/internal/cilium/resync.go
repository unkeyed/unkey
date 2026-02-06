package cilium

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/repeat"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
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
// The loop paginates through all krane-managed CiliumNetworkPolicy resources across all
// namespaces, calling GetDesiredCiliumNetworkPolicyState for each and applying or deleting
// as directed. Errors are logged but don't stop the loop from processing remaining
// policies.
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

			for _, policy := range policies.Items {
				policyID, ok := labels.GetCiliumNetworkPolicyID(policy.GetLabels())
				if !ok {
					logger.Error("unable to get cilium network policy id", "policy", policy.GetName())
					continue
				}

				res, err := c.cluster.GetDesiredCiliumNetworkPolicyState(ctx, connect.NewRequest(&ctrlv1.GetDesiredCiliumNetworkPolicyStateRequest{
					CiliumNetworkPolicyId: policyID,
				}))
				if err != nil {
					if connect.CodeOf(err) == connect.CodeNotFound {
						if err := c.DeleteCiliumNetworkPolicy(ctx, &ctrlv1.DeleteCiliumNetworkPolicy{
							K8SNamespace: policy.GetNamespace(),
							K8SName:      policy.GetName(),
						}); err != nil {
							logger.Error("unable to delete cilium network policy", "error", err.Error(), "policy_id", policyID)
							continue
						}
					}

					logger.Error("unable to get desired cilium network policy state", "error", err.Error(), "policy_id", policyID)
					continue
				}

				switch res.Msg.GetState().(type) {
				case *ctrlv1.CiliumNetworkPolicyState_Apply:
					if err := c.ApplyCiliumNetworkPolicy(ctx, res.Msg.GetApply()); err != nil {
						logger.Error("unable to apply cilium network policy", "error", err.Error(), "policy_id", policyID)
					}
				case *ctrlv1.CiliumNetworkPolicyState_Delete:
					if err := c.DeleteCiliumNetworkPolicy(ctx, res.Msg.GetDelete()); err != nil {
						logger.Error("unable to delete cilium network policy", "error", err.Error(), "policy_id", policyID)
					}
				}
			}

			cursor = policies.GetContinue()
			if cursor == "" {
				break
			}
		}
	})
}
