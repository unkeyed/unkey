package deploy

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
)

// WorkflowServer serves DeployWorkflow with the object's code. It exists until
// DeployService drains and is deleted; then these methods move onto Workflow
type WorkflowServer struct {
	hydrav1.UnimplementedDeployWorkflowServer
	w *Workflow
}

var _ hydrav1.DeployWorkflowServer = (*WorkflowServer)(nil)

func NewWorkflowServer(w *Workflow) *WorkflowServer {
	wf := *w
	wf.asWorkflow = true
	return &WorkflowServer{
		UnimplementedDeployWorkflowServer: hydrav1.UnimplementedDeployWorkflowServer{},
		w:                                 &wf,
	}
}

func (s *WorkflowServer) Create(ctx restate.WorkflowSharedContext, req *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	return s.w.create(ctx, req)
}

func (s *WorkflowServer) Deploy(ctx restate.WorkflowContext, req *hydrav1.DeployRequest) (*hydrav1.DeployResponse, error) {
	return s.w.Deploy(ctx, req)
}

// NotifyInstancesReady tolerates a repeat: a second krane report after the
// promise resolved must not fail
func (s *WorkflowServer) NotifyInstancesReady(ctx restate.WorkflowSharedContext, _ *hydrav1.NotifyInstancesReadyRequest) (*hydrav1.NotifyInstancesReadyResponse, error) {
	if err := restate.Promise[restate.Void](ctx, instancesReadyPromise).Resolve(restate.Void{}); err != nil && !restate.IsTerminalError(err) {
		return nil, err
	}
	return &hydrav1.NotifyInstancesReadyResponse{}, nil
}
