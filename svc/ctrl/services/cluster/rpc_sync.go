package cluster

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

func (s *Service) Sync(ctx context.Context, req *connect.Request[ctrlv1.SyncRequest], stream *connect.ServerStream[ctrlv1.State]) error {
	region := req.Msg.GetRegion()
	sequence := req.Msg.GetSequenceLastSeen()

	s.logger.Info("sync request received",
		"region", region,
		"sequence", sequence,
	)

	if sequence > 0 {
		minSeq, err := db.Query.GetMinStateChangeSequence(ctx, s.db.RO(), region)
		if err != nil {
			return err
		}
		if sequence < uint64(minSeq) {
			return connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("sequence %d is behind minimum retained sequence %d, full resync required", sequence, minSeq))
		}
	}

	if sequence == 0 {
		var err error
		sequence, err = s.bootstrap(ctx, region, stream)
		if err != nil {
			return err
		}
	}

	return s.watch(ctx, region, sequence, stream)
}

func (s *Service) bootstrap(ctx context.Context, region string, stream *connect.ServerStream[ctrlv1.State]) (uint64, error) {
	maxSeq, err := db.Query.GetMaxStateChangeSequence(ctx, s.db.RO(), region)
	if err != nil {
		return 0, err
	}
	sequence := uint64(maxSeq)

	cursor := ""
	for {
		topologies, err := db.Query.ListDesiredDeploymentTopology(ctx, s.db.RO(), db.ListDesiredDeploymentTopologyParams{
			Region:           region,
			DesiredState:     db.DeploymentsDesiredStateRunning,
			PaginationCursor: cursor,
			Limit:            1000,
		})
		if err != nil {
			return 0, err
		}
		if len(topologies) == 0 {
			break
		}
		cursor = topologies[len(topologies)-1].Deployment.ID

		for _, t := range topologies {
			if err := s.streamDeployment(stream, sequence, t); err != nil {
				return 0, err
			}
		}
	}

	cursor = ""
	for {
		sentinels, err := db.Query.ListDesiredSentinels(ctx, s.db.RO(), db.ListDesiredSentinelsParams{
			Region:           region,
			DesiredState:     db.SentinelsDesiredStateRunning,
			PaginationCursor: cursor,
			Limit:            100,
		})
		if err != nil {
			return 0, err
		}
		if len(sentinels) == 0 {
			break
		}
		cursor = sentinels[len(sentinels)-1].ID

		for _, sentinel := range sentinels {
			if err := s.streamSentinel(stream, sequence, sentinel); err != nil {
				return 0, err
			}
		}
	}

	// Send BOOKMARK with sequence
	if err := stream.Send(&ctrlv1.State{
		Sequence: sequence,
		Kind: &ctrlv1.State_Bookmark{
			Bookmark: &ctrlv1.Bookmark{Sequence: sequence},
		},
	}); err != nil {
		return 0, err
	}

	s.logger.Info("bootstrap complete", "sequence", sequence)
	return sequence, nil
}

func (s *Service) watch(ctx context.Context, region string, sequence uint64, stream *connect.ServerStream[ctrlv1.State]) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		changes, err := db.Query.ListStateChanges(ctx, s.db.RO(), db.ListStateChangesParams{
			Region:        region,
			AfterSequence: sequence,
			Limit:         100,
		})
		if err != nil {
			return err
		}

		if len(changes) == 0 {
			time.Sleep(250 * time.Millisecond)
			continue
		}

		for _, c := range changes {
			if err := s.processStateChange(ctx, region, c, stream); err != nil {
				// Stop on error - client will reconnect from last known sequence
				return fmt.Errorf("failed to process state change at sequence %d: %w", c.Sequence, err)
			}
			sequence = c.Sequence
		}
	}
}

// processStateChange fetches the resource and streams it if it applies to this region.
func (s *Service) processStateChange(ctx context.Context, region string, change db.ListStateChangesRow, stream *connect.ServerStream[ctrlv1.State]) error {
	switch change.ResourceType {
	case db.StateChangesResourceTypeDeployment:
		return s.processDeploymentChange(ctx, region, change, stream)
	case db.StateChangesResourceTypeSentinel:
		return s.processSentinelChange(ctx, region, change, stream)
	default:
		s.logger.Warn("unknown resource type", "type", change.ResourceType)
		return nil
	}
}

