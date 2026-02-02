package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/customer"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// EndUserService manages end users for billing purposes.
type EndUserService interface {
	// CreateEndUser creates end user and Stripe customer
	CreateEndUser(ctx context.Context, req *CreateEndUserRequest) (*EndUser, error)

	// GetEndUser retrieves end user by ID
	GetEndUser(ctx context.Context, id string) (*EndUser, error)

	// ListEndUsers lists end users for workspace
	ListEndUsers(ctx context.Context, workspaceID string) ([]*EndUser, error)

	// UpdateEndUser updates end user details
	UpdateEndUser(ctx context.Context, id string, req *UpdateEndUserRequest) (*EndUser, error)

	// GetUsage retrieves usage for end user in date range
	GetUsage(ctx context.Context, endUserID string, start, end time.Time) (*Usage, error)
}

// EndUser represents a customer's end user who gets billed
type EndUser struct {
	ID                   string
	WorkspaceID          string
	ExternalID           string
	PricingModelID       string
	StripeCustomerID     string
	StripeSubscriptionID string
	Email                string
	Name                 string
	Metadata             map[string]string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// Usage represents aggregated usage for an end user
type Usage struct {
	ExternalID     string
	Verifications  int64
	KeysWithAccess int64 // unique keys with VALID=true verification
	Credits        int64 // credits spent
}

// CreateEndUserRequest contains parameters for creating an end user
type CreateEndUserRequest struct {
	WorkspaceID    string
	ExternalID     string
	PricingModelID string
	Email          string
	Name           string
	Metadata       map[string]string
}

// UpdateEndUserRequest contains parameters for updating an end user
type UpdateEndUserRequest struct {
	PricingModelID string
	Email          string
	Name           string
	Metadata       map[string]string
}

// endUserService implements EndUserService
type endUserService struct {
	db         db.Database
	ch         clickhouse.ClickHouse
	stripeKey  string
	connectSvc StripeConnectService
}

// NewEndUserService creates a new end user service
func NewEndUserService(
	database db.Database,
	clickhouse clickhouse.ClickHouse,
	stripeKey string,
	connectService StripeConnectService,
) EndUserService {
	return &endUserService{
		db:         database,
		ch:         clickhouse,
		stripeKey:  stripeKey,
		connectSvc: connectService,
	}
}

// CreateEndUser creates a new end user and corresponding Stripe customer.
//
// Validates: Requirements 3.1, 3.2
func (s *endUserService) CreateEndUser(
	ctx context.Context,
	req *CreateEndUserRequest,
) (*EndUser, error) {
	// Validate required fields
	if req.WorkspaceID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("workspace ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Workspace ID is required"),
		)
	}

	if req.ExternalID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("external ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("External ID is required"),
		)
	}

	if req.PricingModelID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("pricing model ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Pricing model ID is required"),
		)
	}

	// Verify pricing model exists
	_, err := db.Query.PricingModelFindById(ctx, s.db.RO(), req.PricingModelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Pricing model not found"),
			)
		}
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to verify pricing model: %v", err)),
			fault.Public("Failed to validate pricing model"),
		)
	}

	// Check if end user with this external_id already exists
	_, err = db.Query.BillingEndUserFindByExternalId(ctx, s.db.RO(), db.BillingEndUserFindByExternalIdParams{
		WorkspaceID: req.WorkspaceID,
		ExternalID:  req.ExternalID,
	})
	if err != nil && err != sql.ErrNoRows {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to check existing end user: %v", err)),
			fault.Public("Failed to validate end user"),
		)
	}
	if err == nil {
		return nil, fault.Wrap(
			fmt.Errorf("end user with external_id %s already exists", req.ExternalID),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public(fmt.Sprintf("End user with external ID %s already exists", req.ExternalID)),
		)
	}

	// Get connected account for workspace
	connectedAccount, err := s.connectSvc.GetConnectedAccount(ctx, req.WorkspaceID)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to get connected account: %v", err)),
			fault.Public("No Stripe account connected. Please connect your Stripe account first."),
		)
	}

	// Check if account is disconnected
	if connectedAccount.DisconnectedAt != nil {
		return nil, fault.Wrap(
			fmt.Errorf("stripe account is disconnected"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Stripe account is disconnected. Please reconnect your account."),
		)
	}

	// Create Stripe customer in connected account
	stripe.Key = s.stripeKey
	// nolint:exhaustruct // Stripe params have many optional fields
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
		Name:  stripe.String(req.Name),
		Metadata: map[string]string{
			"workspace_id": req.WorkspaceID,
			"external_id":  req.ExternalID,
		},
	}
	customerParams.SetStripeAccount(connectedAccount.StripeAccountID)

	stripeCustomer, err := customer.New(customerParams)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to create Stripe customer: %v", err)),
			fault.Public("Failed to create customer in Stripe"),
		)
	}

	// Serialize metadata to JSON
	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, err = json.Marshal(req.Metadata)
		if err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to serialize metadata: %v", err)),
				fault.Public("Failed to save end user metadata"),
			)
		}
	}

	// Create end user in database
	now := time.Now().UnixMilli()
	endUserID := uid.New(uid.EndUserPrefix)

	insertParams := db.BillingEndUserInsertParams{
		ID:                   endUserID,
		WorkspaceID:          req.WorkspaceID,
		ExternalID:           req.ExternalID,
		PricingModelID:       req.PricingModelID,
		StripeCustomerID:     stripeCustomer.ID,
		StripeSubscriptionID: sql.NullString{}, // No subscription initially
		Email:                sql.NullString{String: req.Email, Valid: req.Email != ""},
		Name:                 sql.NullString{String: req.Name, Valid: req.Name != ""},
		Metadata:             metadataJSON,
		CreatedAtM:           now,
	}

	err = db.Query.BillingEndUserInsert(ctx, s.db.RW(), insertParams)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to create end user: %v", err)),
			fault.Public("Failed to create end user"),
		)
	}

	return &EndUser{
		ID:                   endUserID,
		WorkspaceID:          req.WorkspaceID,
		ExternalID:           req.ExternalID,
		PricingModelID:       req.PricingModelID,
		StripeCustomerID:     stripeCustomer.ID,
		StripeSubscriptionID: "",
		Email:                req.Email,
		Name:                 req.Name,
		Metadata:             req.Metadata,
		CreatedAt:            time.UnixMilli(now),
		UpdatedAt:            time.UnixMilli(now),
	}, nil
}

