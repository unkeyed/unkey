// Package portaldomain implements the PortalDomainService ConnectRPC API, which
// adds, deletes, and retries verification of the hostnames tenants attach to
// their portal.
package portaldomain

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// maxPortalDomainsPerWorkspace is a fixed cap until billing defines per-tier
// portal domain allowances.
const maxPortalDomainsPerWorkspace = 10

// Service implements the PortalDomainService ConnectRPC API. It persists
// portal domains and delegates their verification to the Restate
// PortalDomainService virtual object.
type Service struct {
	ctrlv1connect.UnimplementedPortalDomainServiceHandler
	db            db.Database
	restate       *restateingress.Client
	restateAdmin  *restateadmin.Client
	auditlogs     auditlogs.AuditLogService
	cnameDomain   string
	environmentID string
	bearer        string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	Database db.Database
	// Restate is the ingress client that starts verification workflows.
	Restate *restateingress.Client
	// RestateAdmin cancels verification workflows. Nil skips cancellation.
	RestateAdmin *restateadmin.Client
	// Auditlogs records mutations within the same transaction as the write.
	Auditlogs auditlogs.AuditLogService
	// CnameDomain is the base domain for portal CNAME targets. Empty makes
	// AddPortalDomain fail with FailedPrecondition.
	CnameDomain string
	// EnvironmentID is the portal app's production environment. Empty makes
	// AddPortalDomain fail with FailedPrecondition, and it scopes the route a
	// verified domain's deletion removes.
	EnvironmentID string
	// Bearer is the preshared token callers must send in the Authorization header.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedPortalDomainServiceHandler: ctrlv1connect.UnimplementedPortalDomainServiceHandler{},
		db:                                      cfg.Database,
		restate:                                 cfg.Restate,
		restateAdmin:                            cfg.RestateAdmin,
		auditlogs:                               cfg.Auditlogs,
		cnameDomain:                             cfg.CnameDomain,
		environmentID:                           cfg.EnvironmentID,
		bearer:                                  cfg.Bearer,
	}
}

// AddPortalDomain creates a portal domain and starts its verification workflow.
func (s *Service) AddPortalDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.AddPortalDomainRequest],
) (*connect.Response[ctrlv1.AddPortalDomainResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetPortalId(), "portal_id is required"),
		assert.NotEmpty(req.Msg.GetDomain(), "domain is required"),
	); err != nil {
		// Not InvalidArgument, which is reserved for gatefault codes the API reflects.
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if s.environmentID == "" || s.cnameDomain == "" {
		return nil, gatefault.Connect(portalDomainsNotConfigured())
	}

	domain, parseErr := domaingate.ParseDomain(req.Msg.GetDomain())
	if parseErr != nil {
		return nil, gatefault.ConnectWith(connect.CodeInvalidArgument, parseErr)
	}

	workspaceID := req.Msg.GetWorkspaceId()

	_, err := s.db.FindPortalDomainByWorkspaceAndDomain(ctx, db.FindPortalDomainByWorkspaceAndDomainParams{
		WorkspaceID: workspaceID,
		Domain:      domain,
	})
	if err == nil {
		return nil, gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyExists(domain))
	}
	if !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing portal domain: %w", err))
	}

	// Concurrent creates can both pass, overshooting by at most the request
	// concurrency, which beats serializing every create in a workspace.
	attached, err := s.db.CountPortalDomainsByWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count portal domains: %w", err))
	}
	if attached >= maxPortalDomainsPerWorkspace {
		return nil, gatefault.ConnectWith(connect.CodeResourceExhausted, portalDomainCapReached(attached))
	}

	targetCname := fmt.Sprintf("%s.%s", uid.DNS1035(16), s.cnameDomain)
	verificationToken := uid.Secure(24)
	domainID := uid.New(uid.PortalDomainPrefix)
	now := time.Now().UnixMilli()

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		q := db.NewQueries(tx)

		// One hostname routes one way, so a workspace cannot hold it as both a
		// deploy domain and a portal domain. No unique index spans the two tables.
		_, txErr := q.FindCustomDomainIDByWorkspaceAndDomain(txCtx, db.FindCustomDomainIDByWorkspaceAndDomainParams{
			WorkspaceID: workspaceID,
			Domain:      domain,
		})
		if txErr == nil {
			return gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyExists(domain))
		}
		if !db.IsNotFound(txErr) {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("check custom domain: %w", txErr))
		}

		if txErr := q.InsertPortalDomain(txCtx, db.InsertPortalDomainParams{
			ID:                    domainID,
			WorkspaceID:           workspaceID,
			PortalID:              req.Msg.GetPortalId(),
			Domain:                domain,
			VerificationStatus:    db.PortalDomainsVerificationStatusPending,
			VerificationToken:     verificationToken,
			TargetCname:           targetCname,
			DomainConnectProvider: sql.NullString{Valid: false, String: ""},
			DomainConnectUrl:      sql.NullString{Valid: false, String: ""},
			InvocationID:          sql.NullString{Valid: false, String: ""},
			CreatedAt:             now,
		}); txErr != nil {
			if db.IsDuplicateKeyError(txErr) {
				return gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyExists(domain))
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("insert portal domain: %w", txErr))
		}

		if txErr := s.audit(txCtx, tx, req.Msg.GetActor(), workspaceID, auditlog.PortalDomainCreateEvent,
			fmt.Sprintf("Added portal domain %s", domain), domainID, req.Msg.GetPortalId(), domain); txErr != nil {
			return connect.NewError(connect.CodeInternal, txErr)
		}

		// Submitting inside the transaction makes the RPC all-or-nothing: a failed
		// submit rolls back the row and its audit entry. The workflow tolerates
		// reading before the commit lands, and a TxRetry rerun re-submitting is
		// safe because the virtual object is keyed by domainID.
		sendResp, sendErr := s.startVerification(txCtx, domainID)
		if sendErr != nil {
			logger.Error("failed to trigger portal domain verification",
				"domain", domain,
				"domain_id", domainID,
				"error", sendErr,
			)
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger verification workflow: %w", sendErr))
		}

		if txErr := q.UpdatePortalDomainInvocationID(txCtx, db.UpdatePortalDomainInvocationIDParams{
			ID:           domainID,
			InvocationID: sql.NullString{Valid: true, String: sendResp.Id()},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: now},
		}); txErr != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("record invocation id: %w", txErr))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.AddPortalDomainResponse{
		DomainId:          domainID,
		TargetCname:       targetCname,
		Status:            ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
		VerificationToken: verificationToken,
	}), nil
}

