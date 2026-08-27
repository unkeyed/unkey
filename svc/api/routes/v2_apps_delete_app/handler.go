package handler

import (
	"context"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsDeleteAppRequestBody
	Response = openapi.V2AppsDeleteAppResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apps.deleteApp"
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

	app, err := db.Query.FindAppByProjectAndIdOrSlug(ctx, h.DB.RW(), db.FindAppByProjectAndIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"app not found",
				fault.Code(codes.Data.App.NotFound.URN()),
				fault.Internal("app not found"),
				fault.Public("The requested app does not exist."),
			)
		}

		return fault.Wrap(
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
			Action:       rbac.DeleteApp,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.App,
			ResourceID:   app.ID,
			Action:       rbac.DeleteApp,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(app.ProjectID).App(app.ID),
			permissions.DeleteApp{},
		),
	))
	if err != nil {
		return err
	}

	if app.DeleteProtection.Valid && app.DeleteProtection.Bool {
		return fault.New(
			"delete protected",
			fault.Code(codes.App.Protection.ProtectedResource.URN()),
			fault.Internal("app is protected from deletion"),
			fault.Public("This app has delete protection enabled. Disable it before attempting to delete."),
		)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	// Deletion cascades through the app's environments and deployments. Send
	// returns once Restate accepts the durable workflow; teardown remains
	// eventually consistent.
	_, err = hydrav1.NewAppServiceIngressClient(h.Restate, app.ID).
		Delete().
		Send(ctx, &hydrav1.DeleteAppRequest{
			Actor:         actor,
			CorrelationId: auditlog.NewCorrelationID(),
		})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit app deletion to Restate"),
			fault.Public("Failed to delete app."),
		)
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
