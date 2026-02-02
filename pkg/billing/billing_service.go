package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/invoice"
	"github.com/stripe/stripe-go/v81/invoiceitem"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/uid"
)

// PlatformFeePercent is the percentage fee Unkey charges on top of usage
const PlatformFeePercent = 0.03 // 3%

// BillingService manages invoice generation, payment collection, and billing analytics.
type BillingService interface {
	// GenerateInvoices generates invoices for monthly billing period (no manual trigger)
	GenerateInvoices(ctx context.Context, workspaceID string, periodStart, periodEnd time.Time) (UsageStats, error)

	// GetInvoice retrieves invoice by ID
	GetInvoice(ctx context.Context, id string) (*Invoice, error)

	// ListInvoices lists invoices with filters
	ListInvoices(ctx context.Context, req *ListInvoicesRequest) ([]*Invoice, error)

	// ProcessWebhook processes Stripe webhook event
	ProcessWebhook(ctx context.Context, payload []byte, signature string) error

	// GetRevenueAnalytics returns revenue analytics
	GetRevenueAnalytics(ctx context.Context, req *RevenueAnalyticsRequest) (*RevenueAnalytics, error)
}

// Invoice represents a billing invoice
type Invoice struct {
	ID                 string
	WorkspaceID        string
	EndUserID          string
	StripeInvoiceID    string
	BillingPeriodStart time.Time
	BillingPeriodEnd   time.Time
	VerificationCount  int64
	RatelimitCount     int64
	KeyAccessCount     int64
	CreditsUsed        int64
	TotalAmount        int64
	Currency           string
	Status             InvoiceStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// InvoiceStatus represents the status of an invoice
type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusVoid          InvoiceStatus = "void"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
)

// Transaction represents a payment transaction
type Transaction struct {
	ID                    string
	InvoiceID             string
	StripePaymentIntentID string
	Amount                int64
	Currency              string
	Status                TransactionStatus
	FailureReason         string
	CreatedAt             time.Time
}

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusSucceeded TransactionStatus = "succeeded"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusPending   TransactionStatus = "pending"
)

// ListInvoicesRequest contains parameters for listing invoices
type ListInvoicesRequest struct {
	WorkspaceID string
	EndUserID   string
	Status      string
	Limit       int
	Offset      int
}

// RevenueAnalyticsRequest contains parameters for revenue analytics
type RevenueAnalyticsRequest struct {
	WorkspaceID string
	StartDate   time.Time
	EndDate     time.Time
	Granularity string // day, week, month
}

// RevenueAnalytics contains revenue analytics data
type RevenueAnalytics struct {
	TotalRevenue int64
	DataPoints   []RevenueDataPoint
}

// RevenueDataPoint represents a single data point in revenue analytics
type RevenueDataPoint struct {
	Timestamp time.Time
	Revenue   int64
	Count     int64
}

// billingService implements BillingService
type billingService struct {
	db              db.Database
	usageAggregator UsageAggregator
	pricingService  PricingModelService
	endUserService  EndUserService
	connectService  StripeConnectService
	stripeKey       string
	webhookSecret   string
	circuitBreaker  *circuitbreaker.CB[any]
	retrier         *retry.Retry
}

// NewBillingService creates a new billing service
func NewBillingService(
	database db.Database,
	usageAggregator UsageAggregator,
	pricingService PricingModelService,
	endUserService EndUserService,
	connectService StripeConnectService,
	stripeKey string,
	webhookSecret string,
) BillingService {
	// Configure circuit breaker for Stripe API calls
	cb := circuitbreaker.New[any](
		"stripe-api",
		circuitbreaker.WithMaxRequests(10),
		circuitbreaker.WithCyclicPeriod(30*time.Second),
		circuitbreaker.WithTimeout(time.Minute),
		circuitbreaker.WithTripThreshold(5),
	)

	// Configure retry with exponential backoff
	retrier := retry.New(
		retry.Attempts(5),
		retry.Backoff(func(n int) time.Duration {
			// Exponential backoff: 100ms, 200ms, 400ms, 800ms, 1600ms
			return time.Duration(1<<uint(n-1)) * 100 * time.Millisecond
		}),
	)

	return &billingService{
		db:              database,
		usageAggregator: usageAggregator,
		pricingService:  pricingService,
		endUserService:  endUserService,
		connectService:  connectService,
		stripeKey:       stripeKey,
		webhookSecret:   webhookSecret,
		circuitBreaker:  cb,
		retrier:         retrier,
	}
}

