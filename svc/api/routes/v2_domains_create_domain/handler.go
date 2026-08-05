package handler

import (
	"context"
	"net/http"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsCreateDomainRequestBody
	Response = openapi.V2DomainsCreateDomainResponseBody
)

type Handler struct {
	DB          db.Database
	CtrlClient  ctrl.CustomDomainServiceClient
	LimitsCache cache.Cache[string, keysdb.Limit]
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

	domainName, err := domaingate.ParseDomain(req.Domain)
	if err != nil {
		return err
	}

	// A found row means the domain is taken, so a nil error is the rejection here
	// and NotFound is the path that proceeds.
	_, err = db.Query.FindCustomDomainIDByWorkspaceAndDomain(ctx, h.DB.RO(), db.FindCustomDomainIDByWorkspaceAndDomainParams{
		WorkspaceID: principal.WorkspaceID,
		Domain:      domainName,
	})
	if err == nil {
		return domaingate.AlreadyExists(domainName)
	}
	if !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to check whether the domain is already attached."),
		)
	}

	limits, hit, err := h.LimitsCache.SWR(ctx, principal.WorkspaceID, func(ctx context.Context) (keysdb.Limit, error) {
		return keysdb.Query.FindLimitsByWorkspaceID(ctx, h.DB.RO(), principal.WorkspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to read the workspace's resource limits."),
		)
	}
	if db.IsNotFound(err) || hit == cache.Null {
		return domaingate.LimitsNotConfigured(principal.WorkspaceID)
	}

	attached, err := db.Query.CountCustomDomainsByWorkspace(ctx, h.DB.RO(), principal.WorkspaceID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to count the workspace's custom domains."),
		)
	}

	if err = domaingate.CheckAllowance(attached, limits.CustomDomainsMax); err != nil {
		return err
	}

	res, err := h.CtrlClient.AddCustomDomain(ctx, &ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   principal.WorkspaceID,
		ProjectId:     env.ProjectID,
		AppId:         env.AppID,
		EnvironmentId: env.ID,
		Domain:        domainName,
		Actor:         actor,
	})
	if err != nil {
		return customdomain.MapCtrlError(err, "create custom domain")
	}

	data := openapi.V2DomainsCreateDomainResponseData{
		DomainId: res.GetDomainId(),
		// Verified flags stay false: nothing has been published yet, so neither
		// record can have been read back.
		DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
			Domain:            domainName,
			TargetCname:       res.GetTargetCname(),
			VerificationToken: res.GetVerificationToken(),
			RoutingVerified:   false,
			OwnershipVerified: false,
		}),
		DomainConnect: nil,
	}

	// Absent when the DNS provider does not support Domain Connect.
	if dc := res.GetDomainConnect(); dc != nil {
		data.DomainConnect = &openapi.DomainConnect{
			Provider: dc.GetProvider(),
			Url:      dc.GetUrl(),
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