// DeletePortalDomain stops verification and deletes the domain, its ACME
// challenge, and, when it is the verified row, its route. The certificate stays.
func (s *Service) DeletePortalDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeletePortalDomainRequest],
) (*connect.Response[ctrlv1.DeletePortalDomainResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	dom, err := s.findOwned(ctx, req.Msg.GetWorkspaceId(), req.Msg.GetPortalId(), req.Msg.GetDomainId())
	if err != nil {
		return nil, err
	}

	s.cancelVerification(ctx, dom)

	// Every tenant's route shares the portal project, so the project scope
	// cannot tell tenants apart. Only the verified row for a hostname owns its
	// route; deleting an unverified claim must leave another tenant's in place.
	routeProjectID := ""
	if dom.VerificationStatus == db.PortalDomainsVerificationStatusVerified {
		env, envErr := s.db.FindEnvironmentById(ctx, s.environmentID)
		if envErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find portal environment: %w", envErr))
		}
		routeProjectID = env.ProjectID
	}

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		q := db.NewQueries(tx)

		if routeProjectID != "" {
			if txErr := q.DeleteFrontlineRouteByFQDNAndProject(txCtx, db.DeleteFrontlineRouteByFQDNAndProjectParams{
				Fqdn:      dom.Domain,
				ProjectID: routeProjectID,
			}); txErr != nil && !db.IsNotFound(txErr) {
				return fmt.Errorf("failed to delete frontline route: %w", txErr)
			}
		}

		if txErr := q.DeleteAcmeChallengeByDomainID(txCtx, dom.ID); txErr != nil && !db.IsNotFound(txErr) {
			return fmt.Errorf("failed to delete ACME challenge: %w", txErr)
		}

		if txErr := q.DeletePortalDomainByID(txCtx, dom.ID); txErr != nil {
			return fmt.Errorf("failed to delete portal domain: %w", txErr)
		}

		return s.audit(txCtx, tx, req.Msg.GetActor(), dom.WorkspaceID, auditlog.PortalDomainDeleteEvent,
			fmt.Sprintf("Deleted portal domain %s", dom.Domain), dom.ID, dom.PortalID, dom.Domain)
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.DeletePortalDomainResponse{}), nil
}

