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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ProjectsCreateProjectRequestBody
	Response = openapi.V2ProjectsCreateProjectResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/projects.createProject"
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

	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Project,
		ResourceID:   "*",
		Action:       rbac.CreateProject,
	}))
	if err != nil {
		return err
	}

	deleteProtection := false
	if req.DeleteProtection != nil {
		deleteProtection = *req.DeleteProtection
	}

	projectID := uid.New(uid.ProjectPrefix)
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.InsertProject(ctx, tx, db.InsertProjectParams{
			ID:               projectID,
			WorkspaceID:      principal.WorkspaceID,
			Name:             req.Name,
			Slug:             req.Slug,
			DeleteProtection: sql.NullBool{Valid: true, Bool: deleteProtection},
			CreatedAt:        time.Now().UnixMilli(),
			UpdatedAt:        sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return fault.New("project already exists",
					fault.Code(codes.Data.Project.Duplicate.URN()),
					fault.Internal("project slug already exists"),
					fault.Public(fmt.Sprintf("A project with slug '%s' already exists in this workspace.", req.Slug)),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to create project"), fault.Public("We're unable to create the project."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.ProjectCreateEvent,
				Display:       fmt.Sprintf("Created project %s", projectID),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          projectID,
						Type:        auditlog.ProjectResourceType,
						Meta:        map[string]any{"name": req.Name, "slug": req.Slug, "deleteProtection": deleteProtection},
						Name:        req.Name,
						DisplayName: req.Name,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2ProjectsCreateProjectResponseData{
			Id: projectID,
		},
	})
}
