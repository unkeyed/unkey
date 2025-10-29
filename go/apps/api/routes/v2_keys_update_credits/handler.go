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

type Request = openapi.V2KeysUpdateCreditsRequestBody
type Response = openapi.V2KeysUpdateCreditsResponseBody

// Handler implements zen.Route interface for the v2 keys.updateCredits endpoint
type Handler struct {
	Logger       logging.Logger
	DB           db.Database
	Keys         keys.KeyService
	Auditlogs    auditlogs.AuditLogService
	KeyCache     cache.Cache[string, db.CachedKeyData]
	UsageLimiter usagelimiter.Service
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.updateCredits"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.updateCredits")

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

	key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key does not exist"),
				fault.Public("We could not find the requested key."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve Key information."),
		)
	}

	// Validate key belongs to authorized workspace
	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	// Permission check
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.UpdateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.Api.ID,
			Action:       rbac.UpdateKey,
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

	// Check if credits exist - from new credits table or legacy fields
	hasNewCredits := key.CreditID.Valid
	hasLegacyCredits := key.RemainingRequests.Valid
	hasCredits := hasNewCredits || hasLegacyCredits

	if (req.Operation == openapi.Decrement || req.Operation == openapi.Increment) && !hasCredits {
		return fault.New("wrong operation usage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("You cannot increment or decrement a key with unlimited credits."),
		)
	}

	// The only errors that can be returned here are isNull or notSpecified
	// which firstly is wanted and secondly doesn't matter
	reqVal, _ := req.Value.Get()

	updatedCredits, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.Credit, error) {
		now := time.Now()
		var result db.Credit
		var auditLogMessage string

		// Execute database operation based on which system the key uses
		if hasNewCredits {
			// Work with new credits table
			creditID := key.CreditID.String
			currentRemaining := key.CreditRemaining.Int32

			switch req.Operation {
			case openapi.Set:
				if req.Value.IsNull() {
					// Setting to null means unlimited - delete the credit record
					if err := db.Query.DeleteCredit(ctx, tx, creditID); err != nil {
						return db.Credit{}, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to update key credits."),
						)
					}
					auditLogMessage = "unlimited"
					result = db.Credit{}
				} else {
					remaining := int32(reqVal)
					if err := db.Query.UpsertCredit(ctx, tx, db.UpsertCreditParams{
						ID:                    creditID,
						WorkspaceID:           key.WorkspaceID,
						KeyID:                 sql.NullString{Valid: true, String: key.ID},
						IdentityID:            sql.NullString{Valid: false},
						Remaining:             remaining,
						RefillDay:             sql.NullInt16{Valid: false},
						RefillAmount:          sql.NullInt32{Valid: false},
						CreatedAt:             now.UnixMilli(),
						UpdatedAt:             sql.NullInt64{Valid: true, Int64: now.UnixMilli()},
						RefilledAt:            sql.NullInt64{Valid: false},
						RemainingSpecified:    1,
						RefillDaySpecified:    0,
						RefillAmountSpecified: 0,
					}); err != nil {
						return db.Credit{}, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to update key credits."),
						)
					}
					result.ID = creditID
					result.Remaining = remaining
					result.RefillDay = key.CreditRefillDay
					result.RefillAmount = key.CreditRefillAmount
					auditLogMessage = fmt.Sprintf("%d", remaining)
				}

			case openapi.Increment:
				if err := db.Query.UpdateCreditIncrement(ctx, tx, db.UpdateCreditIncrementParams{
					ID:      creditID,
					Credits: int32(reqVal),
				}); err != nil {
					return db.Credit{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to update key credits."),
					)
				}
				newRemaining := currentRemaining + int32(reqVal)
				result.ID = creditID
				result.Remaining = newRemaining
				result.RefillDay = key.CreditRefillDay
				result.RefillAmount = key.CreditRefillAmount
				auditLogMessage = fmt.Sprintf("%d", newRemaining)

			case openapi.Decrement:
				if err := db.Query.UpdateCreditDecrement(ctx, tx, db.UpdateCreditDecrementParams{
					ID:      creditID,
					Credits: int32(reqVal),
				}); err != nil {
					return db.Credit{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to update key credits."),
					)
				}
				newRemaining := currentRemaining - int32(reqVal)
				if newRemaining < 0 {
					newRemaining = 0
				}
				result.ID = creditID
				result.Remaining = newRemaining
				result.RefillDay = key.CreditRefillDay
				result.RefillAmount = key.CreditRefillAmount
				auditLogMessage = fmt.Sprintf("%d", newRemaining)

			default:
				return db.Credit{}, fault.New("invalid operation",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal(fmt.Sprintf("invalid operation: %s", req.Operation)),
					fault.Public("Invalid operation specified."),
				)
			}
		} else if hasLegacyCredits {
			// Work with legacy keys.remaining_requests field
			currentRemaining := key.RemainingRequests.Int32

			switch req.Operation {
			case openapi.Set:
				if req.Value.IsNull() {
					// Setting to null means unlimited
					if err := db.Query.UpdateKey(ctx, tx, db.UpdateKeyParams{
						ID:                         key.ID,
						Now:                        sql.NullInt64{Valid: true, Int64: now.UnixMilli()},
						NameSpecified:              0,
						IdentityIDSpecified:        0,
						EnabledSpecified:           0,
						MetaSpecified:              0,
						ExpiresSpecified:           0,
						RemainingRequests:          sql.NullInt32{Valid: false},
						RemainingRequestsSpecified: 1,
					}); err != nil {
						return db.Credit{}, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to update key credits."),
						)
					}
					auditLogMessage = "unlimited"
					result = db.Credit{}
				} else {
					remaining := int32(reqVal)
					if err := db.Query.UpdateKey(ctx, tx, db.UpdateKeyParams{
						ID:                         key.ID,
						Now:                        sql.NullInt64{Valid: true, Int64: now.UnixMilli()},
						NameSpecified:              0,
						IdentityIDSpecified:        0,
						EnabledSpecified:           0,
						MetaSpecified:              0,
						ExpiresSpecified:           0,
						RemainingRequests:          sql.NullInt32{Valid: true, Int32: remaining},
						RemainingRequestsSpecified: 1,
					}); err != nil {
						return db.Credit{}, fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"),
							fault.Public("Failed to update key credits."),
						)
					}
					result.Remaining = remaining
					result.RefillDay = key.RefillDay
					result.RefillAmount = key.RefillAmount
					auditLogMessage = fmt.Sprintf("%d", remaining)
				}

			case openapi.Increment:
				if err := db.Query.UpdateKeyCreditsIncrement(ctx, tx, db.UpdateKeyCreditsIncrementParams{
					ID:      key.ID,
					Credits: sql.NullInt32{Valid: true, Int32: int32(reqVal)},
				}); err != nil {
					return db.Credit{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to update key credits."),
					)
				}
				newRemaining := currentRemaining + int32(reqVal)
				result.Remaining = newRemaining
				result.RefillDay = key.RefillDay
				result.RefillAmount = key.RefillAmount
				auditLogMessage = fmt.Sprintf("%d", newRemaining)

			case openapi.Decrement:
				if err := db.Query.UpdateKeyCreditsDecrement(ctx, tx, db.UpdateKeyCreditsDecrementParams{
					ID:      key.ID,
					Credits: sql.NullInt32{Valid: true, Int32: int32(reqVal)},
				}); err != nil {
					return db.Credit{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to update key credits."),
					)
				}
				newRemaining := currentRemaining - int32(reqVal)
				if newRemaining < 0 {
					newRemaining = 0
				}
				result.Remaining = newRemaining
				result.RefillDay = key.RefillDay
				result.RefillAmount = key.RefillAmount
				auditLogMessage = fmt.Sprintf("%d", newRemaining)

			default:
				return db.Credit{}, fault.New("invalid operation",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal(fmt.Sprintf("invalid operation: %s", req.Operation)),
					fault.Public("Invalid operation specified."),
				)
			}
		} else {
			// Key currently has unlimited credits (no credit record)
			// Only SET operation with non-null value is allowed
			if req.Operation == openapi.Set && !req.Value.IsNull() {
				// Create a new credit record in the new credits table
				creditID := uid.New(uid.CreditPrefix)
				remaining := int32(reqVal)

				if err := db.Query.InsertCredit(ctx, tx, db.InsertCreditParams{
					ID:           creditID,
					WorkspaceID:  key.WorkspaceID,
					KeyID:        sql.NullString{Valid: true, String: key.ID},
					IdentityID:   sql.NullString{Valid: false},
					Remaining:    remaining,
					RefillDay:    sql.NullInt16{Valid: false},
					RefillAmount: sql.NullInt32{Valid: false},
					CreatedAt:    now.UnixMilli(),
					UpdatedAt:    sql.NullInt64{Valid: true, Int64: now.UnixMilli()},
					RefilledAt:   sql.NullInt64{Valid: false},
				}); err != nil {
					return db.Credit{}, fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to create key credits."),
					)
				}

				result.ID = creditID
				result.Remaining = remaining
				auditLogMessage = fmt.Sprintf("%d", remaining)
			} else {
				return db.Credit{}, fault.New("no credits configured",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("key has no credits configured"),
					fault.Public("This key has unlimited credits."),
				)
			}
		}

		// Create audit log once at the end
		if err := h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.KeyUpdateEvent,
				Display:     fmt.Sprintf("Updated Key %s, set remaining to %s.", key.ID, auditLogMessage),
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          key.KeyAuthID,
						Type:        auditlog.KeySpaceResourceType,
						Name:        "",
						DisplayName: "",
						Meta:        nil,
					},
					{
						ID:          key.ID,
						Type:        auditlog.KeyResourceType,
						Name:        key.Name.String,
						DisplayName: key.Name.String,
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

	responseData := openapi.Credits{
		Refill:    nil,
		Remaining: nullable.NewNullNullable[int64](),
	}

	// If set to unlimited (null), remaining stays null
	if !(req.Operation == openapi.Set && req.Value.IsNull()) {
		// Has credits - set the value
		responseData.Remaining = nullable.NewNullableWithValue(int64(updatedCredits.Remaining))

		// Add refill config if exists
		if updatedCredits.RefillAmount.Valid {
			var day *int
			interval := openapi.Daily

			if updatedCredits.RefillDay.Valid {
				interval = openapi.Monthly
				day = ptr.P(int(updatedCredits.RefillDay.Int16))
			}

			responseData.Refill = &openapi.CreditsRefill{
				Amount:    int64(updatedCredits.RefillAmount.Int32),
				Interval:  interval,
				RefillDay: day,
			}
		}
	}

	h.KeyCache.Remove(ctx, key.Hash)

	// Invalidate cache - need to use the credit ID that was valid before the update
	// For set-to-unlimited, we delete the credit, so we need the old credit ID
	// For other operations, updatedCredits.ID will be set
	// Legacy credits don't have a credit ID in the credits table to invalidate
	creditIDToInvalidate := ""
	if updatedCredits.ID != "" {
		creditIDToInvalidate = updatedCredits.ID
	} else if hasNewCredits {
		creditIDToInvalidate = key.CreditID.String
	}

	if creditIDToInvalidate != "" {
		if err := h.UsageLimiter.Invalidate(ctx, creditIDToInvalidate); err != nil {
			h.Logger.Error("Failed to invalidate usage limit",
				"error", err.Error(),
				"credit_id", creditIDToInvalidate,
			)
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
