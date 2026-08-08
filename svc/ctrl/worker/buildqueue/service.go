// Package buildqueue admits deployments through Restate's native scoped
// concurrency limits.
package buildqueue

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/restate/compensation"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	runMaxAttempts uint = 5
	queueTimeout        = 6 * time.Hour
)

type Service struct {
	hydrav1.UnimplementedBuildQueueServiceServer
	db db.Database
}

var _ hydrav1.BuildQueueServiceServer = (*Service)(nil)

type Config struct {
	DB db.Database
}

func New(cfg Config) *Service {
	return &Service{
		UnimplementedBuildQueueServiceServer: hydrav1.UnimplementedBuildQueueServiceServer{},
		db:                                   cfg.DB,
	}
}

func (s *Service) Enqueue(ctx restate.ObjectContext, req *hydrav1.DeployRequest) (_ *hydrav1.DeployResponse, retErr error) {
	if req == nil || req.GetDeploymentId() == "" {
		return nil, restate.ToTerminalError(errors.New("deployment_id is required"))
	}

	compensations := compensation.New()
	compensations.Add("fail deployment waiting for build capacity", func(runCtx restate.RunContext) error {
		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		if err := s.db.UpdateDeploymentStatusIfActive(runCtx, db.UpdateDeploymentStatusIfActiveParams{
			ID:               req.GetDeploymentId(),
			Status:           mysqltype.DeploymentsStatusFailed,
			UpdatedAt:        now,
			TerminalStatuses: mysqltype.TerminalDeploymentStatuses,
		}); err != nil {
			return err
		}
		return s.db.EndDeploymentStep(runCtx, db.EndDeploymentStepParams{
			DeploymentID: req.GetDeploymentId(),
			Step:         db.DeploymentStepsStepQueued,
			EndedAt:      now,
			Error:        sql.NullString{Valid: true, String: retErr.Error()},
		})
	})
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, compensations.Execute(ctx))
		}
	}()

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Deployment, error) {
		return s.db.FindDeploymentById(runCtx, req.GetDeploymentId())
	}, restate.WithName("find queued deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read queued deployment."))
	}
	environment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return s.db.FindEnvironmentById(runCtx, deployment.EnvironmentID)
	}, restate.WithName("find queued deployment environment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read deployment environment."))
	}
	if environment.WorkspaceID != deployment.WorkspaceID {
		return nil, restate.ToTerminalError(errors.New("deployment and environment workspace mismatch"))
	}

	if _, err := hydrav1.NewBuildRuleServiceClient(ctx, deployment.WorkspaceID).Configure().Request(
		&hydrav1.ConfigureBuildRulesRequest{WorkspaceId: deployment.WorkspaceID},
	); err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to configure build concurrency."))
	}

	admitted := restate.Awakeable[bool](ctx)
	req.QueueAwakeableId = admitted.Id()
	deployClient := hydrav1.NewDeployServiceClient(ctx, deployment.ID, restate.WithScope(deployment.WorkspaceID))
	var deployFuture restate.ResponseFuture[*hydrav1.DeployResponse]
	if environment.Kind.IsProduction() {
		deployFuture = deployClient.Deploy().RequestFuture(req)
	} else {
		// For N > 1 this reserves one workspace permit for production. At N=1,
		// preview remains at one and Restate's fair scheduler prevents starvation.
		deployFuture = deployClient.Deploy().RequestFuture(req, restate.WithLimitKey("preview"))
	}
	timeoutFuture := restate.After(ctx, queueTimeout)
	completed, waitErr := restate.WaitFirst(ctx, admitted, timeoutFuture)
	if waitErr != nil {
		return nil, waitErr
	}
	if completed == timeoutFuture {
		restate.CancelInvocation(ctx, deployFuture.GetInvocationId())
		return nil, restate.ToTerminalError(fmt.Errorf("deployment exceeded %s build queue timeout", queueTimeout))
	}
	if _, awakeErr := admitted.Result(); awakeErr != nil {
		return nil, awakeErr
	}
	return deployFuture.Response()
}
