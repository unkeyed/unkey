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
	"github.com/unkeyed/unkey/pkg/uid"
)

// Service implements the CustomDomainService ConnectRPC API. It coordinates
// custom domain operations by persisting state to the database and delegating
// verification workflows to Restate.
type Service struct {
	ctrlv1connect.UnimplementedCustomDomainServiceHandler
	db           db.Database
	restate      *restateingress.Client
	logger       logging.Logger
	defaultCname string
}

// Config holds the configuration for creating a new [Service].
type Config struct {
	// Database provides read/write access to custom domain metadata.
	Database db.Database
	// Restate is the ingress client for triggering durable verification workflows.
	Restate *restateingress.Client
	// Logger is used for structured logging throughout the service.
	Logger logging.Logger
	// DefaultCname is the CNAME target that users must point their domains to.
	DefaultCname string
}

// New creates a new [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedCustomDomainServiceHandler: ctrlv1connect.UnimplementedCustomDomainServiceHandler{},
		db:                                      cfg.Database,
		restate:                                 cfg.Restate,
		logger:                                  cfg.Logger,
		defaultCname:                            cfg.DefaultCname,
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

	// Check domain doesn't already exist
	existing, err := db.Query.FindCustomDomainByDomain(ctx, s.db.RO(), domain)
	if err != nil && !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check existing domain: %w", err))
	}
	if existing.ID != "" {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("domain already registered: %s", domain))
	}

	// Create custom domain record
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
		TargetCname:        s.defaultCname,
		CreatedAt:          now,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to create domain: %w", err))
	}

	// Trigger verification workflow
	client := hydrav1.NewCustomDomainServiceIngressClient(s.restate, domain)
	_, err = client.VerifyDomain().Send(ctx, &hydrav1.VerifyDomainRequest{
		WorkspaceId: req.Msg.GetWorkspaceId(),
		Domain:      domain,
	})
	if err != nil {
		s.logger.Warn("failed to trigger verification workflow",
			"domain", domain,
			"error", err,
		)
		// Don't fail the request - domain is created, verification can be retried
	}

	return connect.NewResponse(&ctrlv1.AddCustomDomainResponse{
		DomainId:    domainID,
		TargetCname: s.defaultCname,
		Status:      ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
	}), nil
}

// GetCustomDomain retrieves a custom domain by its domain name.
func (s *Service) GetCustomDomain(
	ctx context.Context,
	req *connect.Request[ctrlv1.GetCustomDomainRequest],
) (*connect.Response[ctrlv1.GetCustomDomainResponse], error) {
	domain, err := db.Query.FindCustomDomainByDomain(ctx, s.db.RO(), req.Msg.GetDomain())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("domain not found: %s", req.Msg.GetDomain()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find domain: %w", err))
	}

	return connect.NewResponse(&ctrlv1.GetCustomDomainResponse{
		Domain: domainToProto(domain),
	}), nil
}

// ListCustomDomains lists all custom domains for a project.
func (s *Service) ListCustomDomains(
	ctx context.Context,
	req *connect.Request[ctrlv1.ListCustomDomainsRequest],
) (*connect.Response[ctrlv1.ListCustomDomainsResponse], error) {
	domains, err := db.Query.ListCustomDomainsByProjectID(ctx, s.db.RO(), req.Msg.GetProjectId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list domains: %w", err))
	}

	result := make([]*ctrlv1.CustomDomain, 0, len(domains))
	for _, d := range domains {
		result = append(result, domainToProto(d))
	}

	return connect.NewResponse(&ctrlv1.ListCustomDomainsResponse{
		Domains: result,
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

	// Reset verification state
	err = db.Query.ResetCustomDomainVerification(ctx, s.db.RW(), db.ResetCustomDomainVerificationParams{
		Domain:             req.Msg.GetDomain(),
		VerificationStatus: db.CustomDomainsVerificationStatusPending,
		CheckAttempts:      0,
		UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to reset verification: %w", err))
	}

	// Trigger new verification workflow
	client := hydrav1.NewCustomDomainServiceIngressClient(s.restate, domain.Domain)
	_, err = client.VerifyDomain().Send(ctx, &hydrav1.VerifyDomainRequest{
		WorkspaceId: domain.WorkspaceID,
		Domain:      domain.Domain,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger verification: %w", err))
	}

	return connect.NewResponse(&ctrlv1.RetryVerificationResponse{
		Status: ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING,
	}), nil
}

// domainToProto converts a database CustomDomain to the protobuf representation.
func domainToProto(d db.CustomDomain) *ctrlv1.CustomDomain {
	status := ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_UNSPECIFIED
	switch d.VerificationStatus {
	case db.CustomDomainsVerificationStatusPending:
		status = ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_PENDING
	case db.CustomDomainsVerificationStatusVerifying:
		status = ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_VERIFYING
	case db.CustomDomainsVerificationStatusVerified:
		status = ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_VERIFIED
	case db.CustomDomainsVerificationStatusFailed:
		status = ctrlv1.CustomDomainStatus_CUSTOM_DOMAIN_STATUS_FAILED
	}

	var lastCheckedAt int64
	if d.LastCheckedAt.Valid {
		lastCheckedAt = d.LastCheckedAt.Int64
	}

	var verificationError string
	if d.VerificationError.Valid {
		verificationError = d.VerificationError.String
	}

	var updatedAt int64
	if d.UpdatedAt.Valid {
		updatedAt = d.UpdatedAt.Int64
	}

	return &ctrlv1.CustomDomain{
		Id:                 d.ID,
		Domain:             d.Domain,
		WorkspaceId:        d.WorkspaceID,
		ProjectId:          d.ProjectID,
		EnvironmentId:      d.EnvironmentID,
		VerificationStatus: status,
		TargetCname:        d.TargetCname,
		CheckAttempts:      d.CheckAttempts,
		LastCheckedAt:      lastCheckedAt,
		VerificationError:  verificationError,
		CreatedAt:          d.CreatedAt,
		UpdatedAt:          updatedAt,
	}
}
