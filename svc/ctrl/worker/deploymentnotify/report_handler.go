package deploymentnotify

import (
	"errors"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
)

// Report requires the deployment ID as its object key and returns the dispatched
// event ID. Stale or suppressed changes return an empty ID without an error.
func (s *Service) Report(ctx restate.ObjectContext, req deployment.ReportRequest) (string, error) {
	if req.DeploymentID == "" || len(req.DeploymentID) > 48 || restate.Key(ctx) != req.DeploymentID || req.Change.Revision() == 0 {
		return "", restate.ToTerminalError(errors.New("invalid deployment report"))
	}
	last, err := restate.Get[deployment.Change](ctx, "last_change")
	if err != nil {
		return "", err
	}
	if last.Terminal() || last.Revision() >= req.Change.Revision() {
		return "", nil
	}
	event, err := restate.Run(ctx, func(runCtx restate.RunContext) (*notification.Event, error) {
		return s.event(runCtx, req)
	}, restate.WithName("freeze deployment report"))
	if err != nil {
		return "", err
	}
	if event == nil {
		return "", nil
	}
	payload, decodeErr := deployment.Decode(*event)
	if decodeErr != nil {
		return "", decodeErr
	}
	id, dispatchErr := s.dispatcher.Dispatch(ctx, *event)
	if dispatchErr != nil {
		return "", dispatchErr
	}
	restate.Set(ctx, "last_change", payload.Change)
	return id, nil
}
