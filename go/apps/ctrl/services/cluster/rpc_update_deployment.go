package cluster

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) UpdateDeploymentStatus(ctx context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStatusRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStatusResponse], error) {

	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), req.Msg.GetDeploymentId())
	if err != nil {
		return nil, err
	}

	allRunning := len(req.Msg.GetInstances()) > 0

	for _, instance := range req.Msg.GetInstances() {
		if instance.Status != ctrlv1.UpdateDeploymentStatusRequest_Instance_STATUS_RUNNING {
			allRunning = false
		}

		var status db.InstancesStatus = db.InstancesStatusInactive
		switch instance.Status {
		case ctrlv1.UpdateDeploymentStatusRequest_Instance_STATUS_PENDING:
			status = db.InstancesStatusPending
		case ctrlv1.UpdateDeploymentStatusRequest_Instance_STATUS_RUNNING:
			status = db.InstancesStatusRunning
		case ctrlv1.UpdateDeploymentStatusRequest_Instance_STATUS_FAILED:
			status = db.InstancesStatusFailed
		}

		err = db.Query.UpsertInstance(ctx, s.db.RW(), db.UpsertInstanceParams{
			ID:            instance.GetId(),
			DeploymentID:  deployment.ID,
			WorkspaceID:   deployment.WorkspaceID,
			ProjectID:     deployment.ProjectID,
			Region:        req.Msg.GetRegion(),
			Address:       instance.GetAddress(),
			CpuMillicores: deployment.CpuMillicores,
			MemoryMib:     deployment.MemoryMib,
			Status:        status,
		})
		if err != nil {
			return nil, err
		}

	}

	if allRunning {
		err = db.Query.UpdateDeploymentStatus(ctx, s.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.Msg.GetDeploymentId(),
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			Status:    db.DeploymentsStatusReady,
		})
		if err != nil {
			return nil, err
		}
	}

	return connect.NewResponse(&ctrlv1.UpdateDeploymentStatusResponse{}), nil

}