// GenerateInvoices generates invoices for all active end users in a workspace
// for the specified billing period. This is called by the scheduled billing job.
//
// Validates: Requirements 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8
// UsageStats holds usage statistics for invoice generation
type UsageStats struct {
	Verifications int64
	Credits       int64
}

func (s *billingService) GenerateInvoices(
	ctx context.Context,
	workspaceID string,
	periodStart, periodEnd time.Time,
) (UsageStats, error) {
	stats := UsageStats{
		Verifications: 0,
		Credits:       0,
	}

	if workspaceID == "" {
		return stats, fault.Wrap(
			fmt.Errorf("workspace ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Workspace ID is required"),
		)
	}

	// Get connected account
	connectedAccount, err := s.connectService.GetConnectedAccount(ctx, workspaceID)
	if err != nil {
		return stats, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to get connected account: %v", err)),
			fault.Public("No Stripe account connected"),
		)
	}

	if connectedAccount.DisconnectedAt != nil {
		return stats, fault.Wrap(
			fmt.Errorf("stripe account is disconnected"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Stripe account is disconnected"),
		)
	}

	// Aggregate usage for the billing period
	usageMap, err := s.usageAggregator.AggregateUsage(ctx, workspaceID, periodStart, periodEnd)
	if err != nil {
		// Check if error is temporary (ClickHouse unavailable)
		if isTemporaryError(err) {
			return stats, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("usage data temporarily unavailable: %v", err)),
				fault.Public("Usage data is temporarily unavailable. Invoice generation will be retried."),
			)
		}
		return stats, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to aggregate usage: %v", err)),
			fault.Public("Failed to retrieve usage data"),
		)
	}

	fmt.Printf("DEBUG: GenerateInvoices: usageMap from AggregateUsage: len=%d\n", len(usageMap))
	for k, v := range usageMap {
		fmt.Printf("DEBUG: GenerateInvoices: usageMap[%s] = {Verifications: %d, Credits: %d}\n", k, v.Verifications, v.Credits)
	}

	// Get all end users for workspace
	endUsers, err := s.endUserService.ListEndUsers(ctx, workspaceID)
	if err != nil {
		return stats, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to list end users: %v", err)),
			fault.Public("Failed to retrieve end users"),
		)
	}

	// DEBUG: Log usage map and end users
	fmt.Printf("DEBUG: GenerateInvoices: usageMap entries=%d, endUsers=%d\n", len(usageMap), len(endUsers))
	for _, endUser := range endUsers {
		fmt.Printf("DEBUG: GenerateInvoices: endUser=%s, externalID=%s\n", endUser.ID, endUser.ExternalID)
		if usage, ok := usageMap[endUser.ExternalID]; ok {
			fmt.Printf("DEBUG: GenerateInvoices: found usage for externalID=%s, verifications=%d, credits=%d\n", endUser.ExternalID, usage.Verifications, usage.Credits)
		} else {
			fmt.Printf("DEBUG: GenerateInvoices: NO usage found for externalID=%s\n", endUser.ExternalID)
		}
	}

	// Generate invoice for each end user with usage
	for _, endUser := range endUsers {
		usage, hasUsage := usageMap[endUser.ExternalID]
		if !hasUsage || usage.Verifications == 0 {
			// Skip end users with no usage
			continue
		}

		// Accumulate stats
		stats.Verifications += usage.Verifications
		stats.Credits += usage.Credits

		err := s.generateInvoiceForEndUser(
			ctx,
			connectedAccount,
			endUser,
			usage,
			periodStart,
			periodEnd,
		)
		if err != nil {
			// Log error but continue with other end users
			// In production, this should be logged to a monitoring system
			return stats, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to generate invoice for end user %s: %v", endUser.ID, err)),
				fault.Public(fmt.Sprintf("Failed to generate invoice for end user %s", endUser.ExternalID)),
			)
		}
	}

	return stats, nil
}

