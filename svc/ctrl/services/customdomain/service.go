package customdomain

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/dns/domainconnect"
	"github.com/unkeyed/unkey/pkg/dns/domaingate"
	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// Service implements the CustomDomainService ConnectRPC API. It coordinates
// custom domain operations by persisting state to the database and delegating
// verification workflows to Restate.
type Service struct {
	ctrlv1connect.UnimplementedCustomDomainServiceHandler
	db                         db.Database
	restate                    *restateingress.Client
	restateAdmin               *restateadmin.Client
	auditlogs                  auditlogs.AuditLogService
	cnameDomain                string
	domainConnectPrivateKeyPEM []byte
	bearer                     string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to custom domain metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable verification workflows.
	Restate *restateingress.Client
	// RestateAdmin is the admin client for canceling invocations.
	RestateAdmin *restateadmin.Client
	// Auditlogs records custom domain mutations within the same transaction as the write.
	Auditlogs auditlogs.AuditLogService
	// CnameDomain is the base domain for custom domain CNAME targets.
	CnameDomain string
	// DomainConnectPrivateKeyPEM is the PEM-encoded RSA private key for signing
	// Domain Connect redirect URLs. If empty, Domain Connect is disabled.
	DomainConnectPrivateKeyPEM []byte
	// Bearer is the preshared token that callers must provide in the Authorization header.
	Bearer string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedCustomDomainServiceHandler: ctrlv1connect.UnimplementedCustomDomainServiceHandler{},
		db:                                      cfg.Database,
		restate:                                 cfg.Restate,
		restateAdmin:                            cfg.RestateAdmin,
		auditlogs:                               cfg.Auditlogs,
		cnameDomain:                             cfg.CnameDomain,
		domainConnectPrivateKeyPEM:              cfg.DomainConnectPrivateKeyPEM,
		bearer:                                  cfg.Bearer,
	}
}

