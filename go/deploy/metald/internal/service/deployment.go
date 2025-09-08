package service

import (
	"context"

	"connectrpc.com/connect"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

// CreateDeployment allocates a network, generates IDs etc
func (s *VMService) CreateDeployment(ctx context.Context, req *connect.Request[metaldv1.CreateDeploymentRequest]) (*connect.Response[metaldv1.CreateDeploymentResponse], error) {
	return connect.NewResponse(&metaldv1.CreateDeploymentResponse{
		VmIds: []string{"ud-001", "ud-002", "ud-003"},
	}), nil
}

// GetDeployment returns all of the VMs and their state for the passed deployment_id
func (s *VMService) GetDeployment(ctx context.Context, req *connect.Request[metaldv1.GetDeploymentRequest]) (*connect.Response[metaldv1.GetDeploymentResponse], error) {

	// Sample VMs to "act" against
	vms := []*metaldv1.GetDeploymentResponse_Vm{
		{
			Id:    "ud-001",
			Host:  "host01.asldkfja.unkey.app",
			State: metaldv1.VmState_VM_STATE_RUNNING,
			Port:  8081,
		},
		{
			Id:    "ud-002",
			Host:  "host02.asldkfja.unkey.app",
			State: metaldv1.VmState_VM_STATE_CREATED,
			Port:  8082,
		},
		{
			Id:    "vm-003",
			Host:  "host03.asldkfja.unkey.app",
			State: metaldv1.VmState_VM_STATE_RUNNING,
			Port:  8083,
		},
	}

	return connect.NewResponse(&metaldv1.GetDeploymentResponse{
		DeploymentId: req.Msg.GetDeploymentId(),
		Vms:          vms,
	}), nil
}
