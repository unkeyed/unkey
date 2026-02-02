package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// PricingModelService manages pricing configurations for billing end users.
type PricingModelService interface {
	// CreatePricingModel creates new pricing model with validation and single currency enforcement
	CreatePricingModel(ctx context.Context, req *CreatePricingModelRequest) (*PricingModel, error)

	// GetPricingModel retrieves pricing model by ID
	GetPricingModel(ctx context.Context, id string) (*PricingModel, error)

	// ListPricingModels lists pricing models for workspace
	ListPricingModels(ctx context.Context, workspaceID string) ([]*PricingModel, error)

	// UpdatePricingModel creates new version of pricing model
	UpdatePricingModel(ctx context.Context, id string, req *UpdatePricingModelRequest) (*PricingModel, error)

	// DeletePricingModel soft deletes pricing model
	DeletePricingModel(ctx context.Context, id string) error

	// CalculateCharges calculates charges for usage based on pricing model
	CalculateCharges(ctx context.Context, model *PricingModel, verifications, keysWithAccess int64) (int64, error)
}

// PricingModel represents a pricing configuration for billing end users
type PricingModel struct {
	ID                    string
	WorkspaceID           string
	Name                  string
	Currency              string
	VerificationUnitPrice float64 // dollars (can be fractional, e.g., 0.001 for 0.1 cent)
	KeyAccessUnitPrice    float64 // dollars per unique key with VALID=true verification
	CreditUnitPrice       float64 // dollars per credit consumed
	TieredPricing         *TieredPricing
	Version               int32
	Active                bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// TieredPricing defines volume-based pricing tiers
type TieredPricing struct {
	Tiers []PricingTier `json:"tiers"`
}

// PricingTier represents a single pricing tier
type PricingTier struct {
	UpTo      int64   `json:"upTo"`      // 0 means unlimited
	UnitPrice float64 `json:"unitPrice"` // dollars (can be fractional)
}

// CreatePricingModelRequest contains parameters for creating a pricing model
type CreatePricingModelRequest struct {
	WorkspaceID           string
	Name                  string
	Currency              string
	VerificationUnitPrice float64
	KeyAccessUnitPrice    float64
	CreditUnitPrice       float64
	TieredPricing         *TieredPricing
}

// UpdatePricingModelRequest contains parameters for updating a pricing model
type UpdatePricingModelRequest struct {
	Name                  string
	VerificationUnitPrice float64
	KeyAccessUnitPrice    float64
	CreditUnitPrice       float64
	TieredPricing         *TieredPricing
}

// pricingModelService implements PricingModelService
type pricingModelService struct {
	db db.Database
}

// NewPricingModelService creates a new pricing model service
func NewPricingModelService(database db.Database) PricingModelService {
	return &pricingModelService{
		db: database,
	}
}

// CreatePricingModel creates a new pricing model with validation and single currency enforcement.
//
// Validates: Requirements 2.1, 2.2, 2.6
func (s *pricingModelService) CreatePricingModel(
	ctx context.Context,
	req *CreatePricingModelRequest,
) (*PricingModel, error) {
	// Validate required fields
	if req.WorkspaceID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("workspace ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Workspace ID is required"),
		)
	}

	if req.Name == "" {
		return nil, fault.Wrap(
			fmt.Errorf("name is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Pricing model name is required"),
		)
	}

	if req.Currency == "" {
		return nil, fault.Wrap(
			fmt.Errorf("currency is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Currency is required"),
		)
	}

	// Validate currency code (ISO 4217)
	if len(req.Currency) != 3 {
		return nil, fault.Wrap(
			fmt.Errorf("invalid currency code: %s", req.Currency),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Currency must be a valid 3-letter ISO code"),
		)
	}

	if req.VerificationUnitPrice < 0 {
		return nil, fault.Wrap(
			fmt.Errorf("verification unit price cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Verification unit price must be non-negative"),
		)
	}

	if req.KeyAccessUnitPrice < 0 {
		return nil, fault.Wrap(
			fmt.Errorf("key access unit price cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Key access unit price must be non-negative"),
		)
	}

	if req.CreditUnitPrice < 0 {
		return nil, fault.Wrap(
			fmt.Errorf("credit unit price cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Credit unit price must be non-negative"),
		)
	}

	// Validate tiered pricing if provided
	if req.TieredPricing != nil {
		if err := validateTieredPricing(req.TieredPricing); err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Invalid tiered pricing configuration"),
			)
		}
	}

	// Enforce single currency per workspace
	existingCurrency, err := db.Query.PricingModelFindWorkspaceCurrency(ctx, s.db.RO(), req.WorkspaceID)
	if err != nil && err != sql.ErrNoRows {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to check workspace currency: %v", err)),
			fault.Public("Failed to validate currency"),
		)
	}

	if err == nil && existingCurrency != req.Currency {
		return nil, fault.Wrap(
			fmt.Errorf("workspace already uses currency %s", existingCurrency),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public(fmt.Sprintf("All pricing models in this workspace must use %s", existingCurrency)),
		)
	}

	// Serialize tiered pricing to JSON
	var tieredPricingJSON []byte
	if req.TieredPricing != nil {
		tieredPricingJSON, err = json.Marshal(req.TieredPricing)
		if err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to serialize tiered pricing: %v", err)),
				fault.Public("Failed to save pricing configuration"),
			)
		}
	}

	// Create pricing model
	now := time.Now().UnixMilli()
	modelID := uid.New("pricing_model")

	insertParams := db.PricingModelInsertParams{
		ID:                    modelID,
		WorkspaceID:           req.WorkspaceID,
		Name:                  req.Name,
		Currency:              req.Currency,
		VerificationUnitPrice: fmt.Sprintf("%.8f", req.VerificationUnitPrice),
		KeyAccessUnitPrice:    fmt.Sprintf("%.8f", req.KeyAccessUnitPrice),
		CreditUnitPrice:       fmt.Sprintf("%.8f", req.CreditUnitPrice),
		TieredPricing:         tieredPricingJSON,
		Version:               1,
		Active:                true,
		CreatedAtM:            now,
	}

	err = db.Query.PricingModelInsert(ctx, s.db.RW(), insertParams)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to create pricing model: %v", err)),
			fault.Public("Failed to create pricing model"),
		)
	}

	return &PricingModel{
		ID:                    modelID,
		WorkspaceID:           req.WorkspaceID,
		Name:                  req.Name,
		Currency:              req.Currency,
		VerificationUnitPrice: req.VerificationUnitPrice,
		KeyAccessUnitPrice:    req.KeyAccessUnitPrice,
		CreditUnitPrice:       req.CreditUnitPrice,
		TieredPricing:         req.TieredPricing,
		Version:               1,
		Active:                true,
		CreatedAt:             time.UnixMilli(now),
		UpdatedAt:             time.UnixMilli(now),
	}, nil
}