// RetryVerification resets and restarts verification of a pending, verifying,
// or failed portal domain. A verified domain is rejected with FailedPrecondition.
func (s *Service) RetryVerification(
	ctx context.Context,
	req *connect.Request[ctrlv1.RetryPortalDomainVerificationRequest],
) (*connect.Response[ctrlv1.RetryPortalDomainVerificationResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	dom, err := s.findOwned(ctx, req.Msg.GetWorkspaceId(), req.Msg.GetPortalId(), req.Msg.GetDomainId())
	if err != nil {
		return nil, err
	}

	if dom.VerificationStatus == db.PortalDomainsVerificationStatusVerified {
		return nil, gatefault.ConnectWith(connect.CodeFailedPrecondition, domaingate.AlreadyVerified(dom.Domain))
	}

	s.cancelVerification(ctx, dom)

	now := time.Now().UnixMilli()

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := s.audit(txCtx, tx, req.Msg.GetActor(), dom.WorkspaceID, auditlog.PortalDomainVerifyEvent,
			fmt.Sprintf("Retried verification for portal domain %s", dom.Domain), dom.ID, dom.PortalID, dom.Domain); txErr != nil {
			return txErr
		}

		sendResp, sendErr := s.startVerification(txCtx, dom.ID)
		if sendErr != nil {
			return fmt.Errorf("failed to trigger verification: %w", sendErr)
		}

		if txErr := db.NewQueries(tx).ResetPortalDomainVerification(txCtx, db.ResetPortalDomainVerificationParams{
			ID:                 dom.ID,
			VerificationStatus: db.PortalDomainsVerificationStatusPending,
			CheckAttempts:      0,
			InvocationID:       sql.NullString{Valid: true, String: sendResp.Id()},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		}); txErr != nil {
			return fmt.Errorf("failed to reset verification: %w", txErr)
		}

		return nil
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.RetryPortalDomainVerificationResponse{
		Status: ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
	}), nil
}

// findOwned loads a portal domain, answering NotFound when it belongs to
// another workspace or portal so the response cannot be used to probe ids.
func (s *Service) findOwned(ctx context.Context, workspaceID, portalID, domainID string) (db.PortalDomain, error) {
	dom, err := s.db.FindPortalDomainById(ctx, domainID)
	if err != nil {
		if db.IsNotFound(err) {
			return db.PortalDomain{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("portal domain not found: %s", domainID))
		}
		return db.PortalDomain{}, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find portal domain: %w", err))
	}
	if dom.WorkspaceID != workspaceID || dom.PortalID != portalID {
		return db.PortalDomain{}, connect.NewError(connect.CodeNotFound, fmt.Errorf("portal domain not found: %s", domainID))
	}
	return dom, nil
}

// cancelVerification cancels dom's running workflow, if any. A failed cancel
// is logged rather than returned: the workflow stops on its own once the row
// is gone or reset.
func (s *Service) cancelVerification(ctx context.Context, dom db.PortalDomain) {
	if !dom.InvocationID.Valid || s.restateAdmin == nil {
		return
	}
	if err := s.restateAdmin.CancelInvocation(ctx, dom.InvocationID.String); err != nil {
		logger.Warn("failed to cancel portal domain verification workflow",
			"domain", dom.Domain,
			"invocation_id", dom.InvocationID.String,
			"error", err,
		)
	}
}

// audit writes one portal domain audit entry inside tx.
func (s *Service) audit(
	ctx context.Context,
	tx db.DBTX,
	a *ctrlv1.ActorInfo,
	workspaceID string,
	event auditlog.AuditLogEvent,
	display, domainID, portalID, domain string,
) error {
	if err := s.auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
		{
			WorkspaceID:   workspaceID,
			Event:         event,
			Display:       display,
			ActorID:       a.GetId(),
			ActorName:     a.GetName(),
			ActorType:     actor.AuditType(a.GetType()),
			ActorMeta:     actor.Meta(a.GetMeta()),
			RemoteIP:      a.GetRemoteIp(),
			UserAgent:     a.GetUserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					ID:          domainID,
					Type:        auditlog.PortalDomainResourceType,
					Meta:        map[string]any{"domain": domain, "portalId": portalID},
					Name:        domain,
					DisplayName: domain,
				},
			},
		},
	}); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

// startVerification submits the verification workflow for domainID.
func (s *Service) startVerification(ctx context.Context, domainID string) (restateingress.SimpleSendResponse, error) {
	return hydrav1.NewPortalDomainServiceIngressClient(s.restate, domainID).
		VerifyDomain().
		Send(ctx, &hydrav1.VerifyPortalDomainRequest{})
}

// portalDomainsNotConfigured is the outcome when this environment has no
// portal app or portal CNAME domain to route a portal domain to.
func portalDomainsNotConfigured() error {
	return fault.New(
		"portal domains not configured",
		fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
		fault.Internal("ctrl api is missing [portal] environment_id or portal_cname_domain"),
		fault.Public("Portal domains are not available in this environment. Contact support@unkey.com."),
	)
}

// portalDomainCapReached is the outcome for a workspace already holding the
// maximum number of portal domains.
func portalDomainCapReached(attached int64) error {
	return fault.New(
		"portal domain cap reached",
		fault.Code(codes.Limits.CustomDomain.Exceeded.URN()),
		fault.Internal(fmt.Sprintf("workspace holds %d of %d allowed portal domains", attached, maxPortalDomainsPerWorkspace)),
		fault.Public(fmt.Sprintf("A workspace can hold at most %d portal domains. Remove a domain you no longer need, then retry.", maxPortalDomainsPerWorkspace)),
	)
}
