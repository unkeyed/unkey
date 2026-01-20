package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

const syncBatchSize = 100

// Sync streams cluster state to a krane agent for the given region.
//
// Each resource carries its actual version, so clients track max(seen versions).
//
// IMPORTANT: Clients must only commit their version tracking after a clean stream
// close. This ensures atomic bootstrap: if a stream breaks mid-bootstrap, the client
// retries from version 0 rather than skipping resources that were never received.
//
// After bootstrap (versionLastSeen=0), clients should garbage-collect any k8s
// resources not mentioned in the bootstrap stream.
//
// Sync is a bounded catch-up stream. The server stops after sending a batch of
// changes; clients reconnect to continue from their last-seen version.
func (s *Service) Sync(ctx context.Context, req *connect.Request[ctrlv1.SyncRequest], stream *connect.ServerStream[ctrlv1.State]) error {
	region := req.Msg.GetRegion()
	versionLastSeen := req.Msg.GetVersionLastSeen()

	s.logger.Info("sync request received",
		"region", region,
		"versionLastSeen", versionLastSeen,
	)

	if err := s.streamStateAfterVersion(ctx, region, versionLastSeen, stream); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("stream state region=%q after_version=%d: %w", region, versionLastSeen, err))
	}

	return nil
}

// streamStateAfterVersion streams all resources with version > afterVersion in global version order.
// It uses a three-step approach:
// 1. Query the next batch of (version, kind) pairs in global order (lightweight UNION ALL)
// 2. Partition versions by kind and hydrate full data with targeted queries
// 3. Merge results by version and stream to the client
//
// As you can see, this is not terribly efficient, but it's easy and will do just fine for now.
// Later we can probably split it up and do 2 separete streams, one for deployments and one for sentinels
func (s *Service) streamStateAfterVersion(ctx context.Context, region string, afterVersion uint64, stream *connect.ServerStream[ctrlv1.State]) error {
	for {
		// Step 1: Get next batch of versions in global order
		versionRows, err := db.Query.ListClusterStateVersions(ctx, s.db.RO(), db.ListClusterStateVersionsParams{
			Region:       region,
			AfterVersion: afterVersion,
			Limit:        int32(syncBatchSize),
		})
		if err != nil {
			return fmt.Errorf("list cluster state versions after_version=%d: %w", afterVersion, err)
		}

		if len(versionRows) == 0 {
			return nil
		}

		// Step 2: Partition versions by kind
		var deploymentVersions, sentinelVersions []uint64
		for _, row := range versionRows {
			switch row.Kind {
			case "deployment":
				deploymentVersions = append(deploymentVersions, row.Version)
			case "sentinel":
				sentinelVersions = append(sentinelVersions, row.Version)
			}
		}

		// Step 3: Hydrate full data
		deploymentsByVersion := make(map[uint64]db.FindDeploymentTopologyByVersionsRow)
		if len(deploymentVersions) > 0 {
			topologies, err := db.Query.FindDeploymentTopologyByVersions(ctx, s.db.RO(), deploymentVersions)
			if err != nil {
				return fmt.Errorf("find deployment topologies by versions: %w", err)
			}
			for _, t := range topologies {
				deploymentsByVersion[t.DeploymentTopology.Version] = t
			}
		}

		sentinelsByVersion := make(map[uint64]db.Sentinel)
		if len(sentinelVersions) > 0 {
			sentinels, err := db.Query.FindSentinelsByVersions(ctx, s.db.RO(), sentinelVersions)
			if err != nil {
				return fmt.Errorf("find sentinels by versions: %w", err)
			}
			for _, sentinel := range sentinels {
				sentinelsByVersion[sentinel.Version] = sentinel
			}
		}

		// Step 4: Stream in global version order
		for _, row := range versionRows {
			var state *ctrlv1.State

			switch row.Kind {
			case "deployment":
				topology, ok := deploymentsByVersion[row.Version]
				if !ok {
					return fmt.Errorf("deployment topology version=%d not found after hydration", row.Version)
				}
				state = s.deploymentTopologyToState(topology)

			case "sentinel":
				sentinel, ok := sentinelsByVersion[row.Version]
				if !ok {
					return fmt.Errorf("sentinel version=%d not found after hydration", row.Version)
				}
				state = s.sentinelToState(sentinel)
			}

			if err := stream.Send(state); err != nil {
				return fmt.Errorf("send state version=%d kind=%s: %w", row.Version, row.Kind, err)
			}
		}

		// Update afterVersion for next iteration
		afterVersion = versionRows[len(versionRows)-1].Version

		// If we got fewer than batch size, we've reached the end
		if len(versionRows) < syncBatchSize {
			return nil
		}
	}
}