// GetPricingModel retrieves a pricing model by ID.
//
// Validates: Requirements 2.2
func (s *pricingModelService) GetPricingModel(ctx context.Context, id string) (*PricingModel, error) {
	if id == "" {
		return nil, fault.Wrap(
			fmt.Errorf("pricing model ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Pricing model ID is required"),
		)
	}

	dbModel, err := db.Query.PricingModelFindById(ctx, s.db.RO(), id)
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
			fault.Internal(fmt.Sprintf("failed to find pricing model: %v", err)),
			fault.Public("Failed to retrieve pricing model"),
		)
	}

	return dbModelToPricingModel(&dbModel)
}

// ListPricingModels lists all pricing models for a workspace.
//
// Validates: Requirements 2.2
func (s *pricingModelService) ListPricingModels(ctx context.Context, workspaceID string) ([]*PricingModel, error) {
	if workspaceID == "" {
		return nil, fault.Wrap(
			fmt.Errorf("workspace ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Workspace ID is required"),
		)
	}

	dbModels, err := db.Query.PricingModelListByWorkspaceId(ctx, s.db.RO(), workspaceID)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to list pricing models: %v", err)),
			fault.Public("Failed to retrieve pricing models"),
		)
	}

	models := make([]*PricingModel, 0, len(dbModels))
	for _, dbModel := range dbModels {
		model, err := dbModelToPricingModel(&dbModel)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}

	return models, nil
}

