package handler

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsCreateDomainRequestBody
	Response = openapi.V2DomainsCreateDomainResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.CustomDomainServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/domains.createDomain"
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

	env, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment not found"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}

	// A caller that may not read the environment must not be able to tell a real
	// one from a missing one, so the rejection becomes the same 404.
	if err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.CreateDomain,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   env.ID,
			Action:       rbac.CreateDomain,
		}),
	)); err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Environment.NotFound.URN(),
			"The requested environment does not exist.",
		)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	res, err := h.CtrlClient.AddCustomDomain(ctx, &ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   principal.WorkspaceID,
		ProjectId:     env.ProjectID,
		AppId:         env.AppID,
		EnvironmentId: env.ID,
		Domain:        req.Domain,
		Actor:         actor,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return fault.New(
				"domain already exists",
				fault.Code(codes.Data.Domain.Duplicate.URN()),
				fault.Internal("domain already registered in workspace"),
				fault.Public(fmt.Sprintf("The domain '%s' is already registered in this workspace.", req.Domain)),
			)
		}
		return ctrlclient.HandleError(err, "create custom domain")
	}

	// Domain Connect discovery is best-effort inside ctrl, so both fields are
	// absent whenever the provider does not support it.
	data := openapi.V2DomainsCreateDomainResponseData{
		DomainId:              res.GetDomainId(),
		TargetCname:           res.GetTargetCname(),
		VerificationToken:     res.GetVerificationToken(),
		DomainConnectProvider: nil,
		DomainConnectUrl:      nil,
	}
	if p := res.GetDomainConnectProvider(); p != "" {
		data.DomainConnectProvider = ptr.P(p)
	}
	if u := res.GetDomainConnectUrl(); u != "" {
		data.DomainConnectUrl = ptr.P(u)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