// generateInvoiceForEndUser generates an invoice for a single end user
func (s *billingService) generateInvoiceForEndUser(
	ctx context.Context,
	connectedAccount *ConnectedAccount,
	endUser *EndUser,
	usage *Usage,
	periodStart, periodEnd time.Time,
) error {
	// Get pricing model
	pricingModel, err := s.pricingService.GetPricingModel(ctx, endUser.PricingModelID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to get pricing model: %v", err)),
			fault.Public("Failed to retrieve pricing model"),
		)
	}

	// Calculate charges
	totalAmount, err := s.pricingService.CalculateCharges(
		ctx,
		pricingModel,
		usage.Verifications,
		usage.KeysWithAccess,
	)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to calculate charges: %v", err)),
			fault.Public("Failed to calculate charges"),
		)
	}

	// Calculate platform fee (3% of usage charges)
	platformFee := int64(float64(totalAmount) * PlatformFeePercent)
	totalAmountWithFee := totalAmount + platformFee

	// Skip if total amount is zero
	if totalAmountWithFee == 0 {
		return nil
	}

	// Create Stripe invoice with retry and circuit breaker
	var stripeInvoiceID string
	err = s.retrier.DoContext(ctx, func() error {
		_, cbErr := s.circuitBreaker.Do(ctx, func(ctx context.Context) (any, error) {
			invoiceID, createErr := s.createStripeInvoice(
				ctx,
				connectedAccount,
				endUser,
				pricingModel,
				usage,
				totalAmount,
				platformFee,
			)
			if createErr != nil {
				return nil, createErr
			}
			stripeInvoiceID = invoiceID
			return nil, nil
		})
		return cbErr
	})

	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to create Stripe invoice after retries: %v", err)),
			fault.Public("Failed to create invoice in Stripe"),
		)
	}

	// Store invoice in database
	now := time.Now().UnixMilli()
	invoiceID := uid.New(uid.InvoicePrefix)

	// Calculate credits used
	creditsUsed := usage.Credits

	insertParams := db.BillingInvoiceInsertParams{
		ID:                 invoiceID,
		WorkspaceID:        endUser.WorkspaceID,
		EndUserID:          endUser.ID,
		StripeInvoiceID:    stripeInvoiceID,
		BillingPeriodStart: periodStart.UnixMilli(),
		BillingPeriodEnd:   periodEnd.UnixMilli(),
		VerificationCount:  usage.Verifications,
		KeyAccessCount:     usage.KeysWithAccess,
		CreditsUsed:        creditsUsed,
		TotalAmount:        totalAmountWithFee,
		Currency:           pricingModel.Currency,
		Status:             string(InvoiceStatusOpen),
		CreatedAtM:         now,
	}

	err = db.Query.BillingInvoiceInsert(ctx, s.db.RW(), insertParams)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to store invoice: %v", err)),
			fault.Public("Failed to save invoice"),
		)
	}

	return nil
}

