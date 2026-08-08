package buildqueue

import (
	"errors"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type RuleService struct {
	hydrav1.UnimplementedBuildRuleServiceServer
	db    db.Database
	admin *restateadmin.Client
}

var _ hydrav1.BuildRuleServiceServer = (*RuleService)(nil)

type RuleConfig struct {
	DB    db.Database
	Admin *restateadmin.Client
}

func NewRuleService(cfg RuleConfig) *RuleService {
	return &RuleService{
		UnimplementedBuildRuleServiceServer: hydrav1.UnimplementedBuildRuleServiceServer{},
		db:                                  cfg.DB,
		admin:                               cfg.Admin,
	}
}

func buildConcurrency(max uint16, found bool) int32 {
	if !found || max < 1 {
		return 1
	}
	return int32(max)
}

func (s *RuleService) Configure(ctx restate.ObjectContext, req *hydrav1.ConfigureBuildRulesRequest) (*hydrav1.ConfigureBuildRulesResponse, error) {
	workspaceID := req.GetWorkspaceId()
	if workspaceID == "" {
		return nil, restate.ToTerminalError(errors.New("workspace_id is required"))
	}

	concurrency, err := restate.Run(ctx, func(runCtx restate.RunContext) (int32, error) {
		limits, findErr := s.db.FindLimitsByWorkspaceID(runCtx, workspaceID)
		if db.IsNotFound(findErr) {
			return buildConcurrency(0, false), nil
		}
		return buildConcurrency(limits.BuildsConcurrentMax, true), findErr
	}, restate.WithName("find workspace build concurrency"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read workspace limits."))
	}

	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.admin.UpsertBuildConcurrencyRules(runCtx, workspaceID, concurrency)
	}, restate.WithName("upsert Restate build concurrency rules"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to configure build concurrency."))
	}

	return &hydrav1.ConfigureBuildRulesResponse{}, nil
}
