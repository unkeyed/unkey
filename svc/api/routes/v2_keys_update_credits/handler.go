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

	auditPreparer, ok := h.Auditlogs.(auditlogs.OutboxPreparer)
	if !ok {
		return fault.New("audit service cannot prepare outbox rows", fault.Internal("update credits requires an outbox preparer"))
	}
	outboxRows, err := auditPreparer.PrepareOutboxRows(ctx, []auditlog.AuditLog{{
		WorkspaceID:   principal.WorkspaceID,
		Event:         auditlog.KeyUpdateEvent,
		Display:       fmt.Sprintf("Updated Key %s, set remaining to pending.", key.ID),
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
	}})
	if err != nil {
		return err
	}
	if len(outboxRows) != 1 {
		return fault.New("invalid audit outbox batch size", fault.Internal("update credits batch requires exactly one audit outbox row"))
	}
	outbox := db.InsertClickhouseOutboxForCreditUpdateParams{
		Version:     outboxRows[0].Version,
		WorkspaceID: outboxRows[0].WorkspaceID,
		EventID:     outboxRows[0].EventID,
		Payload:     outboxRows[0].Payload,
		KeyID:       key.ID,
		CreatedAt:   outboxRows[0].CreatedAt,
	}

	var batchResult db.KeyCreditsBatchResult
	switch req.Operation {
	case openapi.Set:
		batchResult, err = h.DB.BatchRW().UpdateKeyCreditsSetWithAuditBatch(ctx, db.UpdateKeyCreditsSetParams{
			ID:                key.ID,
			Credits:           credits,
			ClearRefillAmount: clearRefill,
			ClearRefillDay:    clearRefill,
		}, outbox)
	case openapi.Increment:
		batchResult, err = h.DB.BatchRW().UpdateKeyCreditsIncrementWithAuditBatch(ctx, db.UpdateKeyCreditsIncrementReturningParams{
			ID: key.ID, Credits: credits,
		}, outbox)
	case openapi.Decrement:
		batchResult, err = h.DB.BatchRW().UpdateKeyCreditsDecrementWithAuditBatch(ctx, db.UpdateKeyCreditsDecrementReturningParams{
			ID: key.ID, Credits: credits,
		}, outbox)
	default:
		return fault.New("invalid operation",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal(fmt.Sprintf("invalid operation: %s", req.Operation)),
			fault.Public("Invalid operation specified."),
		)
	}
	if err != nil {
		// A missing marker is an ambiguous commit. Invalidate both caches even
		// when the endpoint cannot confirm the mutation outcome.
		h.invalidate(ctx, key.Hash, key.ID)
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to update key credits."),
		)
	}
	if !batchResult.Applied {
		switch {
		case batchResult.Missing || batchResult.Deleted:
			return fault.New("key got deleted before credits update",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key got deleted before update"),
				fault.Public("We could not find the requested key."),
			)
		case batchResult.Unlimited:
			return fault.New("key credits became unlimited before update",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("You cannot increment or decrement a key with unlimited credits."),
			)
		case batchResult.Overflow:
			return fault.New("credits increment exceeds maximum",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("credits increment would exceed max int64"),
				fault.Public("The resulting credit balance exceeds the maximum supported value."),
			)
		default:
			return fault.New("credit update did not apply", fault.Internal("guarded credit update matched no row"))
		}
	}

	keyAfterUpdate := key
	keyAfterUpdate.RemainingRequests = batchResult.RemainingRequests
	if !batchResult.RemainingRequests.Valid {
		keyAfterUpdate.RefillAmount = sql.NullInt64{}
		keyAfterUpdate.RefillDay.Valid = false
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

	h.invalidate(ctx, key.Hash, key.ID)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}

func (h *Handler) invalidate(ctx context.Context, keyHash, keyID string) {
	h.KeyCache.Remove(ctx, keyHash)
	if err := h.UsageLimiter.Invalidate(ctx, keyID); err != nil {
		logger.Error("Failed to invalidate usage limit",
			"error", err.Error(),
			"key_id", keyID,
		)
	}
}
