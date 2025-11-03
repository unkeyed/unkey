package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/array"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2IdentitiesUpdateCreditsRequestBody
type Response = openapi.V2IdentitiesUpdateCreditsResponseBody

// Handler implements zen.Route interface for the v2 identities.updateCredits endpoint
type Handler struct {
	Logger       logging.Logger
	DB           db.Database
	Keys         keys.KeyService
	Auditlogs    auditlogs.AuditLogService
	UsageLimiter usagelimiter.Service
	KeyCache     cache.Cache[string, db.CachedKeyData]
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/identities.updateCredits"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/identities.updateCredits")

	// Authentication
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Find the identity
	identity, err := db.Query.FindIdentity(ctx, h.DB.RO(), db.FindIdentityParams{
		Identity:    req.Identity,
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Deleted:     false,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Identity.NotFound.URN()),
				fault.Internal("identity does not exist"),
				fault.Public("This identity does not exist."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve identity information."),
		)
	}

	// Permission check
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   "*",
			Action:       rbac.UpdateIdentity,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Identity,
			ResourceID:   identity.ID,
			Action:       rbac.UpdateIdentity,
		}),
	)))
	if err != nil {
		return err
	}

	if (req.Operation == openapi.Decrement || req.Operation == openapi.Increment) && (!req.Value.IsSpecified() || req.Value.IsNull()) {
		return fault.New("wrong operation usage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("When specifying an increment or decrement operation, a value must be provided."),
		)
	}

	// Check if credits exist from the FindIdentity JOIN
	hasCredits := identity.CreditID.Valid
	isUnlimited := !hasCredits

	var currentCredits db.Credit
	if hasCredits {
		currentCredits = db.Credit{
			ID:           identity.CreditID.String,
			WorkspaceID:  identity.WorkspaceID,
			IdentityID:   sql.NullString{Valid: true, String: identity.ID},
			KeyID:        sql.NullString{Valid: false},
			Remaining:    identity.CreditRemaining.Int32,
			RefillDay:    identity.CreditRefillDay,
			RefillAmount: identity.CreditRefillAmount,
			RefilledAt:   identity.CreditRefilledAt,
		}
	}

	if (req.Operation == openapi.Decrement || req.Operation == openapi.Increment) && isUnlimited {
		return fault.New("wrong operation usage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("You cannot increment or decrement an identity with unlimited credits."),
		)
	}

	// The only errors that can be returned here are isNull or notSpecified
	// which firstly is wanted and secondly doesn't matter
	reqVal, _ := req.Value.Get()

	// Validate that the value fits in int32
	if req.Value.IsSpecified() && !req.Value.IsNull() {
		if reqVal > 2147483647 || reqVal < -2147483648 {
			return fault.New("invalid value",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Value must be between -2147483648 and 2147483647"),
			)
		}
	}

	updatedCredits, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.Credit, error) {
		now := time.Now()
		var result db.Credit
		var auditLogMessage string

		// Execute database operation
		switch req.Operation {
		case openapi.Set:
			if req.Value.IsNull() {
				// Setting to null means unlimited - delete the credit record
				if hasCredits {
					if err := db.Query.DeleteCredit(ctx, tx, currentCredits.ID); err != nil {
						return db.Credit{}, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to update identity credits."),
						)
					}
				}
				auditLogMessage = "unlimited"
				result = db.Credit{} // Empty credit = unlimited
			} else {
				// Set to a specific value using upsert (handles both insert and update, solves concurrency)
				remaining := int32(reqVal)
				creditID := currentCredits.ID
				if !hasCredits {
					creditID = uid.New(uid.CreditPrefix)
				}

				if err := db.Query.UpsertCredit(ctx, tx, db.UpsertCreditParams{
					ID:                    creditID,
					WorkspaceID:           identity.WorkspaceID,
					IdentityID:            sql.NullString{Valid: true, String: identity.ID},
					KeyID:                 sql.NullString{Valid: false},
					Remaining:             remaining,
					RefillDay:             sql.NullInt16{Valid: false},
					RefillAmount:          sql.NullInt32{Valid: false},
					CreatedAt:             now.UnixMilli(),
					UpdatedAt:             sql.NullInt64{Valid: true, Int64: now.UnixMilli()},
					RefilledAt:            sql.NullInt64{Valid: false},
					RemainingSpecified:    1, // Always update remaining
					RefillDaySpecified:    0, // Don't touch existing refill_day
					RefillAmountSpecified: 0, // Don't touch existing refill_amount
				}); err != nil {
					return db.Credit{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to update identity credits."),
					)
				}

				// Start with existing credit data to preserve refill metadata
				if hasCredits {
					result = currentCredits
					result.Remaining = remaining
				} else {
					// New credit record without refill configuration
					result = db.Credit{
						ID:          creditID,
						WorkspaceID: identity.WorkspaceID,
						IdentityID:  sql.NullString{Valid: true, String: identity.ID},
						KeyID:       sql.NullString{Valid: false},
						Remaining:   remaining,
					}
				}
				auditLogMessage = fmt.Sprintf("%d", remaining)
			}

		case openapi.Increment:
			if err := db.Query.UpdateCreditIncrement(ctx, tx, db.UpdateCreditIncrementParams{
				ID:      currentCredits.ID,
				Credits: int32(reqVal),
			}); err != nil {
				return db.Credit{}, fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to update identity credits."),
				)
			}
			newRemaining := currentCredits.Remaining + int32(reqVal)
			result = currentCredits
			result.Remaining = newRemaining
			auditLogMessage = fmt.Sprintf("%d", newRemaining)

		case openapi.Decrement:
			if err := db.Query.UpdateCreditDecrement(ctx, tx, db.UpdateCreditDecrementParams{
				ID:      currentCredits.ID,
				Credits: int32(reqVal),
			}); err != nil {
				return db.Credit{}, fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to update identity credits."),
				)
			}
			newRemaining := currentCredits.Remaining - int32(reqVal)
			if newRemaining < 0 {
				newRemaining = 0
			}
			result = currentCredits
			result.Remaining = newRemaining
			auditLogMessage = fmt.Sprintf("%d", newRemaining)

		default:
			return db.Credit{}, fault.New("invalid operation",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(fmt.Sprintf("invalid operation: %s", req.Operation)),
				fault.Public("Invalid operation specified."),
			)
		}

		// Create audit log once at the end
		if err := h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.IdentityUpdateEvent,
				Display:     fmt.Sprintf("Updated Identity %s, set remaining to %s.", identity.ID, auditLogMessage),
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          identity.ID,
						Type:        auditlog.IdentityResourceType,
						Name:        identity.ExternalID,
						DisplayName: identity.ExternalID,
						Meta:        nil,
					},
				},
			},
		}); err != nil {
			return db.Credit{}, err
		}

		return result, nil
	})

	if err != nil {
		return err
	}

	// Build response
	responseData := openapi.Credits{
		Remaining: nullable.NewNullNullable[int64](),
		Refill:    nil,
	}

	// If set to unlimited (null), return null for remaining
	if req.Operation == openapi.Set && req.Value.IsNull() {

	} else {
		// Has credits
		responseData.Remaining = nullable.NewNullableWithValue(int64(updatedCredits.Remaining))

		// Add refill config if exists
		if updatedCredits.RefillAmount.Valid {
			var interval openapi.CreditInterval
			var refillDay *int
			if updatedCredits.RefillDay.Valid {
				interval = openapi.Monthly
				refillDay = ptr.P(int(updatedCredits.RefillDay.Int16))
			} else {
				interval = openapi.Daily
			}

			responseData.Refill = &openapi.CreditsRefill{
				Amount:    int64(updatedCredits.RefillAmount.Int32),
				RefillDay: refillDay,
				Interval:  interval,
			}
		}
	}

	// Invalidate usage limiter cache for the credit ID
	if identity.CreditID.Valid && identity.CreditID.String != "" {
		if err := h.UsageLimiter.Invalidate(ctx, identity.CreditID.String); err != nil {
			h.Logger.Error("Failed to invalidate usage limit",
				"error", err.Error(),
				"credit_id", identity.CreditID.String,
			)
		}
	}

	// Find and invalidate all keys belonging to this identity
	keys, err := db.Query.ListKeysByIdentityID(ctx, h.DB.RO(), sql.NullString{String: identity.ID, Valid: true})
	if err != nil {
		h.Logger.Error("Failed to find keys for identity",
			"error", err.Error(),
			"identity_id", identity.ID,
		)
	}

	if len(keys) > 0 {
		hashes := array.Map(keys, func(key db.ListKeysByIdentityIDRow) string {
			return key.Hash
		})
		h.KeyCache.Remove(ctx, hashes...)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