func (s *Service) processDeploymentChange(ctx context.Context, region string, change db.ListStateChangesRow, stream *connect.ServerStream[ctrlv1.State]) error {
	d, err := db.Query.FindDeploymentById(ctx, s.db.RO(), change.ResourceID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil
		}
		return err
	}
	ws, err := db.Query.FindWorkspaceByID(ctx, s.db.RO(), d.WorkspaceID)
	if err != nil {
		return err
	}

	if change.Op == db.StateChangesOpDelete {
		return stream.Send(&ctrlv1.State{
			Sequence: change.Sequence,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{
							K8SNamespace: ws.K8sNamespace.String,
							K8SName:      d.K8sName,
						},
					},
				},
			},
		})
	}

	t, err := db.Query.FindDeploymentTopologyByIDAndRegion(ctx, s.db.RO(), db.FindDeploymentTopologyByIDAndRegionParams{
		DeploymentID: change.ResourceID,
		Region:       region,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return stream.Send(&ctrlv1.State{
				Sequence: change.Sequence,
				Kind: &ctrlv1.State_Deployment{
					Deployment: &ctrlv1.DeploymentState{
						State: &ctrlv1.DeploymentState_Delete{
							Delete: &ctrlv1.DeleteDeployment{
								K8SNamespace: ws.K8sNamespace.String,
								K8SName:      d.K8sName,
							},
						},
					},
				},
			})
		}
		return err
	}

	if t.DesiredState != db.DeploymentsDesiredStateRunning {
		return stream.Send(&ctrlv1.State{
			Sequence: change.Sequence,
			Kind: &ctrlv1.State_Deployment{
				Deployment: &ctrlv1.DeploymentState{
					State: &ctrlv1.DeploymentState_Delete{
						Delete: &ctrlv1.DeleteDeployment{
							K8SNamespace: ws.K8sNamespace.String,
							K8SName:      d.K8sName,
						},
					},
				},
			},
		})
	}

	var buildID *string
	if t.BuildID.Valid {
		buildID = &t.BuildID.String
	}
	return stream.Send(&ctrlv1.State{
		Sequence: change.Sequence,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						K8SNamespace:                  t.K8sNamespace.String,
						K8SName:                       t.K8sName,
						WorkspaceId:                   t.WorkspaceID,
						EnvironmentId:                 t.EnvironmentID,
						ProjectId:                     t.ProjectID,
						DeploymentId:                  t.ID,
						Image:                         t.Image.String,
						Replicas:                      t.DesiredReplicas,
						CpuMillicores:                 int64(t.CpuMillicores),
						MemoryMib:                     int64(t.MemoryMib),
						EncryptedEnvironmentVariables: t.EncryptedEnvironmentVariables,
						BuildId:                       buildID,
					},
				},
			},
		},
	})
}

func (s *Service) processSentinelChange(ctx context.Context, region string, change db.ListStateChangesRow, stream *connect.ServerStream[ctrlv1.State]) error {
	sentinel, err := db.Query.FindSentinelByID(ctx, s.db.RO(), change.ResourceID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil
		}
		return err
	}

	if change.Op == db.StateChangesOpDelete || sentinel.Region != region || sentinel.DesiredState != db.SentinelsDesiredStateRunning {
		return stream.Send(&ctrlv1.State{
			Sequence: change.Sequence,
			Kind: &ctrlv1.State_Sentinel{
				Sentinel: &ctrlv1.SentinelState{
					State: &ctrlv1.SentinelState_Delete{
						Delete: &ctrlv1.DeleteSentinel{
							K8SName: sentinel.K8sName,
						},
					},
				},
			},
		})
	}

	return stream.Send(&ctrlv1.State{
		Sequence: change.Sequence,
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
	})
}

func (s *Service) streamDeployment(stream *connect.ServerStream[ctrlv1.State], sequence uint64, t db.ListDesiredDeploymentTopologyRow) error {
	var buildID *string
	if t.Deployment.BuildID.Valid {
		buildID = &t.Deployment.BuildID.String
	}
	return stream.Send(&ctrlv1.State{
		Sequence: sequence,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: &ctrlv1.ApplyDeployment{
						K8SNamespace:                  t.K8sNamespace.String,
						K8SName:                       t.Deployment.K8sName,
						WorkspaceId:                   t.Deployment.WorkspaceID,
						EnvironmentId:                 t.Deployment.EnvironmentID,
						ProjectId:                     t.Deployment.ProjectID,
						DeploymentId:                  t.Deployment.ID,
						Image:                         t.Deployment.Image.String,
						Replicas:                      t.DeploymentTopology.DesiredReplicas,
						CpuMillicores:                 int64(t.Deployment.CpuMillicores),
						MemoryMib:                     int64(t.Deployment.MemoryMib),
						EncryptedEnvironmentVariables: t.Deployment.EncryptedEnvironmentVariables,
						BuildId:                       buildID,
					},
				},
			},
		},
	})
}

func (s *Service) streamSentinel(stream *connect.ServerStream[ctrlv1.State], sequence uint64, sentinel db.Sentinel) error {
	return stream.Send(&ctrlv1.State{
		Sequence: sequence,
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
	})
}
