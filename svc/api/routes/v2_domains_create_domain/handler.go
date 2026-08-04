package handler

import (
	"context"
	"net/http"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dns"
	"github.com/unkeyed/unkey/pkg/dns/domainconnect"
	"github.com/unkeyed/unkey/pkg/dns/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
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

	domain := strings.ToLower(req.Domain)

	if err = domaingate.CheckDomain(domain); err != nil {
		return err
	}

	_, err = db.Query.FindCustomDomainIDByWorkspaceAndDomain(ctx, h.DB.RO(), db.FindCustomDomainIDByWorkspaceAndDomainParams{
		WorkspaceID: principal.WorkspaceID,
		Domain:      domain,
	})
	switch {
	case err == nil:
		return domaingate.AlreadyAttached(domain)
	case !db.IsNotFound(err):
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to check whether the domain is available."),
		)
	}

	allowed, err := db.Query.FindCustomDomainsMaxByWorkspaceID(ctx, h.DB.RO(), principal.WorkspaceID)
	if err != nil {
		if db.IsNotFound(err) {
			return domaingate.LimitsNotConfigured(principal.WorkspaceID)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to read the workspace's resource limits."),
		)
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

	if err = domaingate.CheckAllowance(attached, allowed); err != nil {
		return err
	}

	res, err := h.CtrlClient.AddCustomDomain(ctx, &ctrlv1.AddCustomDomainRequest{
		WorkspaceId:   principal.WorkspaceID,
		ProjectId:     env.ProjectID,
		AppId:         env.AppID,
		EnvironmentId: env.ID,
		Domain:        domain,
		Actor:         actor,
	})
	if err != nil {
		return customdomain.MapCtrlError(err, "create custom domain")
	}

	routing := openapi.DnsRecord{
		Type:  openapi.CNAME,
		Name:  domain,
		Value: res.GetTargetCname(),
		Note:  ptr.P("Create as DNS-only. A proxied record is flattened and cannot be read back."),
	}
	txt := openapi.DnsRecord{
		Type:  openapi.TXT,
		Name:  dns.OwnershipTXTName(domain),
		Value: dns.OwnershipTXTValue(res.GetVerificationToken()),
		Note:  ptr.P("Proves ownership. Needed whenever the routing record cannot be read back, which is not knowable until it is published."),
	}

	if domainconnect.IsApexDomain(domain) {
		routing.Type = openapi.ALIAS
		routing.Note = ptr.P("Apex domains cannot hold a CNAME. Use ALIAS, ANAME, or a flattened CNAME depending on your provider.")
		txt.Note = ptr.P("Proves ownership. An apex domain cannot be verified through its routing record, so this is the only proof available.")
	}

	data := openapi.V2DomainsCreateDomainResponseData{
		DomainId:      res.GetDomainId(),
		DnsRecords:    []openapi.DnsRecord{routing, txt},
		DomainConnect: nil,
	}

	// Discovery inside ctrl is best-effort and yields a provider and a URL together or
	// neither, so a half-filled object would mean ctrl changed, not that the shortcut is
	// partly available. Both are required in the schema, hence the pair check.
	provider, dcURL := res.GetDomainConnectProvider(), res.GetDomainConnectUrl()
	if provider != "" && dcURL != "" {
		data.DomainConnect = &openapi.DomainConnect{
			Provider: provider,
			Url:      dcURL,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