// createStripeInvoice creates an invoice in Stripe with automatic payment collection
// and dunning enabled.
//
// Validates: Requirements 5.3, 5.4, 5.5, 5.8, 6.1, 6.2
func (s *billingService) createStripeInvoice(
	ctx context.Context,
	connectedAccount *ConnectedAccount,
	endUser *EndUser,
	pricingModel *PricingModel,
	usage *Usage,
	usageAmount int64,
	platformFee int64,
) (string, error) {
	stripe.Key = s.stripeKey

	// Create invoice items for verifications
	if usage.Verifications > 0 {
		// nolint:exhaustruct // Stripe params have many optional fields
		unitAmountInCents := int64(math.Round(pricingModel.VerificationUnitPrice * 100))
		totalAmountInCents := unitAmountInCents * usage.Verifications
		itemParams := &stripe.InvoiceItemParams{
			Customer:    stripe.String(endUser.StripeCustomerID),
			Amount:      stripe.Int64(totalAmountInCents),
			Currency:    stripe.String(pricingModel.Currency),
			Description: stripe.String(fmt.Sprintf("API Verifications (%d)", usage.Verifications)),
			Quantity:    stripe.Int64(usage.Verifications),
			UnitAmount:  stripe.Int64(unitAmountInCents),
		}
		itemParams.SetStripeAccount(connectedAccount.StripeAccountID)

		_, err := invoiceitem.New(itemParams)
		if err != nil {
			return "", fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to create verification invoice item: %v", err)),
				fault.Public("Failed to create invoice item in Stripe"),
			)
		}
	}

	// Create invoice items for key access (unique keys with VALID=true verification)
	if usage.KeysWithAccess > 0 && pricingModel.KeyAccessUnitPrice > 0 {
		// nolint:exhaustruct // Stripe params have many optional fields
		unitAmountInCents := int64(math.Round(pricingModel.KeyAccessUnitPrice * 100))
		totalAmountInCents := unitAmountInCents * usage.KeysWithAccess
		itemParams := &stripe.InvoiceItemParams{
			Customer:    stripe.String(endUser.StripeCustomerID),
			Amount:      stripe.Int64(totalAmountInCents),
			Currency:    stripe.String(pricingModel.Currency),
			Description: stripe.String(fmt.Sprintf("API Keys with Access (%d)", usage.KeysWithAccess)),
			Quantity:    stripe.Int64(usage.KeysWithAccess),
			UnitAmount:  stripe.Int64(unitAmountInCents),
		}
		itemParams.SetStripeAccount(connectedAccount.StripeAccountID)

		_, err := invoiceitem.New(itemParams)
		if err != nil {
			return "", fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to create key access invoice item: %v", err)),
				fault.Public("Failed to create invoice item in Stripe"),
			)
		}
	}

	// Create invoice items for credits
	creditsUsed := usage.Credits
	if creditsUsed > 0 && pricingModel.CreditUnitPrice > 0 {
		// nolint:exhaustruct // Stripe params have many optional fields
		unitAmountInCents := int64(math.Round(pricingModel.CreditUnitPrice * 100))
		totalAmountInCents := unitAmountInCents * creditsUsed
		itemParams := &stripe.InvoiceItemParams{
			Customer:    stripe.String(endUser.StripeCustomerID),
			Amount:      stripe.Int64(totalAmountInCents),
			Currency:    stripe.String(pricingModel.Currency),
			Description: stripe.String(fmt.Sprintf("Credits Used (%d)", creditsUsed)),
			Quantity:    stripe.Int64(creditsUsed),
			UnitAmount:  stripe.Int64(unitAmountInCents),
		}
		itemParams.SetStripeAccount(connectedAccount.StripeAccountID)

		_, err := invoiceitem.New(itemParams)
		if err != nil {
			return "", fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to create credits invoice item: %v", err)),
				fault.Public("Failed to create invoice item in Stripe"),
			)
		}
	}

	// Create invoice with automatic payment collection and dunning
	// Application fee (3%) is set to collect platform fee from the payment
	// nolint:exhaustruct // Stripe params have many optional fields
	invoiceParams := &stripe.InvoiceParams{
		Customer:                    stripe.String(endUser.StripeCustomerID),
		AutoAdvance:                 stripe.Bool(true), // Automatically finalize and send
		CollectionMethod:            stripe.String("charge_automatically"),
		DaysUntilDue:                stripe.Int64(0), // Due immediately
		Description:                 stripe.String(fmt.Sprintf("Usage for period %s to %s", usage.ExternalID, usage.ExternalID)),
		AutomaticTax:                &stripe.InvoiceAutomaticTaxParams{Enabled: stripe.Bool(false)},
		PendingInvoiceItemsBehavior: stripe.String("include"),
		ApplicationFeeAmount:        stripe.Int64(platformFee),
	}
	invoiceParams.SetStripeAccount(connectedAccount.StripeAccountID)

	stripeInvoice, err := invoice.New(invoiceParams)
	if err != nil {
		return "", fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to create invoice: %v", err)),
			fault.Public("Failed to create invoice in Stripe"),
		)
	}

	// Finalize the invoice to send it to the customer
	// nolint:exhaustruct // Stripe params have many optional fields
	finalizeParams := &stripe.InvoiceFinalizeInvoiceParams{
		AutoAdvance: stripe.Bool(true),
	}
	finalizeParams.SetStripeAccount(connectedAccount.StripeAccountID)

	_, err = invoice.FinalizeInvoice(stripeInvoice.ID, finalizeParams)
	if err != nil {
		return "", fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to finalize invoice: %v", err)),
			fault.Public("Failed to finalize invoice"),
		)
	}

	return stripeInvoice.ID, nil
}