// deploymentTopologyToState converts a deployment topology row to a State message.
// If the deployment should not be running (replicas=0 or stopped), it returns a Delete.
func (s *Service) deploymentTopologyToState(topology db.FindDeploymentTopologyByVersionsRow) *ctrlv1.State {
	if topology.DeploymentTopology.DesiredReplicas == 0 ||
		topology.DeploymentTopology.DesiredStatus == db.DeploymentTopologyDesiredStatusStopped ||
		topology.DeploymentTopology.DesiredStatus == db.DeploymentTopologyDesiredStatusStopping {
		return &ctrlv1.State{
			Version: topology.DeploymentTopology.Version,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{
							K8SNamespace: topology.K8sNamespace.String,
							K8SName:      topology.Deployment.K8sName,
						},
					},
				},
			},
		}
	}

	var buildID *string
	if topology.Deployment.BuildID.Valid {
		buildID = &topology.Deployment.BuildID.String
	}

	return &ctrlv1.State{
		Version: topology.DeploymentTopology.Version,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						K8SNamespace:                  topology.K8sNamespace.String,
						K8SName:                       topology.Deployment.K8sName,
						WorkspaceId:                   topology.Deployment.WorkspaceID,
						EnvironmentId:                 topology.Deployment.EnvironmentID,
						ProjectId:                     topology.Deployment.ProjectID,
						DeploymentId:                  topology.Deployment.ID,
						Image:                         topology.Deployment.Image.String,
						Replicas:                      topology.DeploymentTopology.DesiredReplicas,
						CpuMillicores:                 int64(topology.Deployment.CpuMillicores),
						MemoryMib:                     int64(topology.Deployment.MemoryMib),
						EncryptedEnvironmentVariables: topology.Deployment.EncryptedEnvironmentVariables,
						BuildId:                       buildID,
					},
				},
			},
		},
	}
}

// sentinelToState converts a sentinel row to a State message.
// If the sentinel should not be running (replicas=0 or not running state),
// it returns a Delete instruction. Otherwise, it returns an Apply instruction.
func (s *Service) sentinelToState(sentinel db.Sentinel) *ctrlv1.State {
	if sentinel.DesiredReplicas == 0 || sentinel.DesiredState != db.SentinelsDesiredStateRunning {
		return &ctrlv1.State{
			Version: sentinel.Version,
			Kind: &ctrlv1.State_Sentinel{
				Sentinel: &ctrlv1.SentinelState{
					State: &ctrlv1.SentinelState_Delete{
						Delete: &ctrlv1.DeleteSentinel{
							K8SName: sentinel.K8sName,
						},
					},
				},
			},
		}
	}

	return &ctrlv1.State{
		Version: sentinel.Version,
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Apply{
					Apply: &ctrlv1.ApplySentinel{
						K8SName:       sentinel.K8sName,
						WorkspaceId:   sentinel.WorkspaceID,
						EnvironmentId: sentinel.EnvironmentID,
						ProjectId:     sentinel.ProjectID,
						SentinelId:    sentinel.ID,
						Image:         sentinel.Image,
						Replicas:      sentinel.DesiredReplicas,
						CpuMillicores: int64(sentinel.CpuMillicores),
						MemoryMib:     int64(sentinel.MemoryMib),
					},
				},
			},
		},
	}
}