// UpdatePricingModel creates a new version of the pricing model.
// Existing invoices continue using the previous version.
//
// Validates: Requirements 2.6, 2.7
func (s *pricingModelService) UpdatePricingModel(
	ctx context.Context,
	id string,
	req *UpdatePricingModelRequest,
) (*PricingModel, error) {
	if id == "" {
		return nil, fault.Wrap(
			fmt.Errorf("pricing model ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Pricing model ID is required"),
		)
	}

	// Get existing model
	existing, err := s.GetPricingModel(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate new values
	if req.Name == "" {
		req.Name = existing.Name
	}

	if req.VerificationUnitPrice < 0 {
		return nil, fault.Wrap(
			fmt.Errorf("verification unit price cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Verification unit price must be non-negative"),
		)
	}

	if req.KeyAccessUnitPrice < 0 {
		return nil, fault.Wrap(
			fmt.Errorf("key access unit price cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Key access unit price must be non-negative"),
		)
	}

	if req.CreditUnitPrice < 0 {
		return nil, fault.Wrap(
			fmt.Errorf("credit unit price cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Credit unit price must be non-negative"),
		)
	}

	// Validate tiered pricing if provided
	if req.TieredPricing != nil {
		if err := validateTieredPricing(req.TieredPricing); err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Invalid tiered pricing configuration"),
			)
		}
	}

	// Serialize tiered pricing to JSON
	var tieredPricingJSON []byte
	if req.TieredPricing != nil {
		tieredPricingJSON, err = json.Marshal(req.TieredPricing)
		if err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to serialize tiered pricing: %v", err)),
				fault.Public("Failed to save pricing configuration"),
			)
		}
	}

	// Update with new version
	now := time.Now().UnixMilli()
	newVersion := existing.Version + 1

	updateParams := db.PricingModelUpdateParams{
		Name:                  req.Name,
		VerificationUnitPrice: fmt.Sprintf("%.8f", req.VerificationUnitPrice),
		KeyAccessUnitPrice:    fmt.Sprintf("%.8f", req.KeyAccessUnitPrice),
		CreditUnitPrice:       fmt.Sprintf("%.8f", req.CreditUnitPrice),
		TieredPricing:         tieredPricingJSON,
		Version:               newVersion,
		UpdatedAtM:            sql.NullInt64{Int64: now, Valid: true},
		ID:                    id,
	}

	err = db.Query.PricingModelUpdate(ctx, s.db.RW(), updateParams)
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to update pricing model: %v", err)),
			fault.Public("Failed to update pricing model"),
		)
	}

	return &PricingModel{
		ID:                    existing.ID,
		WorkspaceID:           existing.WorkspaceID,
		Name:                  req.Name,
		Currency:              existing.Currency,
		VerificationUnitPrice: req.VerificationUnitPrice,
		KeyAccessUnitPrice:    req.KeyAccessUnitPrice,
		CreditUnitPrice:       req.CreditUnitPrice,
		TieredPricing:         req.TieredPricing,
		Version:               newVersion,
		Active:                existing.Active,
		CreatedAt:             existing.CreatedAt,
		UpdatedAt:             time.UnixMilli(now),
	}, nil
}

// DeletePricingModel soft deletes a pricing model.
// Prevents deletion if end users are assigned to it.
//
// Validates: Requirements 2.8
func (s *pricingModelService) DeletePricingModel(ctx context.Context, id string) error {
	if id == "" {
		return fault.Wrap(
			fmt.Errorf("pricing model ID is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Pricing model ID is required"),
		)
	}

	// Check if any end users are using this pricing model
	count, err := db.Query.PricingModelCountEndUsers(ctx, s.db.RO(), id)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to check end user references: %v", err)),
			fault.Public("Failed to validate pricing model deletion"),
		)
	}

	if count > 0 {
		return fault.Wrap(
			fmt.Errorf("pricing model has %d active end users", count),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public(fmt.Sprintf("Cannot delete pricing model with %d active end users", count)),
		)
	}

	// Soft delete the pricing model
	now := time.Now().UnixMilli()
	err = db.Query.PricingModelSoftDelete(ctx, s.db.RW(), db.PricingModelSoftDeleteParams{
		DeletedAtM: sql.NullInt64{Int64: now, Valid: true},
		UpdatedAtM: sql.NullInt64{Int64: now, Valid: true},
		ID:         id,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal(fmt.Sprintf("failed to delete pricing model: %v", err)),
			fault.Public("Failed to delete pricing model"),
		)
	}

	return nil
}