// GetEndUser retrieves an end user by ID.
//
// Validates: Requirements 3.2
func (s *endUserService) GetEndUser(ctx context.Context, id string) (*EndUser, error) {
	if id == "" {
		return nil, fault.Wrap(
			fmt.Errorf("end user ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("End user ID is required"),
		)
	}

	dbEndUser, err := db.Query.BillingEndUserFindById(ctx, s.db.RO(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("End user not found"),
			)
		}
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to find end user: %v", err)),
			fault.Public("Failed to retrieve end user"),
		)
	}

	return dbEndUserToEndUser(&dbEndUser)
}

// ListEndUsers lists all end users for a workspace.
//
// Validates: Requirements 3.2
func (s *endUserService) ListEndUsers(ctx context.Context, workspaceID string) ([]*EndUser, error) {
	if workspaceID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("workspace ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Workspace ID is required"),
		)
	}

	dbEndUsers, err := db.Query.BillingEndUserListByWorkspaceId(ctx, s.db.RO(), workspaceID)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to list end users: %v", err)),
			fault.Public("Failed to retrieve end users"),
		)
	}

	endUsers := make([]*EndUser, 0, len(dbEndUsers))
	for _, dbEndUser := range dbEndUsers {
		endUser, err := dbEndUserToEndUser(&dbEndUser)
		if err != nil {
			return nil, err
		}
		endUsers = append(endUsers, endUser)
	}

	return endUsers, nil
}

// UpdateEndUser updates end user details.
// Pricing model changes apply starting next billing period.
//
// Validates: Requirements 3.4
func (s *endUserService) UpdateEndUser(
	ctx context.Context,
	id string,
	req *UpdateEndUserRequest,
) (*EndUser, error) {
	if id == "" {
		return nil, fault.Wrap(
			fmt.Errorf("end user ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("End user ID is required"),
		)
	}

	// Get existing end user
	existing, err := s.GetEndUser(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify pricing model exists if changed
	if req.PricingModelID != "" && req.PricingModelID != existing.PricingModelID {
		_, err := db.Query.PricingModelFindById(ctx, s.db.RO(), req.PricingModelID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fault.Wrap(
					err,
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Public("Pricing model not found"),
				)
			}
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to verify pricing model: %v", err)),
				fault.Public("Failed to validate pricing model"),
			)
		}
	}

	// Use existing values if not provided
	if req.PricingModelID == "" {
		req.PricingModelID = existing.PricingModelID
	}
	if req.Email == "" {
		req.Email = existing.Email
	}
	if req.Name == "" {
		req.Name = existing.Name
	}
	if req.Metadata == nil {
		req.Metadata = existing.Metadata
	}

	// Serialize metadata to JSON
	var metadataJSON []byte
	if req.Metadata != nil {
		metadataJSON, err = json.Marshal(req.Metadata)
		if err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to serialize metadata: %v", err)),
				fault.Public("Failed to save end user metadata"),
			)
		}
	}

	// Update end user in database
	now := time.Now().UnixMilli()
	updateParams := db.BillingEndUserUpdateParams{
		PricingModelID:       req.PricingModelID,
		StripeSubscriptionID: sql.NullString{String: existing.StripeSubscriptionID, Valid: existing.StripeSubscriptionID != ""},
		Email:                sql.NullString{String: req.Email, Valid: req.Email != ""},
		Name:                 sql.NullString{String: req.Name, Valid: req.Name != ""},
		Metadata:             metadataJSON,
		UpdatedAtM:           sql.NullInt64{Int64: now, Valid: true},
		ID:                   id,
	}

	err = db.Query.BillingEndUserUpdate(ctx, s.db.RW(), updateParams)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to update end user: %v", err)),
			fault.Public("Failed to update end user"),
		)
	}

	return &EndUser{
		ID:                   existing.ID,
		WorkspaceID:          existing.WorkspaceID,
		ExternalID:           existing.ExternalID,
		PricingModelID:       req.PricingModelID,
		StripeCustomerID:     existing.StripeCustomerID,
		StripeSubscriptionID: existing.StripeSubscriptionID,
		Email:                req.Email,
		Name:                 req.Name,
		Metadata:             req.Metadata,
		CreatedAt:            existing.CreatedAt,
		UpdatedAt:            time.UnixMilli(now),
	}, nil
}