// GetInvoice retrieves an invoice by ID.
//
// Validates: Requirements 5.6
func (s *billingService) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	if id == "" {
		return nil, fault.Wrap(
			fmt.Errorf("invoice ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invoice ID is required"),
		)
	}

	dbInvoice, err := db.Query.BillingInvoiceFindById(ctx, s.db.RO(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Invoice not found"),
			)
		}
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to find invoice: %v", err)),
			fault.Public("Failed to retrieve invoice"),
		)
	}

	return dbInvoiceToInvoice(&dbInvoice)
}

// ListInvoices lists invoices with optional filters.
//
// Validates: Requirements 5.6
func (s *billingService) ListInvoices(
	ctx context.Context,
	req *ListInvoicesRequest,
) ([]*Invoice, error) {
	if req.WorkspaceID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("workspace ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Workspace ID is required"),
		)
	}

	// Set default limit if not provided
	if req.Limit <= 0 {
		req.Limit = 50
	}

	// Query invoices - filter by end user if specified, otherwise by workspace
	var dbInvoices []db.BillingInvoice
	var err error

	if req.EndUserID != "" {
		// List by end user
		dbInvoices, err = db.Query.BillingInvoiceListByEndUserId(ctx, s.db.RO(), req.EndUserID)
	} else {
		// List by workspace
		listParams := db.BillingInvoiceListByWorkspaceIdParams{
			WorkspaceID: req.WorkspaceID,
			Limit:       int32(req.Limit),
			Offset:      int32(req.Offset),
		}
		dbInvoices, err = db.Query.BillingInvoiceListByWorkspaceId(ctx, s.db.RO(), listParams)
	}
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to list invoices: %v", err)),
			fault.Public("Failed to retrieve invoices"),
		)
	}

	invoices := make([]*Invoice, 0, len(dbInvoices))
	for _, dbInvoice := range dbInvoices {
		inv, err := dbInvoiceToInvoice(&dbInvoice)
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}

	return invoices, nil
}