// CalculateCharges calculates the total charges for usage based on the pricing model.
// Supports both flat pricing and tiered pricing.
// Returns charges in cents (rounded) for Stripe integration.
//
// Validates: Requirements 5.3
func (s *pricingModelService) CalculateCharges(
	ctx context.Context,
	model *PricingModel,
	verifications, keysWithAccess int64,
) (int64, error) {
	if model == nil {
		return 0, fault.Wrap(
			fmt.Errorf("pricing model is required"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Pricing model is required"),
		)
	}

	if verifications < 0 {
		return 0, fault.Wrap(
			fmt.Errorf("verifications cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Verification count must be non-negative"),
		)
	}

	if keysWithAccess < 0 {
		return 0, fault.Wrap(
			fmt.Errorf("keys with access cannot be negative"),
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Keys with access count must be non-negative"),
		)
	}

	// Use tiered pricing if configured, otherwise use flat pricing
	if model.TieredPricing != nil && len(model.TieredPricing.Tiers) > 0 {
		return calculateTieredCharges(model.TieredPricing, verifications, keysWithAccess, model)
	}

	// Flat pricing - use float64 for fractional pricing
	// Calculate charges for each component:
	// - keyAccessCharges: charged per unique key with VALID=true verification
	// - verificationCharges: charged per verification
	// - creditCharges: charged per credit (based on verifications)
	keyAccessCharges := float64(keysWithAccess) * model.KeyAccessUnitPrice
	verificationCharges := float64(verifications) * model.VerificationUnitPrice
	creditCharges := float64(verifications) * model.CreditUnitPrice
	totalCharges := keyAccessCharges + verificationCharges + creditCharges

	// Round to nearest cent (Stripe uses cents)
	totalInCents := int64(math.Round(totalCharges * 100))
	return totalInCents, nil
}

// calculateTieredCharges calculates charges using tiered pricing
// Returns charges in cents (rounded) for Stripe integration.
// Tiered pricing applies to verifications + rateLimits total.
// Key access and credit charges are added as flat rates on top.
func calculateTieredCharges(tp *TieredPricing, verifications, keysWithAccess int64, model *PricingModel) (int64, error) {
	totalUsage := verifications
	var totalCharge float64
	var usedSoFar int64

	for _, tier := range tp.Tiers {
		if usedSoFar >= totalUsage {
			break
		}

		var tierUsage int64
		if tier.UpTo == 0 {
			// Unlimited tier - use all remaining
			tierUsage = totalUsage - usedSoFar
		} else {
			// Limited tier - use up to the tier limit minus what we've already used
			tierLimit := tier.UpTo - usedSoFar
			remaining := totalUsage - usedSoFar
			tierUsage = min(remaining, tierLimit)
		}

		totalCharge += float64(tierUsage) * tier.UnitPrice
		usedSoFar += tierUsage
	}

	// Add key access charges (flat rate, not tiered)
	totalCharge += float64(keysWithAccess) * model.KeyAccessUnitPrice

	// Add credit charges (flat rate, not tiered)
	totalCharge += float64(verifications) * model.CreditUnitPrice

	// Round to nearest cent (Stripe uses cents)
	return int64(math.Round(totalCharge * 100)), nil
}

// validateTieredPricing validates the tiered pricing configuration
func validateTieredPricing(tp *TieredPricing) error {
	if len(tp.Tiers) == 0 {
		return fmt.Errorf("tiered pricing must have at least one tier")
	}

	// Check that tiers are in ascending order
	for i := 0; i < len(tp.Tiers)-1; i++ {
		if tp.Tiers[i].UpTo >= tp.Tiers[i+1].UpTo && tp.Tiers[i+1].UpTo != 0 {
			return fmt.Errorf("tier limits must be in ascending order")
		}
	}

	// Check that only the last tier can have UpTo = 0 (unlimited)
	for i := 0; i < len(tp.Tiers)-1; i++ {
		if tp.Tiers[i].UpTo == 0 {
			return fmt.Errorf("only the last tier can have unlimited (0) limit")
		}
	}

	// Check that unit prices are non-negative (supports fractional pricing)
	for i, tier := range tp.Tiers {
		if tier.UnitPrice < 0 {
			return fmt.Errorf("tier %d unit price cannot be negative", i)
		}
	}

	return nil
}

// dbModelToPricingModel converts a database model to a domain model
func dbModelToPricingModel(dbModel *db.PricingModel) (*PricingModel, error) {
	var tieredPricing *TieredPricing
	if len(dbModel.TieredPricing) > 0 {
		if err := json.Unmarshal(dbModel.TieredPricing, &tieredPricing); err != nil {
			return nil, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal(fmt.Sprintf("failed to deserialize tiered pricing: %v", err)),
				fault.Public("Failed to parse pricing configuration"),
			)
		}
	}

	updatedAt := time.UnixMilli(dbModel.CreatedAtM)
	if dbModel.UpdatedAtM.Valid {
		updatedAt = time.UnixMilli(dbModel.UpdatedAtM.Int64)
	}

	return &PricingModel{
		ID:                    dbModel.ID,
		WorkspaceID:           dbModel.WorkspaceID,
		Name:                  dbModel.Name,
		Currency:              dbModel.Currency,
		VerificationUnitPrice: parseUnitPrice(dbModel.VerificationUnitPrice),
		KeyAccessUnitPrice:    parseUnitPrice(dbModel.KeyAccessUnitPrice),
		CreditUnitPrice:       parseUnitPrice(dbModel.CreditUnitPrice),
		TieredPricing:         tieredPricing,
		Version:               dbModel.Version,
		Active:                dbModel.Active,
		CreatedAt:             time.UnixMilli(dbModel.CreatedAtM),
		UpdatedAt:             updatedAt,
	}, nil
}

// min returns the minimum of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// parseUnitPrice parses a unit price string to float64
func parseUnitPrice(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}
