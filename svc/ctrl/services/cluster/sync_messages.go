package cluster

import (
	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Message sending helpers - centralized protobuf construction.

func (s *Service) sendDeploymentDelete(stream *connect.ServerStream[ctrlv1.State], sequence uint64, namespace, name string) error {
	return stream.Send(&ctrlv1.State{
		Sequence: sequence,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						K8SNamespace: namespace,
						K8SName:      name,
					},
				},
			},
		},
	})
}

func (s *Service) sendDeploymentApply(stream *connect.ServerStream[ctrlv1.State], sequence uint64, apply *ctrlv1.ApplyDeployment) error {
	return stream.Send(&ctrlv1.State{
		Sequence: sequence,
		Kind: &ctrlv1.State_Deployment{
			Deployment: &ctrlv1.DeploymentState{
				State: &ctrlv1.DeploymentState_Apply{
					Apply: apply,
				},
			},
		},
	})
}

func (s *Service) sendDeploymentApplyFromTopology(stream *connect.ServerStream[ctrlv1.State], sequence uint64, topology db.ListDesiredDeploymentTopologyRow) error {
	var buildID *string
	if topology.Deployment.BuildID.Valid {
		buildID = &topology.Deployment.BuildID.String
	}
	return s.sendDeploymentApply(stream, sequence, &ctrlv1.ApplyDeployment{
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
	})
}

func newApplyDeploymentFromTopology(topology db.FindDeploymentTopologyByIDAndRegionRow) *ctrlv1.ApplyDeployment {
	var buildID *string
	if topology.BuildID.Valid {
		buildID = &topology.BuildID.String
	}
	return &ctrlv1.ApplyDeployment{
		K8SNamespace:                  topology.K8sNamespace.String,
		K8SName:                       topology.K8sName,
		WorkspaceId:                   topology.WorkspaceID,
		EnvironmentId:                 topology.EnvironmentID,
		ProjectId:                     topology.ProjectID,
		DeploymentId:                  topology.ID,
		Image:                         topology.Image.String,
		Replicas:                      topology.DesiredReplicas,
		CpuMillicores:                 int64(topology.CpuMillicores),
		MemoryMib:                     int64(topology.MemoryMib),
		EncryptedEnvironmentVariables: topology.EncryptedEnvironmentVariables,
		BuildId:                       buildID,
	}
}

func (s *Service) sendSentinelDelete(stream *connect.ServerStream[ctrlv1.State], sequence uint64, name string) error {
	return stream.Send(&ctrlv1.State{
		Sequence: sequence,
		Kind: &ctrlv1.State_Sentinel{
			Sentinel: &ctrlv1.SentinelState{
				State: &ctrlv1.SentinelState_Delete{
					Delete: &ctrlv1.DeleteSentinel{
						K8SName: name,
					},
				},
			},
		},
	})
}

func (s *Service) sendSentinelApply(stream *connect.ServerStream[ctrlv1.State], sequence uint64, sentinel db.Sentinel) error {
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
