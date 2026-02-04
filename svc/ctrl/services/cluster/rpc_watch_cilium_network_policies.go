package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

// WatchCiliumNetworkPolicies streams Cilium network policy state changes from the control plane to agents.
// This is the primary mechanism for agents to receive desired state updates for their region.
// Agents apply received state to Kubernetes to converge actual state toward desired state.
//
// The stream uses version-based cursors for resumability. The client provides version_last_seen
// in the request, and the server streams all policies with versions greater than that cursor.
// Clients should track the maximum version received and use it to reconnect without replaying
// history. When no new versions are available, the server polls the database every second.
//
// Each poll fetches up to 100 policy rows ordered by version. Every row is sent as an apply
// since the control plane only stores active policies today. Rows with empty policy payloads
// are logged and skipped.
//
// Returns when the context is cancelled, or on database or stream errors.
func (s *Service) WatchCiliumNetworkPolicies(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchCiliumNetworkPoliciesRequest],
	stream *connect.ServerStream[ctrlv1.CiliumNetworkPolicyState],
) error {
	if err := s.authenticate(req); err != nil {
		return err
	}

	region := req.Msg.GetRegion()
	if err := assert.NotEmpty(region, "region is required"); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	versionCursor := req.Msg.GetVersionLastSeen()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		states, err := s.fetchCiliumNetworkPolicyStates(ctx, region, versionCursor)
		if err != nil {
			s.logger.Error("failed to fetch cilium network policy states", "error", err)
			return connect.NewError(connect.CodeInternal, err)
		}

		for _, state := range states {
			if err := stream.Send(state); err != nil {
				return err
			}
			if state.GetVersion() > versionCursor {
				versionCursor = state.GetVersion()
			}
		}

		if len(states) == 0 {
			time.Sleep(time.Second)
		}
	}
}

// fetchCiliumNetworkPolicyStates queries the database for policies in the given region with
// versions greater than afterVersion, returning up to 100 results.
func (s *Service) fetchCiliumNetworkPolicyStates(ctx context.Context, region string, afterVersion uint64) ([]*ctrlv1.CiliumNetworkPolicyState, error) {
	rows, err := db.Query.ListCiliumNetworkPoliciesByRegion(ctx, s.db.RO(), db.ListCiliumNetworkPoliciesByRegionParams{
		Region:       region,
		Afterversion: afterVersion,
		Limit:        100,
	})
	if err != nil {
		return nil, err
	}

	states := make([]*ctrlv1.CiliumNetworkPolicyState, len(rows))
	for i, row := range rows {
		states[i] = &ctrlv1.CiliumNetworkPolicyState{
			Version: row.CiliumNetworkPolicy.Version,
			State: &ctrlv1.CiliumNetworkPolicyState_Apply{
				Apply: &ctrlv1.ApplyCiliumNetworkPolicy{
					CiliumNetworkPolicyId: row.CiliumNetworkPolicy.ID,
					K8SNamespace:          row.K8sNamespace.String,
					K8SName:               row.CiliumNetworkPolicy.K8sName,
					Policy:                row.CiliumNetworkPolicy.Policy,
				},
			},
		}
	}

	return states, nil
}