// AddCustomDomain creates a new custom domain and starts the verification workflow.
func (s *Service) AddCustomDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.AddCustomDomainRequest],
) (*connect.Response[ctrlv1.AddCustomDomainResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"),
		assert.NotEmpty(req.Msg.GetAppId(), "app_id is required"),
		assert.NotEmpty(req.Msg.GetEnvironmentId(), "environment_id is required"),
		assert.NotEmpty(req.Msg.GetDomain(), "domain is required"),
	); err != nil {
		// Not InvalidArgument: callers derive these from resolved state, and it keeps
		// InvalidArgument exclusively a gatefault code.
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	domain := strings.ToLower(req.Msg.GetDomain())
	if err := gatefault.ConnectWith(connect.CodeInvalidArgument, domaingate.CheckDomain(domain)); err != nil {
		return nil, err
	}

	// Generate unique CNAME target for this domain
	targetCname := fmt.Sprintf("%s.%s", uid.DNS1035(16), s.cnameDomain)

	// Generate verification token for TXT record ownership verification
	verificationToken := uid.Secure(24)

	_, err := s.db.FindCustomDomainIDByWorkspaceAndDomain(ctx, db.FindCustomDomainIDByWorkspaceAndDomainParams{
		WorkspaceID: req.Msg.GetWorkspaceId(),
		Domain:      domain,
	})
	if err == nil {
		return nil, gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyAttached(domain))
	}
	if !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing domain: %w", err))
	}

	// Before Domain Connect discovery, so a workspace at its allowance cannot use
	// rejected requests to drive outbound discovery traffic.
	allowed, err := s.db.FindCustomDomainsMaxByWorkspaceID(ctx, req.Msg.GetWorkspaceId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, gatefault.ConnectWith(
				connect.CodeFailedPrecondition,
				domaingate.LimitsNotConfigured(req.Msg.GetWorkspaceId()),
			)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read workspace limits: %w", err))
	}

	// Concurrent creates can both pass. The overshoot is bounded by request
	// concurrency, which beats serializing every create in a workspace behind a lock.
	attached, err := s.db.CountCustomDomainsByWorkspace(ctx, req.Msg.GetWorkspaceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to count custom domains: %w", err))
	}
	if err := gatefault.ConnectWith(
		connect.CodeResourceExhausted,
		domaingate.CheckAllowance(attached, allowed),
	); err != nil {
		return nil, err
	}

	// Domain Connect discovery (best-effort, before DB insert so we can persist results)
	var dcProvider, dcURL string
	if len(s.domainConnectPrivateKeyPEM) > 0 {
		logger.Info("running domain connect discovery", "domain", domain)

		// Build redirect URL back to app settings via the public Domain Connect
		// callback. Going through a public path keeps the return navigation working
		// even though the SameSite=Strict session cookie is dropped on the cross-site
		// redirect from the DNS provider — the callback page does a same-site client
		// redirect to the (authenticated) settings page, which carries the cookie.
		var redirectURL string
		ws, wsErr := s.db.FindWorkspaceByID(ctx, req.Msg.GetWorkspaceId())
		if wsErr != nil {
			logger.Warn("failed to fetch workspace for redirect URL", "error", wsErr)
		} else {
			settingsPath := fmt.Sprintf("/%s/projects/%s/apps/%s/settings", ws.Slug, req.Msg.GetProjectId(), req.Msg.GetAppId())
			redirectURL = fmt.Sprintf("https://app.unkey.com/integrations/domain-connect/callback?to=%s", url.QueryEscape(settingsPath))
		}

		result, dcErr := domainconnect.Discover(ctx, domain, s.domainConnectPrivateKeyPEM, map[string]string{
			"target":            targetCname,
			"verificationToken": verificationToken,
		}, redirectURL)
		if dcErr != nil {
			logger.Warn("domain connect discovery failed", "domain", domain, "error", dcErr)
		} else if result != nil {
			logger.Info("domain connect provider found", "domain", domain, "provider", result.ProviderName)
			dcProvider = result.ProviderName
			dcURL = result.URL
		} else {
			logger.Info("domain connect not supported by provider", "domain", domain)
		}
	} else {
		logger.Debug("domain connect disabled, skipping discovery", "domain", domain)
	}

	// Create custom domain record (workflow needs it in DB)
	domainID := uid.New(uid.DomainPrefix)
	now := time.Now().UnixMilli()

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if txErr := db.NewQueries(tx).InsertCustomDomain(txCtx, db.InsertCustomDomainParams{
			ID:                    domainID,
			WorkspaceID:           req.Msg.GetWorkspaceId(),
			ProjectID:             req.Msg.GetProjectId(),
			AppID:                 req.Msg.GetAppId(),
			EnvironmentID:         req.Msg.GetEnvironmentId(),
			Domain:                domain,
			ChallengeType:         db.CustomDomainsChallengeTypeHTTP01,
			VerificationStatus:    db.CustomDomainsVerificationStatusPending,
			VerificationToken:     verificationToken,
			TargetCname:           targetCname,
			DomainConnectProvider: sql.NullString{Valid: dcProvider != "", String: dcProvider},
			DomainConnectUrl:      sql.NullString{Valid: dcURL != "", String: dcURL},
			InvocationID:          sql.NullString{String: "", Valid: false},
			CreatedAt:             now,
		}); txErr != nil {
			if db.IsDuplicateKeyError(txErr) {
				return gatefault.ConnectWith(connect.CodeAlreadyExists, domaingate.AlreadyAttached(domain))
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("insert custom domain: %w", txErr))
		}

		// An absent actor is attributed to the system rather than rejected: losing
		// attribution on one entry beats refusing the write, since the entry and the
		// domain commit together.
		a := req.Msg.GetActor()
		if txErr := s.auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   req.Msg.GetWorkspaceId(),
				Event:         auditlog.DomainCreateEvent,
				Display:       fmt.Sprintf("Added custom domain %s", domain),
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
						Type:        auditlog.DomainResourceType,
						Meta:        map[string]any{"domain": domain, "projectId": req.Msg.GetProjectId(), "appId": req.Msg.GetAppId(), "environmentId": req.Msg.GetEnvironmentId()},
						Name:        domain,
						DisplayName: domain,
					},
				},
			},
		}); txErr != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("insert audit log: %w", txErr))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Trigger verification workflow and store invocation ID
	// Domain ID is the virtual object key (not domain name, since domains are workspace-scoped)
	client := hydrav1.NewCustomDomainServiceIngressClient(s.restate, domainID)
	sendResp, sendErr := client.VerifyDomain().Send(ctx, &hydrav1.VerifyDomainRequest{})
	if sendErr != nil {
		logger.Error(
			"failed to trigger verification workflow",
			"domain", domain,
			"domain_id", domainID,
			"error", sendErr,
		)

		// The 24 hour window is only evaluated inside the workflow, so a domain whose
		// workflow never started would sit in `pending` forever, indistinguishable
		// from one still polling DNS. `failed` is what RetryVerification resets from.
		if failErr := s.db.UpdateCustomDomainFailed(ctx, db.UpdateCustomDomainFailedParams{
			ID:                 domainID,
			VerificationStatus: db.CustomDomainsVerificationStatusFailed,
			VerificationError:  sql.NullString{Valid: true, String: "verification could not be started, retry verification"},
			UpdatedAt:          sql.NullInt64{Valid: true, Int64: now},
		}); failErr != nil {
			logger.Error(
				"failed to record unstarted verification",
				"domain_id", domainID,
				"error", failErr,
			)
		}
	} else {
		_ = s.db.UpdateCustomDomainInvocationID(ctx, db.UpdateCustomDomainInvocationIDParams{
			ID:           domainID,
			InvocationID: sql.NullString{Valid: true, String: sendResp.Id()},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: now},
		})
	}

	return connect.NewResponse(&ctrlv1.AddCustomDomainResponse{
		DomainId:              domainID,
		TargetCname:           targetCname,
		Status:                ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
		DomainConnectProvider: dcProvider,
		DomainConnectUrl:      dcURL,
		VerificationToken:     verificationToken,
	}), nil
}

