package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	github "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/githubapp"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsUpdateAppRequestBody
	Response = openapi.V2AppsUpdateAppResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService

	// GitHubClient resolves and verifies repositories for the `git` connection.
	// GitHubAppName is the App slug used to build actionable install URLs in
	// error messages; empty means GitHub connection is not configured.
	GitHubClient  github.GitHubClient
	GitHubAppName string
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

	// Group the app.update and repository connect/disconnect events this request
	// emits under one correlation id.
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	data, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.App, error) {
		app, err := db.Query.FindAppByProjectAndIdOrSlug(ctx, tx, db.FindAppByProjectAndIdOrSlugParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Project:     req.Project,
			App:         req.App,
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
				ResourceType: rbac.App,
				ResourceID:   "*",
				Action:       rbac.UpdateApp,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.App,
				ResourceID:   app.ID,
				Action:       rbac.UpdateApp,
			}),
		))
		if err != nil {
			return openapi.App{}, err
		}

		// connect_repository gates every git change, disconnect included.
		gitSpecified := req.Git.IsSpecified()
		if gitSpecified {
			err = principal.Authorize(rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.App,
					ResourceID:   "*",
					Action:       rbac.ConnectRepository,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.App,
					ResourceID:   app.ID,
					Action:       rbac.ConnectRepository,
				}),
			))
			if err != nil {
				return openapi.App{}, err
			}
		}

		updatedAt := time.Now().UnixMilli()
		update := db.UpdateAppParams{
			WorkspaceID:               principal.AuthorizedWorkspaceID,
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

		deleteProtection := app.DeleteProtection.Bool
		if req.DeleteProtection != nil {
			deleteProtection = *req.DeleteProtection
			update.DeleteProtection = sql.NullBool{Valid: true, Bool: *req.DeleteProtection}
			update.DeleteProtectionSpecified = 1
		}

		// gitState is the connection echoed in the response: current when git is
		// unspecified, nil on disconnect, the new repository on connect.
		gitState, err := h.applyGitChange(ctx, tx, app, req.Git, &update)
		if err != nil {
			return openapi.App{}, err
		}

		// appColumnsChanged tracks user-facing app fields (an `app.update` event);
		// a git connect also writes the default branch, but that is audited under
		// the `app.connect_repository` event instead.
		appColumnsChanged := update.NameSpecified == 1 || update.SlugSpecified == 1 || update.DeleteProtectionSpecified == 1

		// Persist the app row only when a user field or the default branch changed.
		// updatedAt is reflected in the response only when a write actually happened.
		responseUpdatedAt := app.UpdatedAt.Int64
		if appColumnsChanged || update.DefaultBranchSpecified == 1 {
			if err = db.Query.UpdateApp(ctx, tx, update); err != nil {
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
			responseUpdatedAt = updatedAt
		}

		logs := make([]auditlog.AuditLog, 0, 2)
		if appColumnsChanged {
			logs = append(logs, auditlog.AuditLog{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
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
						Meta:        map[string]any{"name": name, "slug": slug, "deleteProtection": deleteProtection},
						Name:        name,
						DisplayName: name,
					},
				},
			})
		}
		if gitSpecified {
			event := auditlog.AppDisconnectRepositoryEvent
			display := fmt.Sprintf("Disconnected the GitHub repository from app %s", app.ID)
			meta := map[string]any{}
			resourceName := app.ID
			if gitState != nil {
				event = auditlog.AppConnectRepositoryEvent
				display = fmt.Sprintf("Connected app %s to %s", app.ID, gitState.Repository)
				meta = map[string]any{"repository": gitState.Repository}
				if gitState.DefaultBranch != nil {
					meta["defaultBranch"] = *gitState.DefaultBranch
				}
				resourceName = gitState.Repository
			}
			logs = append(logs, auditlog.AuditLog{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         event,
				Display:       display,
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
						Meta:        meta,
						Name:        resourceName,
						DisplayName: resourceName,
					},
				},
			})
		}
		if len(logs) > 0 {
			if err = h.Auditlogs.Insert(ctx, tx, logs); err != nil {
				return openapi.App{}, err
			}
		}

		return openapi.App{
			Id:                  app.ID,
			Name:                name,
			Slug:                slug,
			SourceType:          "",
			Git:                 gitState,
			Oci:                 nil,
			CurrentDeploymentId: app.CurrentDeploymentID.String,
			IsRolledBack:        app.IsRolledBack,
			DeleteProtection:    deleteProtection,
			CreatedAt:           app.CreatedAt,
			UpdatedAt:           responseUpdatedAt,
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

// applyGitChange applies the `git` field to the app's repository connection and
// returns the connection state to reflect in the response. It also stamps the
// resulting default branch onto `update` (the resolved or overridden branch on
// connect/retarget, empty on disconnect) so the app row is written in the same
// transaction.
func (h *Handler) applyGitChange(
	ctx context.Context,
	tx db.DBTX,
	app db.App,
	git nullable.Nullable[openapi.AppGitUpdateInput],
	update *db.UpdateAppParams,
) (*openapi.AppGit, error) {

	if !git.IsSpecified() {
		// No change requested: reflect the app's current connection, if any.
		conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, tx, app.ID)
		if err != nil {
			if db.IsNotFound(err) {
				return githubapp.GitResponse("", ""), nil
			}
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to load github repo connection"),
				fault.Public("Failed to retrieve app."),
			)
		}
		return githubapp.GitResponse(conn.RepositoryFullName, app.DefaultBranch), nil
	}

	if git.IsNull() {
		// Disconnect: drop the connection and clear the tracked branch, which has
		// no meaning without a repository.
		if err := db.Query.DeleteGithubRepoConnectionsByAppId(ctx, tx, app.ID); err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to disconnect github repo connection"),
				fault.Public("Failed to disconnect the GitHub repository."),
			)
		}
		update.DefaultBranch = ""
		update.DefaultBranchSpecified = 1
		return githubapp.GitResponse("", ""), nil
	}

	// Connect, replace, or retarget the tracked branch.
	requested := git.MustGet()

	if requested.Repository == nil {
		// Branch-only change: retarget the currently connected repository. Its
		// identity does not change, so no GitHub lookup is needed.
		conn, err := db.Query.FindGithubRepoConnectionByAppId(ctx, tx, app.ID)
		if err != nil {
			if db.IsNotFound(err) {
				return nil, fault.New(
					"no repository connected",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("cannot set a branch without a connected repository"),
					fault.Public("Connect a GitHub repository before setting the branch it tracks."),
				)
			}
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to load github repo connection"),
				fault.Public("Failed to retrieve app."),
			)
		}

		branch := *requested.DefaultBranch
		update.DefaultBranch = branch
		update.DefaultBranchSpecified = 1
		return githubapp.GitResponse(conn.RepositoryFullName, branch), nil
	}

	if h.GitHubAppName == "" {
		return nil, fault.New(
			"github not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("github app credentials are not configured for this deployment"),
			fault.Public("GitHub repository connection is not enabled."),
		)
	}

	installations, err := db.Query.FindGithubAppInstallationsByWorkspaceId(ctx, tx, app.WorkspaceID)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load github installations"),
			fault.Public("Failed to connect the GitHub repository."),
		)
	}

	resolved, err := githubapp.Resolve(h.GitHubClient, h.GitHubAppName, installations, *requested.Repository)
	if err != nil {
		return nil, err
	}

	// A fresh connect adopts the repository's GitHub default branch; replacing an
	// already-connected repository keeps the branch the app currently tracks, so
	// swapping the repository never silently retargets it. An explicit
	// defaultBranch always wins over both.
	fallback := resolved.Repository.DefaultBranch
	if _, connErr := db.Query.FindGithubRepoConnectionByAppId(ctx, tx, app.ID); connErr == nil {
		fallback = app.DefaultBranch
	} else if !db.IsNotFound(connErr) {
		return nil, fault.Wrap(
			connErr,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load github repo connection"),
			fault.Public("Failed to connect the GitHub repository."),
		)
	}
	branch := githubapp.DefaultBranch(fallback, requested.DefaultBranch)
	update.DefaultBranch = branch
	update.DefaultBranchSpecified = 1

	now := time.Now().UnixMilli()
	if err = db.Query.UpsertGithubRepoConnection(ctx, tx, db.UpsertGithubRepoConnectionParams{
		WorkspaceID:        app.WorkspaceID,
		ProjectID:          app.ProjectID,
		AppID:              app.ID,
		InstallationID:     resolved.InstallationID,
		RepositoryID:       resolved.Repository.ID,
		RepositoryFullName: resolved.Repository.FullName,
		DefaultBranch:      sql.NullString{String: branch, Valid: true},
		CreatedAt:          now,
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
	}); err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to upsert github repo connection"),
			fault.Public("Failed to connect the GitHub repository."),
		)
	}

	update.DefaultBranch = branch
	update.DefaultBranchSpecified = 1

	return githubapp.GitResponse(resolved.Repository.FullName, branch), nil
}
