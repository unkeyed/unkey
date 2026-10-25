package deploymentnotify

import (
	"errors"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	workernotification "github.com/unkeyed/unkey/svc/ctrl/worker/notification"
)

type Config struct {
	DB           db.Database
	Clock        clock.Clock
	DashboardURL string
	Dispatcher   *workernotification.Dispatcher
}

type Service struct {
	db           db.Database
	clock        clock.Clock
	dashboardURL string
	dispatcher   *workernotification.Dispatcher
}

func New(cfg Config) (*Service, error) {
	if err := assert.All(assert.NotNil(cfg.DB), assert.NotNil(cfg.Clock), assert.NotNil(cfg.Dispatcher)); err != nil {
		return nil, err
	}
	return &Service{db: cfg.DB, clock: cfg.Clock, dashboardURL: strings.TrimRight(cfg.DashboardURL, "/"), dispatcher: cfg.Dispatcher}, nil
}

func (s *Service) Definition() restate.ServiceDefinition {
	return restate.NewObject(deployment.ServiceName).
		Handler("Report", restate.NewObjectHandler(s.Report, restate.WithIngressPrivate(true), restate.WithInvocationRetryPolicy(
			restate.WithInitialRetryInterval(time.Second), restate.WithMaxRetryInterval(time.Minute),
			restate.WithMaxRetryAttempts(20), restate.PauseOnMaxAttempts(),
		)))
}

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
	id, dispatchErr := s.dispatcher.Dispatch(ctx, *event)
	if dispatchErr != nil {
		return "", dispatchErr
	}
	restate.Set(ctx, "last_change", req.Change)
	return id, nil
}

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
