package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

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
	KeyCache     cache.Cache[string, db.FindKeyForVerificationRow]
	UsageLimiter usagelimiter.Service
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route match des
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

	if (req.Operation == openapi.Decrement || req.Operation == openapi.Increment) && !key.RemainingRequests.Valid {
		return fault.New("wrong operation usage",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("You cannot increment or decrement a key with unlimited credits."),
		)
	}

	credits := sql.NullInt32{Int32: 0, Valid: false}

	// The only errors that can be returned here are isNull or notSpecified
	// which firstly is wanted and secondly doesn't matter
	reqVal, _ := req.Value.Get()

	// Value has been set as not null
	if !req.Value.IsNull() && req.Value.IsSpecified() {
		credits = sql.NullInt32{Int32: int32(reqVal), Valid: true} // nolint:gosec
	}

	key, err = db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.FindLiveKeyByIDRow, error) {
		switch req.Operation {
		case openapi.Set:
			err = db.Query.UpdateKeyCreditsSet(ctx, tx, db.UpdateKeyCreditsSetParams{
				ID:      key.ID,
				Credits: credits,
			})
		case openapi.Increment:
			err = db.Query.UpdateKeyCreditsIncrement(ctx, tx, db.UpdateKeyCreditsIncrementParams{
				ID:      key.ID,
				Credits: credits,
			})
		case openapi.Decrement:
			err = db.Query.UpdateKeyCreditsDecrement(ctx, tx, db.UpdateKeyCreditsDecrementParams{
				ID:      key.ID,
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
				ID:           key.ID,
				RefillAmount: sql.NullInt32{Int32: 0, Valid: false},
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
		if keyAfterUpdate.RemainingRequests.Valid {
			remaining = fmt.Sprintf("%d", keyAfterUpdate.RemainingRequests.Int32)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.KeyUpdateEvent,
				Display:     fmt.Sprintf("Updated Key %s, set remaining to %s.", key.ID, remaining),
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				ActorType:   auditlog.RootKeyActor,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          key.KeyAuthID,
						Type:        auditlog.KeyAuthResourceType,
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

		return keyAfterUpdate, err
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

	if key.RemainingRequests.Valid {
		responseData.Remaining = nullable.NewNullableWithValue(int64(key.RemainingRequests.Int32))
	}

	if key.RefillAmount.Valid {
		var day *int
		interval := openapi.KeyCreditsRefillIntervalDaily

		if key.RefillDay.Valid {
			interval = openapi.KeyCreditsRefillIntervalMonthly
			day = ptr.P(int(key.RefillDay.Int16))
		}

		responseData.Refill = &openapi.KeyCreditsRefill{
			Amount:    int64(key.RefillAmount.Int32),
			Interval:  interval,
			RefillDay: day,
		}
	}

	h.KeyCache.Remove(ctx, key.Hash)
	h.UsageLimiter.Invalidate(ctx, key.ID)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