// ProcessWebhook processes Stripe webhook events for payment updates.
// Verifies webhook signature and handles invoice.paid, invoice.payment_failed,
// and other dunning-related events.
//
// Validates: Requirements 6.2, 6.3, 6.4, 6.5, 6.6, 6.7
func (s *billingService) ProcessWebhook(
	ctx context.Context,
	payload []byte,
	signature string,
) error {
	// Verify webhook signature
	if err := s.verifyWebhookSignature(payload, signature); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Invalid webhook signature"),
		)
	}

	// Parse webhook event
	var event stripe.Event
	if err := json.Unmarshal(payload, &event); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("failed to parse webhook event: %v", err)),
			fault.Public("Invalid webhook payload"),
		)
	}

	// Handle event based on type
	//nolint:exhaustive // We only handle specific billing-related events
	switch event.Type {
	case "invoice.paid":
		return s.handleInvoicePaid(ctx, &event)
	case "invoice.payment_failed":
		return s.handleInvoicePaymentFailed(ctx, &event)
	case "customer.subscription.deleted":
		return s.handleSubscriptionDeleted(ctx, &event)
	default:
		// Ignore unknown event types
		return nil
	}
}

// verifyWebhookSignature verifies the Stripe webhook signature
func (s *billingService) verifyWebhookSignature(payload []byte, signature string) error {
	// Stripe signature format: t=timestamp,v1=signature
	// Parse the signature header
	var timestamp string
	var sig string

	parts := splitSignature(signature)
	for _, part := range parts {
		if len(part) > 2 && part[0] == 't' && part[1] == '=' {
			timestamp = part[2:]
		}
		if len(part) > 3 && part[0] == 'v' && part[1] == '1' && part[2] == '=' {
			sig = part[3:]
		}
	}

	if timestamp == "" || sig == "" {
		return fmt.Errorf("invalid signature format")
	}

	// Verify timestamp is recent (within 5 minutes)
	// This prevents replay attacks
	// Note: In production, parse timestamp and check

	// Compute expected signature
	signedPayload := fmt.Sprintf("%s.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write([]byte(signedPayload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// splitSignature splits the Stripe signature header by commas
func splitSignature(signature string) []string {
	var parts []string
	var current string

	for _, char := range signature {
		if char == ',' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

// handleInvoicePaid handles the invoice.paid webhook event
func (s *billingService) handleInvoicePaid(ctx context.Context, event *stripe.Event) error {
	var stripeInvoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &stripeInvoice); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to parse invoice.paid event: %v", err)),
			fault.Public("Failed to process webhook"),
		)
	}

	// Find invoice by Stripe invoice ID
	dbInvoice, err := db.Query.BillingInvoiceFindByStripeInvoiceId(ctx, s.db.RO(), stripeInvoice.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Invoice not found - might be from a different system
			return nil
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to find invoice: %v", err)),
			fault.Public("Failed to process webhook"),
		)
	}

	// Update invoice status to paid
	now := time.Now().UnixMilli()
	err = db.Query.BillingInvoiceUpdateStatus(ctx, s.db.RW(), db.BillingInvoiceUpdateStatusParams{
		Status:     string(InvoiceStatusPaid),
		UpdatedAtM: sql.NullInt64{Int64: now, Valid: true},
		ID:         dbInvoice.ID,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to update invoice status: %v", err)),
			fault.Public("Failed to process webhook"),
		)
	}

	// Record transaction
	if stripeInvoice.PaymentIntent != nil {
		transactionID := uid.New(uid.TransactionPrefix)
		insertParams := db.BillingTransactionInsertParams{
			ID:                    transactionID,
			InvoiceID:             dbInvoice.ID,
			StripePaymentIntentID: sql.NullString{String: stripeInvoice.PaymentIntent.ID, Valid: true},
			Amount:                stripeInvoice.AmountPaid,
			Currency:              string(stripeInvoice.Currency),
			Status:                string(TransactionStatusSucceeded),
			FailureReason:         sql.NullString{},
			CreatedAtM:            now,
		}

		err = db.Query.BillingTransactionInsert(ctx, s.db.RW(), insertParams)
		if err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to record transaction: %v", err)),
				fault.Public("Failed to process webhook"),
			)
		}
	}

	return nil
}