// DeleteCustomDomain deletes a custom domain and its associated resources.
func (s *Service) DeleteCustomDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteCustomDomainRequest],
) (*connect.Response[ctrlv1.DeleteCustomDomainResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	// Find the domain scoped to workspace
	domain, err := s.db.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: req.Msg.GetWorkspaceId(),
		Domain:      req.Msg.GetDomain(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find domain: %w", err))
	}

	// Validate project ownership
	if domain.ProjectID != req.Msg.GetProjectId() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
	}

	// Cancel any running verification workflow
	if domain.InvocationID.Valid && s.restateAdmin != nil {
		if cancelErr := s.restateAdmin.CancelInvocation(ctx, domain.InvocationID.String); cancelErr != nil {
			logger.Warn(
				"failed to cancel verification workflow",
				"domain", domain.Domain,
				"invocation_id", domain.InvocationID.String,
				"error", cancelErr,
			)
			// Continue with deletion even if cancel fails
		}
	}

	// Delete in transaction: frontline route, ACME challenge, custom domain
	err = db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		// Delete the frontline route only if it belongs to this caller's project.
		// frontline_routes enforces UNIQUE(fully_qualified_domain_name), so exactly
		// one route exists per FQDN, owned by whichever project verified it. Scoping
		// the delete by project_id prevents deleting a route that another workspace
		// legitimately owns when this workspace merely holds an unverified custom
		// domain row for the same FQDN.
		if deleteErr := db.NewQueries(tx).DeleteFrontlineRouteByFQDNAndProject(txCtx, db.DeleteFrontlineRouteByFQDNAndProjectParams{
			Fqdn:      req.Msg.GetDomain(),
			ProjectID: domain.ProjectID,
		}); deleteErr != nil && !db.IsNotFound(deleteErr) {
			return fmt.Errorf("failed to delete frontline route: %w", deleteErr)
		}

		// Delete ACME challenge if exists
		if deleteErr := db.NewQueries(tx).DeleteAcmeChallengeByDomainID(txCtx, domain.ID); deleteErr != nil && !db.IsNotFound(deleteErr) {
			return fmt.Errorf("failed to delete ACME challenge: %w", deleteErr)
		}

		// Delete custom domain
		if deleteErr := db.NewQueries(tx).DeleteCustomDomainByID(txCtx, domain.ID); deleteErr != nil {
			return fmt.Errorf("failed to delete custom domain: %w", deleteErr)
		}

		return nil
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.DeleteCustomDomainResponse{}), nil
}

// RetryVerification resets and restarts verification for a failed domain.
func (s *Service) RetryVerification(
	ctx context.Context,
	req *connect.Request[ctrlv1.RetryVerificationRequest],
) (*connect.Response[ctrlv1.RetryVerificationResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	// Find the domain scoped to workspace
	domain, err := s.db.FindCustomDomainByWorkspaceAndDomain(ctx, db.FindCustomDomainByWorkspaceAndDomainParams{
		WorkspaceID: req.Msg.GetWorkspaceId(),
		Domain:      req.Msg.GetDomain(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find domain: %w", err))
	}

	// Validate project ownership
	if domain.ProjectID != req.Msg.GetProjectId() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
	}

	// Cancel any existing verification workflow
	if domain.InvocationID.Valid && s.restateAdmin != nil {
		if cancelErr := s.restateAdmin.CancelInvocation(ctx, domain.InvocationID.String); cancelErr != nil {
			logger.Warn(
				"failed to cancel old verification workflow",
				"domain", domain.Domain,
				"invocation_id", domain.InvocationID.String,
				"error", cancelErr,
			)
			// Continue anyway - we'll start a new workflow
		}
	}

	// Trigger new verification workflow keyed by domain ID
	client := hydrav1.NewCustomDomainServiceIngressClient(s.restate, domain.ID)
	sendResp, sendErr := client.VerifyDomain().Send(ctx, &hydrav1.VerifyDomainRequest{})
	if sendErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger verification: %w", sendErr))
	}

	// Reset verification state with new invocation ID
	err = s.db.ResetCustomDomainVerification(ctx, db.ResetCustomDomainVerificationParams{
		ID:                 domain.ID,
		VerificationStatus: db.CustomDomainsVerificationStatusPending,
		CheckAttempts:      0,
		InvocationID:       sql.NullString{Valid: true, String: sendResp.Id()},
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reset verification: %w", err))
	}

	return connect.NewResponse(&ctrlv1.RetryVerificationResponse{
		Status: ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
	}), nil
}
