package cluster

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/privatenetwork"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

const privateNetworkChunkSize = 1000

// StreamPrivateNetworkState sends the platform's complete private network
// catalog, which Ctrl keeps current from CDC instead of reading it per call.
// It reports Unavailable until the catalog is built and while its CDC stream
// is silent. Krane treats a stream without the final complete chunk as partial.
func (s *Service) StreamPrivateNetworkState(ctx context.Context, req *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
	result := "success"
	defer func() { metrics.PrivateNetworkSnapshotsTotal.WithLabelValues(result).Inc() }()

	if err := auth.Authenticate(req, s.bearer); err != nil {
		result = "unauthenticated"
		return err
	}

	cluster, err := s.resolveCluster(ctx, req.Msg.GetCluster())
	if err != nil {
		result = "unknown_cluster"
		if connect.CodeOf(err) == connect.CodeInternal {
			result = "database_error"
		}
		return err
	}

	if s.privateNetwork == nil {
		result = "unavailable"
		return connect.NewError(connect.CodeUnavailable, errors.New("private network catalog is not configured"))
	}
	snapshot, err := s.privateNetwork.Snapshot(ctx, cluster.RegionPlatform)
	if err != nil {
		result = "unavailable"
		return connect.NewError(connect.CodeUnavailable, err)
	}

	if err := sendPrivateNetworkSnapshot(stream, snapshot, req.Msg.GetKnownVersion()); err != nil {
		result = "send_error"
		return err
	}
	return nil
}

func sendPrivateNetworkSnapshot(stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk], snapshot privatenetwork.Snapshot, knownVersion string) error {
	total := uint64(len(snapshot.Connections))
	if knownVersion != "" && knownVersion == snapshot.Version {
		return stream.Send(&ctrlv1.PrivateNetworkStateChunk{
			Complete: true, Unchanged: true, Version: snapshot.Version, SnapshotId: snapshot.ID, Total: total, Certified: snapshot.Certified,
		})
	}

	for start := 0; start < len(snapshot.Connections); start += privateNetworkChunkSize {
		if err := stream.Send(&ctrlv1.PrivateNetworkStateChunk{
			Connections: snapshot.Connections[start:min(start+privateNetworkChunkSize, len(snapshot.Connections))],
			Version:     snapshot.Version, SnapshotId: snapshot.ID,
		}); err != nil {
			return err
		}
	}
	return stream.Send(&ctrlv1.PrivateNetworkStateChunk{
		Complete: true, Total: total, Topology: snapshot.Topology, Version: snapshot.Version, SnapshotId: snapshot.ID, Certified: snapshot.Certified,
	})
}