// GetUsage retrieves usage for an end user in a date range by querying ClickHouse.
//
// Validates: Requirements 3.6
func (s *endUserService) GetUsage(
	ctx context.Context,
	endUserID string,
	start, end time.Time,
) (*Usage, error) {
	if endUserID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("end user ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("End user ID is required"),
		)
	}

	// Get end user to retrieve external_id
	endUser, err := s.GetEndUser(ctx, endUserID)
	if err != nil {
		return nil, err
	}

	// Calculate year/month bounds for the date range
	startYear, startMonth := start.Year(), int(start.Month())
	endYear, endMonth := end.Year(), int(end.Month())

	// Query ClickHouse for verifications
	// Use the new end-user billable table that includes external_id
	// Query across all months in the date range
	verificationsQuery := `
		SELECT sum(count) as count
		FROM default.end_user_billable_verifications_per_month_v1
		WHERE workspace_id = ?
		AND external_id = ?
		AND (
			(year > ? OR (year = ? AND month >= ?)) AND
			(year < ? OR (year = ? AND month <= ?))
		)
	`

	var verifications int64
	err = s.ch.Conn().QueryRow(
		ctx,
		verificationsQuery,
		endUser.WorkspaceID,
		endUser.ExternalID,
		// Start condition: (year > startYear OR (year = startYear AND month >= startMonth))
		startYear, startYear, startMonth,
		// End condition: (year < endYear OR (year = endYear AND month <= endMonth))
		endYear, endYear, endMonth,
	).Scan(&verifications)
	if err != nil && err.Error() != "sql: no rows in result set" {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to query verifications: %v", err)),
			fault.Public("Failed to retrieve usage data"),
		)
	}

	return &Usage{
		ExternalID:     endUser.ExternalID,
		Verifications:  verifications,
		KeysWithAccess: 0,
	}, nil
}

// dbEndUserToEndUser converts a database model to a domain model
func dbEndUserToEndUser(dbEndUser *db.BillingEndUser) (*EndUser, error) {
	var metadata map[string]string
	if len(dbEndUser.Metadata) > 0 {
		if err := json.Unmarshal(dbEndUser.Metadata, &metadata); err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to deserialize metadata: %v", err)),
				fault.Public("Failed to parse end user metadata"),
			)
		}
	}

	updatedAt := time.UnixMilli(dbEndUser.CreatedAtM)
	if dbEndUser.UpdatedAtM.Valid {
		updatedAt = time.UnixMilli(dbEndUser.UpdatedAtM.Int64)
	}

	email := ""
	if dbEndUser.Email.Valid {
		email = dbEndUser.Email.String
	}

	name := ""
	if dbEndUser.Name.Valid {
		name = dbEndUser.Name.String
	}

	subscriptionID := ""
	if dbEndUser.StripeSubscriptionID.Valid {
		subscriptionID = dbEndUser.StripeSubscriptionID.String
	}

	return &EndUser{
		ID:                   dbEndUser.ID,
		WorkspaceID:          dbEndUser.WorkspaceID,
		ExternalID:           dbEndUser.ExternalID,
		PricingModelID:       dbEndUser.PricingModelID,
		StripeCustomerID:     dbEndUser.StripeCustomerID,
		StripeSubscriptionID: subscriptionID,
		Email:                email,
		Name:                 name,
		Metadata:             metadata,
		CreatedAt:            time.UnixMilli(dbEndUser.CreatedAtM),
		UpdatedAt:            updatedAt,
	}, nil
}
