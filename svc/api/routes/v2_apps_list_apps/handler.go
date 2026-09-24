package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/githubapp"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsListAppsRequestBody
	Response = openapi.V2AppsListAppsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apps.listApps"
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

	project, err := db.Query.FindProjectByIdOrSlug(ctx, h.DB.RO(), db.FindProjectByIdOrSlugParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		Project:     req.Project,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"project not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project not found"),
				fault.Public("The requested project does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve project."),
		)
	}

	legacyPermission := rbac.T(rbac.Tuple{
		ResourceType: rbac.App,
		ResourceID:   "*",
		Action:       rbac.ReadApp,
	})
	legacyAllowed := rbac.Check(legacyPermission, principal.Permissions) == nil
	collection := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(project.ID).App("*")
	collectionURN, err := urn.ParseV1(collection.String())
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Internal("invalid app collection resource"), fault.Public("Failed to retrieve apps."))
	}
	if !legacyAllowed && !rbac.HasPermissionIn(collectionURN, permissions.Read, principal.Permissions) {
		return fault.New(
			"project not found",
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking project existence"),
			fault.Public("The requested project does not exist."),
		)
	}

	p := pagination.Parse(req.Limit, req.Cursor, 100)
	search := mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))

	rows, err := pagination.FetchAuthorized(ctx, p, func(ctx context.Context, cursor string, limit int32) ([]db.ListAppsByProjectRow, error) {
		return db.Query.ListAppsByProject(ctx, h.DB.RO(), db.ListAppsByProjectParams{
			ProjectID: project.ID,
			IDCursor:  cursor,
			Search:    search,
			Limit:     limit,
		})
	}, func(row db.ListAppsByProjectRow) bool {
		if legacyAllowed {
			return true
		}
		resource := urn.New().Workspace(row.WorkspaceID).Project(row.ProjectID).App(row.ID)
		return rbac.Check(rbac.U(resource, permissions.Read), principal.Permissions) == nil
	}, func(row db.ListAppsByProjectRow) string { return row.ID })
	if errors.Is(err, pagination.ErrScanLimit) {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("authorized app scan limit exceeded"),
			fault.Public("The app scan limit was reached. Narrow the search filter and retry."),
		)
	}
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve apps."),
		)
	}

	rows, pg := pagination.Paginate(rows, p, func(r db.ListAppsByProjectRow) string { return r.ID })
	data := make([]openapi.App, len(rows))
	for i, row := range rows {
		var oci *openapi.AppOCI
		sourceType := openapi.AppSourceType("")
		switch row.SourceType {
		case db.AppsSourceTypeGit:
			sourceType = openapi.Git
		case db.AppsSourceTypeOci:
			if !row.OciImageReference.Valid {
				return fault.New(
					"OCI app source is missing",
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("OCI app has no configured image source"),
					fault.Public("Failed to retrieve apps."),
				)
			}
			sourceType = openapi.Oci
			oci = &openapi.AppOCI{Image: row.OciImageReference.String}
		case db.AppsSourceTypeUnknown:
		}
		data[i] = openapi.App{
			Id:                  row.ID,
			Name:                row.Name,
			Slug:                row.Slug,
			SourceType:          sourceType,
			Git:                 githubapp.GitResponse(row.RepositoryFullName.String, row.GithubDefaultBranch.String),
			Oci:                 oci,
			CurrentDeploymentId: row.CurrentDeploymentID.String,
			IsRolledBack:        row.IsRolledBack,
			DeleteProtection:    row.DeleteProtection.Bool,
			CreatedAt:           row.CreatedAt,
			UpdatedAt:           row.UpdatedAt.Int64,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       data,
		Pagination: pg,
	})
}
