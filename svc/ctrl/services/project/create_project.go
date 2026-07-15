package project

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// CreateProject creates an empty project. Apps and their environments are
// created separately via [app.Service.CreateApp], so a fresh project starts
// with no apps.
func (s *Service) CreateProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateProjectRequest],
) (*connect.Response[ctrlv1.CreateProjectResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetName(), "name is required"),
		assert.NotEmpty(req.Msg.GetSlug(), "slug is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	workspaceID := req.Msg.GetWorkspaceId()

	// Authoritative entitlement gate: CLI, API, and git-triggered deploys skip
	// the dashboard, so creation is enforced here, not just in the UI. Observe
	// mode (the default) logs would-block instead of failing.
	entitlement, err := s.db.FindWorkspaceDeployEntitlement(ctx, workspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("workspace %q not found", workspaceID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load workspace: %w", err))
	}
	if !deployEntitled(entitlement.DeployPlan, entitlement.DeployPlanOverride) {
		if s.enforceDeployGate {
			return nil, connect.NewError(
				connect.CodeFailedPrecondition,
				fmt.Errorf("workspace %q has no Compute plan", workspaceID),
			)
		}
		logger.Warn("deploy gate would block project creation",
			"event", "deploy_gate.would_block",
			"workspaceId", workspaceID,
		)
	}

	projectID := uid.New(uid.ProjectPrefix)
	now := time.Now().UnixMilli()

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.NewQueries(tx).InsertProject(txCtx, db.InsertProjectParams{
			ID:               projectID,
			WorkspaceID:      workspaceID,
			Name:             req.Msg.GetName(),
			Slug:             req.Msg.GetSlug(),
			DeleteProtection: sql.NullBool{Valid: false},
			CreatedAt:        now,
			UpdatedAt:        sql.NullInt64{Valid: false},
		}); txErr != nil {
			if db.IsDuplicateKeyError(txErr) {
				return connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("project with slug %q already exists: %w", req.Msg.GetSlug(), txErr))
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert project: %w", txErr))
		}

		a := req.Msg.GetActor()
		if txErr := s.auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   workspaceID,
				Event:         auditlog.ProjectCreateEvent,
				Display:       fmt.Sprintf("Created project %s", projectID),
				ActorID:       a.GetId(),
				ActorName:     a.GetName(),
				ActorType:     actor.AuditType(a.GetType()),
				ActorMeta:     actor.Meta(a.GetMeta()),
				RemoteIP:      a.GetRemoteIp(),
				UserAgent:     a.GetUserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          projectID,
						Type:        auditlog.ProjectResourceType,
						Meta:        map[string]any{"name": req.Msg.GetName(), "slug": req.Msg.GetSlug()},
						Name:        req.Msg.GetName(),
						DisplayName: req.Msg.GetName(),
					},
				},
			},
		}); txErr != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert audit log: %w", txErr))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.CreateProjectResponse{
		Id: projectID,
	}), nil
}
