package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

// GetDesiredCiliumNetworkPolicyState returns the target state for a single Cilium network policy
// in the caller's region. This is a point query alternative to [Service.WatchCiliumNetworkPolicies]
// for cases where an agent needs to fetch state for a specific policy rather than streaming all changes.
//
// Returns CodeUnauthenticated if bearer token is invalid, CodeInvalidArgument if the
// X-Krane-Region header is missing, CodeNotFound if no policy exists with the given ID
// in the specified region, or CodeInternal for database errors.
func (s *Service) GetDesiredCiliumNetworkPolicyState(
	ctx context.Context,
	req *connect.Request[ctrlv1.GetDesiredCiliumNetworkPolicyStateRequest],
) (*connect.Response[ctrlv1.CiliumNetworkPolicyState], error) {
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")
	if err := assert.NotEmpty(region, "region is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	policyID := req.Msg.GetCiliumNetworkPolicyId()
	if err := assert.NotEmpty(policyID, "cilium network policy id is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	policy, err := db.Query.FindCiliumNetworkPolicyByIDAndRegion(ctx, s.db.RO(), db.FindCiliumNetworkPolicyByIDAndRegionParams{
		Region:                region,
		CiliumNetworkPolicyID: policyID,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if len(policy.Policy) == 0 {
		s.logger.Error("empty cilium policy payload", "policy_id", policy.ID)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("empty cilium policy payload for %s", policy.ID))
	}

	return connect.NewResponse(&ctrlv1.CiliumNetworkPolicyState{
		State: &ctrlv1.CiliumNetworkPolicyState_Apply{
			Apply: &ctrlv1.ApplyCiliumNetworkPolicy{
				CiliumNetworkPolicyId: policy.ID,
				K8SNamespace:          policy.EnvironmentID,
				K8SName:               policy.ID,
				Policy:                policy.Policy,
			},
		},
	}), nil
}
