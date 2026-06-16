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
	Request  = openapi.V2ProjectsUpdateProjectRequestBody
	Response = openapi.V2ProjectsUpdateProjectResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/projects.updateProject"
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

	data, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.Project, error) {
		project, err := db.Query.FindProjectByWorkspaceAndSlug(ctx, tx, db.FindProjectByWorkspaceAndSlugParams{
			WorkspaceID: principal.WorkspaceID,
			Slug:        req.Slug,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return openapi.Project{}, fault.New(
					"project not found",
					fault.Code(codes.Data.Project.NotFound.URN()),
					fault.Internal("project not found"),
					fault.Public("The requested project does not exist."),
				)
			}

			return openapi.Project{}, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve project."),
			)
		}

		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Project,
				ResourceID:   "*",
				Action:       rbac.UpdateProject,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Project,
				ResourceID:   project.ID,
				Action:       rbac.UpdateProject,
			}),
		))
		if err != nil {
			return openapi.Project{}, err
		}

		updatedAt := time.Now().UnixMilli()
		update := db.UpdateProjectParams{
			WorkspaceID:               principal.WorkspaceID,
			Slug:                      req.Slug,
			UpdatedAt:                 sql.NullInt64{Valid: true, Int64: updatedAt},
			NameSpecified:             0,
			Name:                      "",
			DeleteProtectionSpecified: 0,
			DeleteProtection:          sql.NullBool{Valid: false, Bool: false},
		}

		name := project.Name
		if req.Name != nil {
			name = *req.Name
			update.Name = *req.Name
			update.NameSpecified = 1
		}

		deleteProtection := project.DeleteProtection.Bool
		if req.DeleteProtection != nil {
			deleteProtection = *req.DeleteProtection
			update.DeleteProtection = sql.NullBool{Valid: true, Bool: *req.DeleteProtection}
			update.DeleteProtectionSpecified = 1
		}

		if update.NameSpecified == 0 && update.DeleteProtectionSpecified == 0 {
			return openapi.Project{
				Id:               project.ID,
				Name:             project.Name,
				Slug:             project.Slug,
				CreatedAt:        project.CreatedAt,
				UpdatedAt:        project.UpdatedAt.Int64,
				DeleteProtection: project.DeleteProtection.Bool,
			}, nil
		}

		err = db.Query.UpdateProject(ctx, tx, update)
		if err != nil {
			return openapi.Project{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to update project"),
				fault.Public("We're unable to update the project."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.ProjectUpdateEvent,
				Display:       fmt.Sprintf("Updated project %s", project.ID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          project.ID,
						Type:        auditlog.ProjectResourceType,
						Meta:        map[string]any{"name": name, "slug": project.Slug, "deleteProtection": deleteProtection},
						Name:        name,
						DisplayName: name,
					},
				},
			},
		})
		if err != nil {
			return openapi.Project{}, err
		}

		return openapi.Project{
			Id:               project.ID,
			Name:             name,
			Slug:             project.Slug,
			CreatedAt:        project.CreatedAt,
			UpdatedAt:        updatedAt,
			DeleteProtection: deleteProtection,
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
