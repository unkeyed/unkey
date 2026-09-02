package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/oapi-codegen/nullable"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	github "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/githubapp"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsUpdateAppRequestBody
	Response = openapi.V2AppsUpdateAppResponseBody
)

type Handler struct {
	DB         db.Database
	Auditlogs  auditlogs.AuditLogService
	CtrlClient ctrl.AppServiceClient

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
	gitSpecified := req.Git.IsSpecified()
	if req.Name == nil && req.Slug == nil && !gitSpecified && req.Oci == nil && req.DeleteProtection == nil {
		return fault.New(
			"no app updates provided",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("request has no update fields"),
			fault.Public("Provide at least one field to update."),
		)
	}
	if req.Oci != nil && (req.Name != nil || req.Slug != nil || gitSpecified || req.DeleteProtection != nil) {
		return fault.New(
			"OCI image update must be standalone",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("OCI image update was combined with another update field"),
			fault.Public("Update the OCI image in a separate request."),
		)
	}

	// Group the app.update and repository connect/disconnect events this request
	// emits under one correlation id.
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	data, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.App, error) {
		app, err := db.Query.FindAppByProjectAndIdOrSlug(ctx, tx, db.FindAppByProjectAndIdOrSlugParams{
			WorkspaceID: principal.WorkspaceID,
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
		if gitSpecified && app.SourceType == db.AppsSourceTypeOci {
			return openapi.App{}, fault.New(
				"git update is incompatible with app source",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("cannot update git configuration for an OCI-sourced app"),
				fault.Public("Git configuration cannot be updated for OCI-sourced apps."),
			)
		}
		if req.Oci != nil && app.SourceType != db.AppsSourceTypeOci {
			return openapi.App{}, fault.New(
				"image update is incompatible with app source",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("cannot update OCI image configuration for a non-OCI app"),
				fault.Public("OCI image configuration can only be updated for OCI-sourced apps."),
			)
		}
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
			WorkspaceID:               principal.WorkspaceID,
			ID:                        app.ID,
			UpdatedAt:                 sql.NullInt64{Valid: true, Int64: updatedAt},
			NameSpecified:             0,
			Name:                      "",
			SlugSpecified:             0,
			Slug:                      "",
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
		gitState, err := h.applyGitChange(ctx, tx, app, req.Git)
		if err != nil {
			return openapi.App{}, err
		}

		appColumnsChanged := update.NameSpecified == 1 || update.SlugSpecified == 1 || update.DeleteProtectionSpecified == 1

		responseUpdatedAt := app.UpdatedAt.Int64
		if appColumnsChanged {
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
				WorkspaceID:   principal.WorkspaceID,
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

		var oci *openapi.AppOCI
		if app.SourceType == db.AppsSourceTypeOci {
			imageReference := ""
			if req.Oci != nil {
				imageReference = req.Oci.Image
			} else {
				ociSource, ociErr := db.Query.FindAppSourceOciByAppId(ctx, tx, app.ID)
				if ociErr != nil {
					return openapi.App{}, ociErr
				}
				imageReference = ociSource.ImageReference
			}
			oci = &openapi.AppOCI{Image: imageReference}
		}
		sourceType := openapi.AppSourceType("")
		switch app.SourceType {
		case db.AppsSourceTypeGit:
			sourceType = openapi.Git
		case db.AppsSourceTypeOci:
			sourceType = openapi.Oci
		case db.AppsSourceTypeUnknown:
		}

		return openapi.App{
			Id:                  app.ID,
			Name:                name,
			Slug:                slug,
			SourceType:          sourceType,
			Git:                 gitState,
			Oci:                 oci,
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

	if req.Oci != nil {
		actor, actorErr := ctrlclient.Actor(s)
		if actorErr != nil {
			return actorErr
		}
		ctrlRes, ctrlErr := h.CtrlClient.UpdateOciImageSource(ctx, &ctrlv1.UpdateOciImageSourceRequest{
			WorkspaceId:    principal.WorkspaceID,
			AppId:          data.Id,
			ImageReference: req.Oci.Image,
			Actor:          actor,
		})
		if ctrlErr != nil {
			return ctrlclient.HandleError(ctrlErr, "update OCI image source")
		}
		data.Oci = &openapi.AppOCI{Image: ctrlRes.GetImageReference()}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}

func (h *Handler) applyGitChange(
	ctx context.Context,
	tx db.DBTX,
	app db.App,
	git nullable.Nullable[openapi.AppGitUpdateInput],
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
		return githubapp.GitResponse(conn.RepositoryFullName, conn.DefaultBranch.String), nil
	}

	if git.IsNull() {
		if err := db.Query.DeleteGithubRepoConnectionsByAppId(ctx, tx, app.ID); err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to disconnect github repo connection"),
				fault.Public("Failed to disconnect the GitHub repository."),
			)
		}
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
		rowsAffected, updateErr := db.Query.UpdateGithubRepoConnectionDefaultBranch(ctx, tx, db.UpdateGithubRepoConnectionDefaultBranchParams{
			DefaultBranch: sql.NullString{Valid: true, String: branch},
			UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			WorkspaceID:   app.WorkspaceID,
			AppID:         app.ID,
		})
		if updateErr != nil {
			return nil, fault.Wrap(
				updateErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to update github connection default branch"),
				fault.Public("Failed to update the branch tracked by this app."),
			)
		}
		if rowsAffected != 1 {
			return nil, fault.New(
				"repository connection changed during update",
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("expected to update one github connection, updated %d", rowsAffected)),
				fault.Public("Failed to update the branch tracked by this app."),
			)
		}
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
	if existing, connErr := db.Query.FindGithubRepoConnectionByAppId(ctx, tx, app.ID); connErr == nil {
		if existing.DefaultBranch.Valid && existing.DefaultBranch.String != "" {
			fallback = existing.DefaultBranch.String
		}
	} else if !db.IsNotFound(connErr) {
		return nil, fault.Wrap(
			connErr,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load github repo connection"),
			fault.Public("Failed to connect the GitHub repository."),
		)
	}
	branch := githubapp.DefaultBranch(fallback, requested.DefaultBranch)

	now := time.Now().UnixMilli()
	if err = db.Query.UpsertGithubRepoConnection(ctx, tx, db.UpsertGithubRepoConnectionParams{
		WorkspaceID:        app.WorkspaceID,
		ProjectID:          app.ProjectID,
		AppID:              app.ID,
		InstallationID:     resolved.InstallationID,
		RepositoryID:       resolved.Repository.ID,
		RepositoryFullName: resolved.Repository.FullName,
		DefaultBranch:      sql.NullString{Valid: true, String: branch},
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

	return githubapp.GitResponse(resolved.Repository.FullName, branch), nil
}
