package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	github "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/githubapp"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsCreateAppRequestBody
	Response = openapi.V2AppsCreateAppResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.AppServiceClient
	Auditlogs  auditlogs.AuditLogService

	// GitHubClient resolves and verifies repositories for the optional `git`
	// connection. GitHubAppName is the App slug used to build actionable install
	// URLs in error messages; empty means GitHub connection is not configured.
	GitHubClient  github.GitHubClient
	GitHubAppName string
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apps.createApp"
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

	// Tag the repository-connect audit event with a correlation id so it can be
	// traced back to this request. The app.create event is emitted separately by
	// ctrl and is not part of this correlation.
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	project, err := db.Query.FindProjectByIdOrSlug(ctx, h.DB.RO(), db.FindProjectByIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
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

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.CreateApp,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   project.ID,
			Action:       rbac.CreateApp,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(project.ID).App("*"),
			permissions.CreateApp{},
		),
	))
	if err != nil {
		return err
	}

	var resolved githubapp.Resolved
	if req.Git != nil {
		if err = principal.Authorize(rbac.T(rbac.Tuple{
			ResourceType: rbac.App,
			ResourceID:   "*",
			Action:       rbac.ConnectRepository,
		})); err != nil {
			return err
		}

		if h.GitHubAppName == "" {
			return fault.New(
				"github not configured",
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("github app credentials are not configured for this deployment"),
				fault.Public("GitHub repository connection is not enabled."),
			)
		}

		installations, iErr := db.Query.FindGithubAppInstallationsByWorkspaceId(ctx, h.DB.RO(), principal.WorkspaceID)
		if iErr != nil {
			return fault.Wrap(
				iErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to load github installations"),
				fault.Public("Failed to connect the GitHub repository."),
			)
		}

		resolved, err = githubapp.Resolve(h.GitHubClient, h.GitHubAppName, installations, req.Git.Repository)
		if err != nil {
			return err
		}
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}
	res, err := h.CtrlClient.CreateApp(ctx, &ctrlv1.CreateAppRequest{
		WorkspaceId: principal.WorkspaceID,
		ProjectId:   project.ID,
		Name:        req.Name,
		Slug:        req.Slug,
		Actor:       actor,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return fault.New(
				"app already exists",
				fault.Code(codes.Data.App.Duplicate.URN()),
				fault.Internal("app slug already exists in project"),
				fault.Public(fmt.Sprintf("An app with slug '%s' already exists in this project.", req.Slug)),
			)
		}
		return ctrlclient.HandleError(err, "create app")
	}

	appID := res.GetId()

	// If the connection write fails, the app stays created but unconnected. That
	// is a valid repo-less app the caller can attach later via apps.updateApp, so
	// we surface the error rather than roll the app back.
	if req.Git != nil {
		// A create is always a fresh connect, so it adopts the repository's GitHub
		// default branch unless the caller passes one.
		defaultBranch := githubapp.DefaultBranch(resolved.Repository.DefaultBranch, req.Git.DefaultBranch)

		err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			now := time.Now().UnixMilli()
			if txErr := db.Query.UpsertGithubRepoConnection(ctx, tx, db.UpsertGithubRepoConnectionParams{
				WorkspaceID:        principal.WorkspaceID,
				ProjectID:          project.ID,
				AppID:              appID,
				InstallationID:     resolved.InstallationID,
				RepositoryID:       resolved.Repository.ID,
				RepositoryFullName: resolved.Repository.FullName,
				DefaultBranch:      sql.NullString{String: defaultBranch, Valid: true},
				CreatedAt:          now,
				UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
			}); txErr != nil {
				return fault.Wrap(
					txErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("failed to upsert github repo connection"),
					fault.Public("Failed to connect the GitHub repository."),
				)
			}

			if txErr := db.Query.UpdateApp(ctx, tx, db.UpdateAppParams{
				WorkspaceID:               principal.WorkspaceID,
				ID:                        appID,
				UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
				NameSpecified:             0,
				Name:                      "",
				SlugSpecified:             0,
				Slug:                      "",
				DefaultBranchSpecified:    1,
				DefaultBranch:             defaultBranch,
				DeleteProtectionSpecified: 0,
				DeleteProtection:          sql.NullBool{Valid: false, Bool: false},
			}); txErr != nil {
				return fault.Wrap(
					txErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("failed to set app default branch"),
					fault.Public("Failed to connect the GitHub repository."),
				)
			}

			return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
				{
					WorkspaceID:   principal.WorkspaceID,
					Event:         auditlog.AppConnectRepositoryEvent,
					Display:       fmt.Sprintf("Connected app %s to %s", appID, resolved.Repository.FullName),
					ActorID:       principal.Subject.ID,
					ActorName:     principal.Subject.Name,
					ActorMeta:     map[string]any{},
					ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
					RemoteIP:      s.Location(),
					UserAgent:     s.UserAgent(),
					CorrelationID: "",
					Resources: []auditlog.AuditLogResource{
						{
							ID:          appID,
							Type:        auditlog.AppResourceType,
							Meta:        map[string]any{"repository": resolved.Repository.FullName, "defaultBranch": defaultBranch},
							Name:        resolved.Repository.FullName,
							DisplayName: resolved.Repository.FullName,
						},
					},
				},
			})
		})
		if err != nil {
			return err
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2AppsCreateAppResponseData{
			AppId: appID,
		},
	})
}
