package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/services/environment"
)

// envSpec defines the slug and human-readable description for a default environment.
type envSpec struct {
	slug        string
	description string
}

// defaultEnvironments are the environments created automatically for every new app.
var defaultEnvironments = []envSpec{
	{slug: "production", description: "Production"},
	{slug: "preview", description: "Preview"},
}

// CreateApp creates an app with default environments and their settings in a
// single transaction.
func (s *Service) CreateApp(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateAppRequest],
) (*connect.Response[ctrlv1.CreateAppResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"),
		assert.NotEmpty(req.Msg.GetName(), "name is required"),
		assert.NotEmpty(req.Msg.GetSlug(), "slug is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	workspaceID := req.Msg.GetWorkspaceId()
	projectID := req.Msg.GetProjectId()
	appID := uid.New(uid.AppPrefix)
	now := time.Now().UnixMilli()

	err := db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		if txErr := queries.InsertApp(txCtx, db.InsertAppParams{
			ID:               appID,
			WorkspaceID:      workspaceID,
			ProjectID:        projectID,
			Name:             req.Msg.GetName(),
			Slug:             req.Msg.GetSlug(),
			DefaultBranch:    "main",
			DeleteProtection: sql.NullBool{Valid: false},
			CreatedAt:        now,
			UpdatedAt:        sql.NullInt64{Valid: false},
		}); txErr != nil {
			return fmt.Errorf("insert app: %w", txErr)
		}

		environmentSpecs := make([]environment.CreateSpec, 0, len(defaultEnvironments))
		for _, env := range defaultEnvironments {
			environmentSpecs = append(environmentSpecs, environment.CreateSpec{
				ID:          uid.New(uid.EnvironmentPrefix),
				Slug:        env.slug,
				Description: env.description,
			})
		}

		if txErr := environment.CreateMany(txCtx, tx, environment.CreateManyParams{
			App: environment.AppScope{
				WorkspaceID: workspaceID,
				ProjectID:   projectID,
				AppID:       appID,
			},
			Environments: environmentSpecs,
			Now:          now,
		}); txErr != nil {
			return fmt.Errorf("create default environments: %w", txErr)
		}

		a := req.Msg.GetActor()
		if txErr := s.auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   workspaceID,
				Event:         auditlog.AppCreateEvent,
				Display:       fmt.Sprintf("Created app %s", appID),
				ActorID:       a.GetId(),
				ActorName:     a.GetName(),
				ActorType:     actor.AuditType(a.GetType()),
				ActorMeta:     actor.Meta(a.GetMeta()),
				RemoteIP:      a.GetRemoteIp(),
				UserAgent:     a.GetUserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          appID,
						Type:        auditlog.AppResourceType,
						Meta:        map[string]any{"name": req.Msg.GetName(), "slug": req.Msg.GetSlug(), "projectId": projectID},
						Name:        req.Msg.GetName(),
						DisplayName: req.Msg.GetName(),
					},
				},
			},
		}); txErr != nil {
			return fmt.Errorf("insert audit log: %w", txErr)
		}

		return nil
	})
	if err != nil {
		if db.IsDuplicateKeyError(err) {
			return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("app with slug %q already exists in project", req.Msg.GetSlug()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create app: %w", err))
	}

	return connect.NewResponse(&ctrlv1.CreateAppResponse{
		Id: appID,
	}), nil
}