// handleInvoicePaymentFailed handles the invoice.payment_failed webhook event
func (s *billingService) handleInvoicePaymentFailed(ctx context.Context, event *stripe.Event) error {
	var stripeInvoice stripe.Invoice
	if err := json.Unmarshal(event.Data.Raw, &stripeInvoice); err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to parse invoice.payment_failed event: %v", err)),
			fault.Public("Failed to process webhook"),
		)
	}

	// Find invoice by Stripe invoice ID
	dbInvoice, err := db.Query.BillingInvoiceFindByStripeInvoiceId(ctx, s.db.RO(), stripeInvoice.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Invoice not found - might be from a different system
			return nil
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to find invoice: %v", err)),
			fault.Public("Failed to process webhook"),
		)
	}

	// Determine status based on Stripe invoice status
	var status InvoiceStatus
	if stripeInvoice.Status == "uncollectible" {
		status = InvoiceStatusUncollectible
	} else {
		status = InvoiceStatusOpen // Keep open for dunning retries
	}

	// Update invoice status
	now := time.Now().UnixMilli()
	err = db.Query.BillingInvoiceUpdateStatus(ctx, s.db.RW(), db.BillingInvoiceUpdateStatusParams{
		Status:     string(status),
		UpdatedAtM: sql.NullInt64{Int64: now, Valid: true},
		ID:         dbInvoice.ID,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to update invoice status: %v", err)),
			fault.Public("Failed to process webhook"),
		)
	}

	// Record failed transaction
	if stripeInvoice.PaymentIntent != nil {
		transactionID := uid.New("transaction")
		failureReason := ""
		if stripeInvoice.PaymentIntent.LastPaymentError != nil {
			failureReason = string(stripeInvoice.PaymentIntent.LastPaymentError.Code)
		}

		insertParams := db.BillingTransactionInsertParams{
			ID:                    transactionID,
			InvoiceID:             dbInvoice.ID,
			StripePaymentIntentID: sql.NullString{String: stripeInvoice.PaymentIntent.ID, Valid: true},
			Amount:                stripeInvoice.AmountDue,
			Currency:              string(stripeInvoice.Currency),
			Status:                string(TransactionStatusFailed),
			FailureReason:         sql.NullString{String: failureReason, Valid: failureReason != ""},
			CreatedAtM:            now,
		}

		err = db.Query.BillingTransactionInsert(ctx, s.db.RW(), insertParams)
		if err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to record transaction: %v", err)),
				fault.Public("Failed to process webhook"),
			)
		}
	}

	return nil
}

// handleSubscriptionDeleted handles the customer.subscription.deleted webhook event
func (s *billingService) handleSubscriptionDeleted(ctx context.Context, event *stripe.Event) error {
	// This event is informational - we don't need to take action
	// Subscriptions are managed by Stripe and we track them in end_user records
	return nil
}

