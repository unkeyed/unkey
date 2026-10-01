package privatenetwork

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (r *Reconciler) snapshot(ctx context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
	stream, err := r.cluster.StreamPrivateNetworkState(ctx, &ctrlv1.StreamPrivateNetworkStateRequest{Cluster: r.clusterKey})
	if err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Warn("close private network snapshot stream", "error", err)
		}
	}()

	var snapshotConnections []*ctrlv1.PrivateNetworkConnection
	for stream.Receive() {
		chunk := stream.Msg()
		if chunk.GetComplete() {
			if chunk.GetTotal() != uint64(len(snapshotConnections)) || len(chunk.GetConnections()) != 0 {
				return nil, fmt.Errorf("private network snapshot has %d connections, complete chunk reports %d", len(snapshotConnections), chunk.GetTotal())
			}
			return snapshotConnections, nil
		}
		snapshotConnections = append(snapshotConnections, chunk.GetConnections()...)
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	return nil, fmt.Errorf("private network snapshot ended before it was complete")
}

type snapshotRejection struct {
	connectionSpec        *ctrlv1.PrivateNetworkConnection
	retainedConnectionKey string
	err                   error
}

func validateSnapshot(snapshotConnections []*ctrlv1.PrivateNetworkConnection) ([]*ctrlv1.PrivateNetworkConnection, []snapshotRejection) {
	identities := make(map[string]int, len(snapshotConnections))
	for _, connectionSpec := range snapshotConnections {
		if connectionSpec != nil {
			identities[connectionIdentity(connectionSpec)]++
		}
	}

	eligible := make([]*ctrlv1.PrivateNetworkConnection, 0, len(snapshotConnections))
	var rejected []snapshotRejection
	for i, connectionSpec := range snapshotConnections {
		if connectionSpec == nil {
			rejected = append(rejected, snapshotRejection{connectionSpec: nil, retainedConnectionKey: "", err: fmt.Errorf("invalid private network snapshot connection %d: connection is nil", i)})
			continue
		}
		err := assert.All(
			assert.NotEmpty(connectionSpec.GetWorkspaceId(), "workspace ID is required"),
			assert.NotEmpty(connectionSpec.GetProjectId(), "project ID is required"),
			assert.NotEmpty(connectionSpec.GetTargetAppId(), "target app ID is required"),
			assert.NotEmpty(connectionSpec.GetCallerDeploymentId(), "caller deployment ID is required"),
			assert.NotEmpty(connectionSpec.GetConnectionId(), "connection ID is required"),
			assert.Equal(len(validation.IsValidLabelValue(connectionSpec.GetCallerDeploymentId())), 0, "invalid caller deployment ID"),
			assert.Equal(len(validation.IsValidLabelValue(connectionSpec.GetConnectionId())), 0, "invalid connection ID"),
			assert.Equal(len(validation.IsDNS1123Label(connectionSpec.GetConnectionName())), 0, "invalid connection name"),
			assert.Equal(len(validation.IsDNS1123Label(connectionSpec.GetK8SNamespace())), 0, "invalid Kubernetes namespace"),
			assert.True(connectionSpec.GetTargetDeploymentId() == "" || connectionSpec.GetTargetPort() > 0, "resolved target port must be positive"),
			assert.GreaterOrEqual(connectionSpec.GetTargetPort(), int32(0), "port must not be negative"),
			assert.LessOrEqual(connectionSpec.GetTargetPort(), int32(65535), "port must be at most 65535"),
			assert.Equal(identities[connectionIdentity(connectionSpec)], 1, "duplicate connection identity"),
		)
		if err != nil {
			rejected = append(rejected, snapshotRejection{
				connectionSpec:        connectionSpec,
				retainedConnectionKey: publishedConnectionKey(connectionSpec),
				err:                   fmt.Errorf("invalid private network snapshot connection %d: %w", i, err),
			})
			continue
		}
		eligible = append(eligible, connectionSpec)
	}
	return eligible, rejected
}

func connectionIdentity(connectionSpec *ctrlv1.PrivateNetworkConnection) string {
	return connectionSpec.GetWorkspaceId() + "/" + connectionSpec.GetProjectId() + "/" + connectionSpec.GetCallerDeploymentId() + "/" + connectionSpec.GetConnectionName()
}

func publishedConnectionKey(connectionSpec *ctrlv1.PrivateNetworkConnection) string {
	if len(validation.IsDNS1123Label(connectionSpec.GetK8SNamespace())) != 0 || connectionSpec.GetConnectionId() == "" || connectionSpec.GetCallerDeploymentId() == "" {
		return ""
	}
	return connectionSpec.GetK8SNamespace() + "/" + connectionResourceName(connectionSpec)
}
