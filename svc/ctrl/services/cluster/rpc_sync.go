package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Sync streams cluster state to an edge node for the given region.
//
// If sequence_last_seen is 0, Sync bootstraps the full desired state for the region.
// Stream close signals bootstrap completion. The client tracks the highest sequence
// from received messages and uses it for the next sync request.
//
// Sync is a bounded catch-up stream. The server stops after sending a batch of
// changes; clients reconnect to continue from their last-seen sequence.
func (s *Service) Sync(ctx context.Context, req *connect.Request[ctrlv1.SyncRequest], stream *connect.ServerStream[ctrlv1.State]) error {
	region := req.Msg.GetRegion()
	sequenceLastSeen := req.Msg.GetSequenceLastSeen()

	s.logger.Info("sync request received",
		"region", region,
		"sequenceLastSeen", sequenceLastSeen,
	)

	sequenceAfter := sequenceLastSeen
	if sequenceLastSeen == 0 {
		boundary, err := s.bootstrap(ctx, region, stream)
		if err != nil {
			return fmt.Errorf("bootstrap region=%q: %w", region, err)
		}
		sequenceAfter = boundary
	}

	changes, err := db.Query.ListStateChanges(ctx, s.db.RW(), db.ListStateChangesParams{
		Region:        region,
		AfterSequence: sequenceAfter,
		Limit:         100,
	})
	if err != nil {
		return fmt.Errorf("list state changes region=%q after=%d: %w", region, sequenceAfter, err)
	}

	for _, change := range changes {
		if err := s.processStateChange(ctx, region, change, stream); err != nil {
			return fmt.Errorf("process state change sequence=%d: %w", change.Sequence, err)
		}
	}

	return nil
}