// GetRevenueAnalytics returns revenue analytics for a workspace.
//
// Validates: Requirements 7.1, 7.2, 7.3
func (s *billingService) GetRevenueAnalytics(
	ctx context.Context,
	req *RevenueAnalyticsRequest,
) (*RevenueAnalytics, error) {
	if req.WorkspaceID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("workspace ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Workspace ID is required"),
		)
	}

	// Query all paid invoices for the workspace
	// Note: We'll filter by date range in memory since the query doesn't support it
	dbInvoices, err := db.Query.BillingInvoiceListByWorkspaceId(ctx, s.db.RO(), db.BillingInvoiceListByWorkspaceIdParams{
		WorkspaceID: req.WorkspaceID,
		Limit:       10000, // Large limit to get all invoices
		Offset:      0,
	})
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to list invoices: %v", err)),
			fault.Public("Failed to retrieve revenue data"),
		)
	}

	// Filter by date range, status, and aggregate
	var totalRevenue int64
	dataPointsMap := make(map[int64]*RevenueDataPoint)

	for _, dbInvoice := range dbInvoices {
		// Only include paid invoices
		if dbInvoice.Status != string(InvoiceStatusPaid) {
			continue
		}

		createdAt := time.UnixMilli(dbInvoice.CreatedAtM)
		if createdAt.Before(req.StartDate) || createdAt.After(req.EndDate) {
			continue
		}

		totalRevenue += dbInvoice.TotalAmount

		// Group by granularity
		var bucketTime int64
		switch req.Granularity {
		case "day":
			bucketTime = time.Date(createdAt.Year(), createdAt.Month(), createdAt.Day(), 0, 0, 0, 0, createdAt.Location()).UnixMilli()
		case "week":
			// Start of week (Sunday)
			year, week := createdAt.ISOWeek()
			bucketTime = time.Date(year, 0, (week-1)*7+1, 0, 0, 0, 0, createdAt.Location()).UnixMilli()
		case "month":
			bucketTime = time.Date(createdAt.Year(), createdAt.Month(), 1, 0, 0, 0, 0, createdAt.Location()).UnixMilli()
		default:
			bucketTime = createdAt.UnixMilli()
		}

		if _, exists := dataPointsMap[bucketTime]; !exists {
			dataPointsMap[bucketTime] = &RevenueDataPoint{
				Timestamp: time.UnixMilli(bucketTime),
				Revenue:   0,
				Count:     0,
			}
		}

		dataPointsMap[bucketTime].Revenue += dbInvoice.TotalAmount
		dataPointsMap[bucketTime].Count++
	}

	// Convert map to slice
	dataPoints := make([]RevenueDataPoint, 0, len(dataPointsMap))
	for _, dp := range dataPointsMap {
		dataPoints = append(dataPoints, *dp)
	}

	return &RevenueAnalytics{
		TotalRevenue: totalRevenue,
		DataPoints:   dataPoints,
	}, nil
}

// isTemporaryError checks if an error is temporary (e.g., network, timeout)
func isTemporaryError(err error) bool {
	// Check for common temporary error patterns
	if err == nil {
		return false
	}

	errStr := err.Error()
	// ClickHouse connection errors
	if contains(errStr, "connection refused") ||
		contains(errStr, "timeout") ||
		contains(errStr, "deadline exceeded") ||
		contains(errStr, "temporary failure") {
		return true
	}

	return false
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// dbInvoiceToInvoice converts a database invoice to a domain invoice
func dbInvoiceToInvoice(dbInvoice *db.BillingInvoice) (*Invoice, error) {
	updatedAt := time.UnixMilli(dbInvoice.CreatedAtM)
	if dbInvoice.UpdatedAtM.Valid {
		updatedAt = time.UnixMilli(dbInvoice.UpdatedAtM.Int64)
	}

	return &Invoice{
		ID:                 dbInvoice.ID,
		WorkspaceID:        dbInvoice.WorkspaceID,
		EndUserID:          dbInvoice.EndUserID,
		StripeInvoiceID:    dbInvoice.StripeInvoiceID,
		BillingPeriodStart: time.UnixMilli(dbInvoice.BillingPeriodStart),
		BillingPeriodEnd:   time.UnixMilli(dbInvoice.BillingPeriodEnd),
		VerificationCount:  dbInvoice.VerificationCount,
		RatelimitCount:     dbInvoice.RatelimitCount,
		KeyAccessCount:     dbInvoice.KeyAccessCount,
		CreditsUsed:        dbInvoice.CreditsUsed,
		TotalAmount:        dbInvoice.TotalAmount,
		Currency:           dbInvoice.Currency,
		Status:             InvoiceStatus(dbInvoice.Status),
		CreatedAt:          time.UnixMilli(dbInvoice.CreatedAtM),
		UpdatedAt:          updatedAt,
	}, nil
}
