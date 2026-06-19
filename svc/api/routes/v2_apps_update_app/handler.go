package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsUpdateAppRequestBody
	Response = openapi.V2AppsUpdateAppResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apps.updateApp"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	data, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.App, error) {
		app, err := db.Query.FindAppByWorkspaceAndId(ctx, tx, db.FindAppByWorkspaceAndIdParams{
			WorkspaceID: principal.WorkspaceID,
			ID:          req.AppId,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return openapi.App{}, fault.New(
					"app not found",
					fault.Code(codes.Data.App.NotFound.URN()),
					fault.Internal("app not found"),
					fault.Public("The requested app does not exist."),
				)
			}

			return openapi.App{}, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve app."),
			)
		}

		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Project,
				ResourceID:   "*",
				Action:       rbac.UpdateApp,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Project,
				ResourceID:   app.ProjectID,
				Action:       rbac.UpdateApp,
			}),
		))
		if err != nil {
			return openapi.App{}, err
		}

		updatedAt := time.Now().UnixMilli()
		update := db.UpdateAppParams{
			WorkspaceID:               principal.WorkspaceID,
			ID:                        app.ID,
			UpdatedAt:                 sql.NullInt64{Valid: true, Int64: updatedAt},
			NameSpecified:             0,
			Name:                      "",
			SlugSpecified:             0,
			Slug:                      "",
			DefaultBranchSpecified:    0,
			DefaultBranch:             "",
			DeleteProtectionSpecified: 0,
			DeleteProtection:          sql.NullBool{Valid: false, Bool: false},
		}

		name := app.Name
		if req.Name != nil {
			name = *req.Name
			update.Name = *req.Name
			update.NameSpecified = 1
		}

		slug := app.Slug
		if req.Slug != nil {
			slug = *req.Slug
			update.Slug = *req.Slug
			update.SlugSpecified = 1
		}

		defaultBranch := app.DefaultBranch
		if req.DefaultBranch != nil {
			defaultBranch = *req.DefaultBranch
			update.DefaultBranch = *req.DefaultBranch
			update.DefaultBranchSpecified = 1
		}

		deleteProtection := app.DeleteProtection.Bool
		if req.DeleteProtection != nil {
			deleteProtection = *req.DeleteProtection
			update.DeleteProtection = sql.NullBool{Valid: true, Bool: *req.DeleteProtection}
			update.DeleteProtectionSpecified = 1
		}

		if update.NameSpecified == 0 && update.SlugSpecified == 0 &&
			update.DefaultBranchSpecified == 0 && update.DeleteProtectionSpecified == 0 {
			return openapi.App{
				Id:                  app.ID,
				Name:                app.Name,
				Slug:                app.Slug,
				ProjectId:           app.ProjectID,
				DefaultBranch:       app.DefaultBranch,
				CurrentDeploymentId: app.CurrentDeploymentID.String,
				IsRolledBack:        app.IsRolledBack,
				DeleteProtection:    app.DeleteProtection.Bool,
				CreatedAt:           app.CreatedAt,
				UpdatedAt:           app.UpdatedAt.Int64,
			}, nil
		}

		err = db.Query.UpdateApp(ctx, tx, update)
		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return openapi.App{}, fault.Wrap(
					err,
					fault.Code(codes.Data.App.Duplicate.URN()),
					fault.Internal("app slug already exists in project"),
					fault.Public(fmt.Sprintf("An app with slug '%s' already exists in this project.", slug)),
				)
			}

			return openapi.App{}, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to update app"),
				fault.Public("We're unable to update the app."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.AppUpdateEvent,
				Display:       fmt.Sprintf("Updated app %s", app.ID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          app.ID,
						Type:        auditlog.AppResourceType,
						Meta:        map[string]any{"name": name, "slug": slug, "defaultBranch": defaultBranch, "deleteProtection": deleteProtection},
						Name:        name,
						DisplayName: name,
					},
				},
			},
		})
		if err != nil {
			return openapi.App{}, err
		}

		return openapi.App{
			Id:                  app.ID,
			Name:                name,
			Slug:                slug,
			ProjectId:           app.ProjectID,
			DefaultBranch:       defaultBranch,
			CurrentDeploymentId: app.CurrentDeploymentID.String,
			IsRolledBack:        app.IsRolledBack,
			DeleteProtection:    deleteProtection,
			CreatedAt:           app.CreatedAt,
			UpdatedAt:           updatedAt,
		}, nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
