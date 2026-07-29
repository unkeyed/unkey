package handler

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2KeysUpdateCreditsRequestBody
type Response = openapi.V2KeysUpdateCreditsResponseBody

// Handler implements zen.Route interface for the v2 keys.updateCredits endpoint
type Handler struct {
	DB           db.Database
	Auditlogs    auditlogs.AuditLogService
	KeyCache     cache.Cache[string, keysdb.CachedKeyData]
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
	// Authentication
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := db.Query.FindLiveKeyForCreditsByID(ctx, h.DB.RO(), req.KeyId)
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
	if key.WorkspaceID != principal.WorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	// Permission check
	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.UpdateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.ApiID,
			Action:       rbac.UpdateKey,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Keyspace(key.KeyAuthID).Key(req.KeyId),
			permissions.UpdateKey{},
		),
	))
	if err != nil {
		return err
	}

	if (req.Operation == openapi.Decrement || req.Operation == openapi.Increment) && (!req.Value.IsSpecified() || req.Value.IsNull()) {
		return fault.New("wrong operation usage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("When specifying an increment or decrement operation, a value must be provided."),
		)
	}

	if (req.Operation == openapi.Decrement || req.Operation == openapi.Increment) && !key.RemainingRequests.Valid {
		return fault.New("wrong operation usage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("You cannot increment or decrement a key with unlimited credits."),
		)
	}

	credits := sql.NullInt64{Int64: 0, Valid: false}

	// The only errors that can be returned here are isNull or notSpecified
	// which firstly is wanted and secondly doesn't matter
	reqVal, _ := req.Value.Get()

	// Value has been set as not null
	if !req.Value.IsNull() && req.Value.IsSpecified() {
		credits = sql.NullInt64{Int64: reqVal, Valid: true}
	}
	clearRefill := int64(0)
	if !credits.Valid {
		clearRefill = 1
	}

	keyAfterUpdate := key
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		var updateResult sql.Result
		switch req.Operation {
		case openapi.Set:
			updateResult, err = db.Query.UpdateKeyCreditsSet(ctx, tx, db.UpdateKeyCreditsSetParams{
				ID:                key.ID,
				Credits:           credits,
				ClearRefillAmount: clearRefill,
				ClearRefillDay:    clearRefill,
			})
			keyAfterUpdate.RemainingRequests = credits
			if !credits.Valid {
				keyAfterUpdate.RefillAmount = sql.NullInt64{}
				keyAfterUpdate.RefillDay = sql.NullInt16{}
			}
		case openapi.Increment:
			updateResult, err = db.Query.UpdateKeyCreditsIncrementReturning(ctx, tx, db.UpdateKeyCreditsIncrementReturningParams{
				ID:      key.ID,
				Credits: credits,
			})
		case openapi.Decrement:
			updateResult, err = db.Query.UpdateKeyCreditsDecrementReturning(ctx, tx, db.UpdateKeyCreditsDecrementReturningParams{
				ID:      key.ID,
				Credits: credits,
			})
		default:
			return fault.New("invalid operation",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(fmt.Sprintf("invalid operation: %s", req.Operation)),
				fault.Public("Invalid operation specified."),
			)
		}
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to update key credits."),
			)
		}

		rowsAffected, rowsAffectedErr := updateResult.RowsAffected()
		if rowsAffectedErr != nil {
			return fault.Wrap(rowsAffectedErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to verify updated key credits"),
				fault.Public("Failed to update key credits."),
			)
		}
		if rowsAffected == 0 {
			currentCredits, findErr := db.Query.FindLiveKeyCredits(ctx, tx, key.ID)
			switch {
			case db.IsNotFound(findErr):
				return fault.New("key got deleted before credits update",
					fault.Code(codes.Data.Key.NotFound.URN()),
					fault.Internal("key got deleted before update"),
					fault.Public("We could not find the requested key."),
				)
			case findErr != nil:
				return fault.Wrap(findErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("unable to verify key after credits update matched no rows"),
					fault.Public("Failed to update key credits."),
				)
			case req.Operation != openapi.Set && !currentCredits.Valid:
				return fault.New("key credits became unlimited before update",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Public("You cannot increment or decrement a key with unlimited credits."),
				)
			case req.Operation == openapi.Increment && credits.Int64 > math.MaxInt64-currentCredits.Int64:
				return fault.New("credits increment exceeds maximum",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("credits increment would exceed max int64"),
					fault.Public("The resulting credit balance exceeds the maximum supported value."),
				)
			default:
				// MySQL reports changed rows, so a valid no-op can report zero.
				keyAfterUpdate.RemainingRequests = currentCredits
			}
		}

		if req.Operation != openapi.Set && rowsAffected > 0 {
			keyAfterUpdate.RemainingRequests.Int64, err = updateResult.LastInsertId()
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("unable to read updated key credits"),
					fault.Public("Failed to update key credits."),
				)
			}
			keyAfterUpdate.RemainingRequests.Valid = true
		}

		remaining := "unlimited"
		if keyAfterUpdate.RemainingRequests.Valid {
			remaining = fmt.Sprintf("%d", keyAfterUpdate.RemainingRequests.Int64)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.KeyUpdateEvent,
				Display:       fmt.Sprintf("Updated Key %s, set remaining to %s.", key.ID, remaining),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
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
		})

		return err
	})

	if err != nil {
		return err
	}

	null := nullable.Nullable[int64]{}
	null.SetNull()

	responseData := openapi.KeyCreditsData{
		Refill:    nil,
		Remaining: null,
	}

	if keyAfterUpdate.RemainingRequests.Valid {
		responseData.Remaining = nullable.NewNullableWithValue(keyAfterUpdate.RemainingRequests.Int64)
	}

	if keyAfterUpdate.RefillAmount.Valid {
		var day int
		interval := openapi.KeyCreditsRefillIntervalDaily

		if keyAfterUpdate.RefillDay.Valid {
			interval = openapi.KeyCreditsRefillIntervalMonthly
			day = int(keyAfterUpdate.RefillDay.Int16)
		}

		responseData.Refill = &openapi.KeyCreditsRefill{
			Amount:    keyAfterUpdate.RefillAmount.Int64,
			Interval:  interval,
			RefillDay: day,
		}
	}

	h.KeyCache.Remove(ctx, key.Hash)
	if err := h.UsageLimiter.Invalidate(ctx, key.ID); err != nil {
		logger.Error("Failed to invalidate usage limit",
			"error", err.Error(),
			"key_id", key.ID,
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
