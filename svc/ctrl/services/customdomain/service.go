package customdomain

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Service implements the CustomDomainService ConnectRPC API. It coordinates
// custom domain operations by persisting state to the database and delegating
// verification workflows to Restate.
type Service struct {
	ctrlv1connect.UnimplementedCustomDomainServiceHandler
	db           db.Database
	restate      *restateingress.Client
	restateAdmin *restateadmin.Client
	logger       logging.Logger
	cnameDomain      string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to custom domain metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable verification workflows.
	Restate *restateingress.Client
	// RestateAdmin is the admin client for canceling invocations.
	RestateAdmin *restateadmin.Client
	// Logger is used for structured logging throughout the service.
	Logger logging.Logger
	// CnameDomain is the base domain for custom domain CNAME targets.
	CnameDomain string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedCustomDomainServiceHandler: ctrlv1connect.UnimplementedCustomDomainServiceHandler{},
		db:                                      cfg.Database,
		restate:                                 cfg.Restate,
		restateAdmin:                            cfg.RestateAdmin,
		logger:                                  cfg.Logger,
		cnameDomain:                                 cfg.CnameDomain,
	}
}

// domainRegex validates domain format (basic validation)
var domainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

// isValidDomain checks if the domain has a valid format.
func isValidDomain(domain string) bool {
	if len(domain) > 253 {
		return false
	}
	return domainRegex.MatchString(domain)
}

// AddCustomDomain creates a new custom domain and starts the verification workflow.
func (s *Service) AddCustomDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.AddCustomDomainRequest],
) (*connect.Response[ctrlv1.AddCustomDomainResponse], error) {
	domain := req.Msg.GetDomain()

	// Validate domain format
	if !isValidDomain(domain) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid domain format: %s", domain))
	}

	// Generate unique CNAME target for this domain
	targetCname := fmt.Sprintf("%s.%s", uid.DNS1035(16), s.cnameDomain)

	// Generate verification token for TXT record ownership verification
	verificationToken := uid.Secure(24)

	// Check domain doesn't already exist
	existing, err := db.Query.FindCustomDomainByDomain(ctx, s.db.RO(), domain)
	if err != nil && !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing domain: %w", err))
	}
	if existing.ID != "" {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("domain already registered: %s", domain))
	}

	// Create custom domain record first (workflow needs it in DB)
	domainID := uid.New(uid.DomainPrefix)
	now := time.Now().UnixMilli()

	err = db.Query.InsertCustomDomain(ctx, s.db.RW(), db.InsertCustomDomainParams{
		ID:                 domainID,
		WorkspaceID:        req.Msg.GetWorkspaceId(),
		ProjectID:          req.Msg.GetProjectId(),
		EnvironmentID:      req.Msg.GetEnvironmentId(),
		Domain:             domain,
		ChallengeType:      db.CustomDomainsChallengeTypeHTTP01,
		VerificationStatus: db.CustomDomainsVerificationStatusPending,
		VerificationToken:  verificationToken,
		TargetCname:        targetCname,
		CreatedAt:          now,
		InvocationID:       sql.NullString{String: "", Valid: false},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create domain: %w", err))
	}

	// Trigger verification workflow and store invocation ID
	// Domain is passed as the virtual object key when creating the client
	client := hydrav1.NewCustomDomainServiceIngressClient(s.restate, domain)
	sendResp, sendErr := client.VerifyDomain().Send(ctx, &hydrav1.VerifyDomainRequest{})
	if sendErr != nil {
		s.logger.Warn("failed to trigger verification workflow",
			"domain", domain,
			"error", sendErr,
		)
		// Don't fail the request - domain is created, verification can be retried
	} else {
		_ = db.Query.UpdateCustomDomainInvocationID(ctx, s.db.RW(), db.UpdateCustomDomainInvocationIDParams{
			ID:           domainID,
			InvocationID: sql.NullString{Valid: true, String: sendResp.Id()},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: now},
		})
	}

	return connect.NewResponse(&ctrlv1.AddCustomDomainResponse{
		DomainId:    domainID,
		TargetCname: targetCname,
		Status:      ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
	}), nil
}

// DeleteCustomDomain deletes a custom domain and its associated resources.
func (s *Service) DeleteCustomDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteCustomDomainRequest],
) (*connect.Response[ctrlv1.DeleteCustomDomainResponse], error) {
	// Find the domain first
	domain, err := db.Query.FindCustomDomainByDomain(ctx, s.db.RO(), req.Msg.GetDomain())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find domain: %w", err))
	}

	// Validate tenant ownership
	if domain.WorkspaceID != req.Msg.GetWorkspaceId() || domain.ProjectID != req.Msg.GetProjectId() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
	}

	// Cancel any running verification workflow
	if domain.InvocationID.Valid && s.restateAdmin != nil {
		if cancelErr := s.restateAdmin.CancelInvocation(ctx, domain.InvocationID.String); cancelErr != nil {
			s.logger.Warn("failed to cancel verification workflow",
				"domain", domain.Domain,
				"invocation_id", domain.InvocationID.String,
				"error", cancelErr,
			)
			// Continue with deletion even if cancel fails
		}
	}

	// Delete in transaction: frontline route, ACME challenge, custom domain
	err = db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		// Delete frontline route if exists
		if deleteErr := db.Query.DeleteFrontlineRouteByFQDN(txCtx, tx, req.Msg.GetDomain()); deleteErr != nil && !db.IsNotFound(deleteErr) {
			return fmt.Errorf("failed to delete frontline route: %w", deleteErr)
		}

		// Delete ACME challenge if exists
		if deleteErr := db.Query.DeleteAcmeChallengeByDomainID(txCtx, tx, domain.ID); deleteErr != nil && !db.IsNotFound(deleteErr) {
			return fmt.Errorf("failed to delete ACME challenge: %w", deleteErr)
		}

		// Delete custom domain
		if deleteErr := db.Query.DeleteCustomDomainByID(txCtx, tx, domain.ID); deleteErr != nil {
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
	// Find the domain first
	domain, err := db.Query.FindCustomDomainByDomain(ctx, s.db.RO(), req.Msg.GetDomain())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find domain: %w", err))
	}

	// Validate tenant ownership
	if domain.WorkspaceID != req.Msg.GetWorkspaceId() || domain.ProjectID != req.Msg.GetProjectId() {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
	}

	// Cancel any existing verification workflow
	if domain.InvocationID.Valid && s.restateAdmin != nil {
		if cancelErr := s.restateAdmin.CancelInvocation(ctx, domain.InvocationID.String); cancelErr != nil {
			s.logger.Warn("failed to cancel old verification workflow",
				"domain", domain.Domain,
				"invocation_id", domain.InvocationID.String,
				"error", cancelErr,
			)
			// Continue anyway - we'll start a new workflow
		}
	}

	// Trigger new verification workflow
	// Domain is passed as the virtual object key when creating the client
	client := hydrav1.NewCustomDomainServiceIngressClient(s.restate, domain.Domain)
	sendResp, sendErr := client.VerifyDomain().Send(ctx, &hydrav1.VerifyDomainRequest{})
	if sendErr != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger verification: %w", sendErr))
	}

	// Reset verification state with new invocation ID
	err = db.Query.ResetCustomDomainVerification(ctx, s.db.RW(), db.ResetCustomDomainVerificationParams{
		Domain:             req.Msg.GetDomain(),
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
