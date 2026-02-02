package billing

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// noopStripeConnectService is a no-op implementation of StripeConnectService
// used when billing is not configured.
type noopStripeConnectService struct{}

// NewNoopStripeConnectService creates a no-op Stripe Connect service.
func NewNoopStripeConnectService() StripeConnectService {
	return &noopStripeConnectService{}
}

func (s *noopStripeConnectService) GetAuthorizationURL(
	ctx context.Context,
	workspaceID, redirectURI, state string,
) (string, error) {
	return "", fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopStripeConnectService) ExchangeCode(ctx context.Context, code string) (*ConnectedAccount, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopStripeConnectService) DisconnectAccount(ctx context.Context, workspaceID string) error {
	return fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopStripeConnectService) GetConnectedAccount(ctx context.Context, workspaceID string) (*ConnectedAccount, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

// noopPricingModelService is a no-op implementation of PricingModelService
// used when billing is not configured.
type noopPricingModelService struct{}

// NewNoopPricingModelService creates a no-op pricing model service.
func NewNoopPricingModelService() PricingModelService {
	return &noopPricingModelService{}
}

func (s *noopPricingModelService) CreatePricingModel(
	ctx context.Context,
	req *CreatePricingModelRequest,
) (*PricingModel, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopPricingModelService) GetPricingModel(ctx context.Context, id string) (*PricingModel, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopPricingModelService) ListPricingModels(ctx context.Context, workspaceID string) ([]*PricingModel, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopPricingModelService) UpdatePricingModel(
	ctx context.Context,
	id string,
	req *UpdatePricingModelRequest,
) (*PricingModel, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopPricingModelService) DeletePricingModel(ctx context.Context, id string) error {
	return fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopPricingModelService) CalculateCharges(
	ctx context.Context,
	model *PricingModel,
	verifications, keysWithAccess int64,
) (int64, error) {
	return 0, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

// noopEndUserService is a no-op implementation of EndUserService
// used when billing is not configured.
type noopEndUserService struct{}

// NewNoopEndUserService creates a no-op end user service.
func NewNoopEndUserService() EndUserService {
	return &noopEndUserService{}
}

func (s *noopEndUserService) CreateEndUser(
	ctx context.Context,
	req *CreateEndUserRequest,
) (*EndUser, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopEndUserService) GetEndUser(ctx context.Context, id string) (*EndUser, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopEndUserService) ListEndUsers(ctx context.Context, workspaceID string) ([]*EndUser, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopEndUserService) UpdateEndUser(
	ctx context.Context,
	id string,
	req *UpdateEndUserRequest,
) (*EndUser, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopEndUserService) GetUsage(
	ctx context.Context,
	endUserID string,
	start, end time.Time,
) (*Usage, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

// noopBillingService is a no-op implementation of BillingService
// used when billing is not configured.
type noopBillingService struct{}

// NewNoopBillingService creates a no-op billing service.
func NewNoopBillingService() BillingService {
	return &noopBillingService{}
}

func (s *noopBillingService) GenerateInvoices(
	ctx context.Context,
	workspaceID string,
	periodStart, periodEnd time.Time,
) error {
	return fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopBillingService) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopBillingService) ListInvoices(
	ctx context.Context,
	req *ListInvoicesRequest,
) ([]*Invoice, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopBillingService) ProcessWebhook(
	ctx context.Context,
	payload []byte,
	signature string,
) error {
	return fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}

func (s *noopBillingService) GetRevenueAnalytics(
	ctx context.Context,
	req *RevenueAnalyticsRequest,
) (*RevenueAnalytics, error) {
	return nil, fault.Wrap(
		fault.New("billing not configured"),
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Public("Billing is not configured"),
	)
}
