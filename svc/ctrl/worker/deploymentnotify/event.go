package deploymentnotify

import (
	"errors"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) event(ctx restate.RunContext, req deployment.ReportRequest) (*notification.Event, error) {
	row, err := s.db.FindDeploymentForDeploy(ctx, req.DeploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.ToTerminalError(errors.New("deployment not found"))
		}
		return nil, err
	}
	if row.Status.IsTerminal() {
		switch {
		case row.Status == mysqltype.DeploymentsStatusReady || row.Status == mysqltype.DeploymentsStatusFailed:
		case row.Status == mysqltype.DeploymentsStatusCancelled:
			if req.Change != deployment.Cancelled {
				return nil, nil
			}
		case row.Status == mysqltype.DeploymentsStatusSuperseded:
			if req.Change != deployment.Superseded && req.Change != deployment.CompletedWithoutPromotion {
				return nil, nil
			}
		default:
			return nil, nil
		}
	}
	if req.Change == deployment.Live && row.Status != mysqltype.DeploymentsStatusReady {
		return nil, nil
	}
	environmentURL := ""
	route, err := s.db.FindFrontlineRouteByDeploymentIDAndSticky(ctx, db.FindFrontlineRouteByDeploymentIDAndStickyParams{
		DeploymentID: row.ID, Sticky: db.FrontlineRoutesStickyDeployment,
	})
	if err != nil && !db.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		environmentURL = "https://" + route.FullyQualifiedDomainName
	}
	logURL := fmt.Sprintf("%s/%s/projects/%s/apps/%s/deployments/%s", s.dashboardURL, row.WorkspaceSlug, row.ProjectID, row.AppID, row.ID)
	event, err := deployment.Event(row.WorkspaceID, notification.Resource{
		ProjectID: row.ProjectID, AppID: row.AppID, EnvironmentID: row.EnvironmentID,
	}, s.clock.Now().UnixMilli(), deployment.Payload{
		DeploymentID: row.ID, Change: req.Change, Project: row.ProjectSlug, App: row.AppSlug,
		Environment: row.EnvironmentSlug, EnvironmentURL: environmentURL, LogURL: logURL,
	})
	if err != nil {
		return nil, err
	}
	return &event, nil
}
