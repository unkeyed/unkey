package handler

import (
	"context"
	"database/sql"
	"fmt"
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
	keyData := db.ToKeyData(key)

	if keyData.Key.WorkspaceID != principal.WorkspaceID {
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
			ResourceID:   keyData.Api.ID,
			Action:       rbac.UpdateKey,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(keyData.KeyAuth.ProjectID).Keyspace(keyData.Key.KeyAuthID).Key(req.KeyId),
			permissions.Write,
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

	if (req.Operation == openapi.Decrement || req.Operation == openapi.Increment) && !keyData.Key.RemainingRequests.Valid {
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

	key, err = db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.FindLiveKeyByIDRow, error) {
		switch req.Operation {
		case openapi.Set:
			err = db.Query.UpdateKeyCreditsSet(ctx, tx, db.UpdateKeyCreditsSetParams{
				ID:      keyData.Key.ID,
				Credits: credits,
			})
		case openapi.Increment:
			err = db.Query.UpdateKeyCreditsIncrement(ctx, tx, db.UpdateKeyCreditsIncrementParams{
				ID:      keyData.Key.ID,
				Credits: credits,
			})
		case openapi.Decrement:
			err = db.Query.UpdateKeyCreditsDecrement(ctx, tx, db.UpdateKeyCreditsDecrementParams{
				ID:      keyData.Key.ID,
				Credits: credits,
			})
		default:
			return db.FindLiveKeyByIDRow{}, fault.New("invalid operation",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal(fmt.Sprintf("invalid operation: %s", req.Operation)),
				fault.Public("Invalid operation specified."),
			)
		}
		if err != nil {
			return db.FindLiveKeyByIDRow{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to update key credits."),
			)
		}

		// Reset the Refill data since it's not needed anymore
		if req.Value.IsNull() {
			err = db.Query.UpdateKeyCreditsRefill(ctx, tx, db.UpdateKeyCreditsRefillParams{
				ID:           keyData.Key.ID,
				RefillAmount: sql.NullInt64{Int64: 0, Valid: false},
				RefillDay:    sql.NullInt16{Int16: 0, Valid: false},
			})
			if err != nil {
				return db.FindLiveKeyByIDRow{}, fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to reset key refill data."),
				)
			}
		}

		keyAfterUpdate, keyErr := db.Query.FindLiveKeyByID(ctx, tx, req.KeyId)
		if keyErr != nil {
			if db.IsNotFound(keyErr) {
				return db.FindLiveKeyByIDRow{}, fault.Wrap(
					keyErr,
					fault.Code(codes.Data.Key.NotFound.URN()),
					fault.Internal("key got deleted after update"),
					fault.Public("We could not find the requested key."),
				)
			}

			return db.FindLiveKeyByIDRow{}, fault.Wrap(keyErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve key information."),
			)
		}

		remaining := "unlimited"
		if keyAfterUpdate.KeyRemainingRequests.Valid {
			remaining = fmt.Sprintf("%d", keyAfterUpdate.KeyRemainingRequests.Int64)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.KeyUpdateEvent,
				Display:       fmt.Sprintf("Updated Key %s, set remaining to %s.", keyData.Key.ID, remaining),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          keyData.Key.KeyAuthID,
						Type:        auditlog.KeySpaceResourceType,
						Name:        "",
						DisplayName: "",
						Meta:        nil,
					},
					{
						ID:          keyData.Key.ID,
						Type:        auditlog.KeyResourceType,
						Name:        keyData.Key.Name.String,
						DisplayName: keyData.Key.Name.String,
						Meta:        nil,
					},
				},
			},
		})

		return keyAfterUpdate, err
	})

	if err != nil {
		return err
	}
	keyData = db.ToKeyData(key)

	null := nullable.Nullable[int64]{}
	null.SetNull()

	responseData := openapi.KeyCreditsData{
		Refill:    nil,
		Remaining: null,
	}

	if keyData.Key.RemainingRequests.Valid {
		responseData.Remaining = nullable.NewNullableWithValue(int64(keyData.Key.RemainingRequests.Int64))
	}

	if keyData.Key.RefillAmount.Valid {
		var day int
		interval := openapi.KeyCreditsRefillIntervalDaily

		if keyData.Key.RefillDay.Valid {
			interval = openapi.KeyCreditsRefillIntervalMonthly
			day = int(keyData.Key.RefillDay.Int16)
		}

		responseData.Refill = &openapi.KeyCreditsRefill{
			Amount:    int64(keyData.Key.RefillAmount.Int64),
			Interval:  interval,
			RefillDay: day,
		}
	}

	h.KeyCache.Remove(ctx, keyData.Key.Hash)
	if err := h.UsageLimiter.Invalidate(ctx, keyData.Key.ID); err != nil {
		logger.Error("Failed to invalidate usage limit",
			"error", err.Error(),
			"key_id", keyData.Key.ID,
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
