package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalExchangeSessionRequestBody
	Response = openapi.V2PortalExchangeSessionResponseBody
)

// Handler implements zen.Route for the portal session exchange endpoint.
// This endpoint is unauthenticated — it validates the session token itself.
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.exchangeSession" }

// Handle exchanges a short-lived session token for a long-lived browser session.
// The exchange is atomic: find the valid token, mark it as exchanged, and create
// the browser session all within a single transaction. Concurrent exchanges race
// on the UPDATE and only one succeeds.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if req.SessionId == "" {
		return fault.New("sessionId is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("missing sessionId"),
			fault.Public("sessionId is required."),
		)
	}

	nowMs := time.Now().UnixMilli()

	type exchangeResult struct {
		browserSessionID string
		expiresAt        int64
		workspaceID      string
		externalID       string
		portalConfigID   string
	}

	result, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (exchangeResult, error) {
		var zero exchangeResult

		// Find the valid, unexpired, unexchanged token.
		sessionToken, txErr := db.Query.FindValidPortalSessionToken(ctx, tx, db.FindValidPortalSessionTokenParams{
			ID:  req.SessionId,
			Now: nowMs,
		})
		if txErr != nil {
			if db.IsNotFound(txErr) {
				return zero, fault.New("invalid or expired session",
					fault.Code(codes.Portal.Session.SessionNotFound.URN()),
					fault.Internal("session token not found, already exchanged, or expired"),
					fault.Public("Session is invalid, expired, or has already been used."),
				)
			}
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error finding session token"),
				fault.Public("Failed to exchange session."),
			)
		}

		// Atomically mark as exchanged. The WHERE clause ensures only unexchanged
		// tokens match, so concurrent exchanges race on the same row.
		exchangeRes, txErr := db.Query.ExchangePortalSessionToken(ctx, tx, db.ExchangePortalSessionTokenParams{
			ExchangedAt: sql.NullInt64{Int64: nowMs, Valid: true},
			ID:          req.SessionId,
			Now:         nowMs,
		})
		if txErr != nil {
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error exchanging session token"),
				fault.Public("Failed to exchange session."),
			)
		}

		rowsAffected, txErr := exchangeRes.RowsAffected()
		if txErr != nil {
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("failed to check rows affected"),
				fault.Public("An internal error occurred."),
			)
		}

		if rowsAffected == 0 {
			return zero, fault.New("session not found",
				fault.Code(codes.Portal.Session.SessionNotFound.URN()),
				fault.Internal("concurrent exchange: token already used"),
				fault.Public("Session is invalid, expired, or has already been used."),
			)
		}

		// Create the browser session with 24-hour expiry.
		browserSessionID := string(uid.PortalSessionPrefix) + "_" + uid.Secure()
		browserExpiresAt := nowMs + int64(24*time.Hour/time.Millisecond)

		txErr = db.Query.InsertPortalSession(ctx, tx, db.InsertPortalSessionParams{
			ID:             browserSessionID,
			WorkspaceID:    sessionToken.WorkspaceID,
			PortalConfigID: sessionToken.PortalConfigID,
			ExternalID:     sessionToken.ExternalID,
			Permissions:    sessionToken.Permissions,
			Preview:        sessionToken.Preview,
			ExpiresAt:      browserExpiresAt,
			CreatedAt:      nowMs,
		})
		if txErr != nil {
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert browser session"),
				fault.Public("Failed to create session."),
			)
		}

		txErr = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				Event:       auditlog.PortalSessionExchangeEvent,
				WorkspaceID: sessionToken.WorkspaceID,
				ActorType:   auditlog.SystemActor,
				ActorID:     req.SessionId,
				ActorName:   "portal session token",
				ActorMeta:   map[string]any{},
				Display:     fmt.Sprintf("Exchanged portal session for %s", sessionToken.ExternalID),
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          browserSessionID,
						DisplayName: sessionToken.ExternalID,
						Name:        sessionToken.ExternalID,
						Meta:        map[string]any{"portalConfigId": sessionToken.PortalConfigID},
						Type:        auditlog.PortalSessionResourceType,
					},
				},
			},
		})
		if txErr != nil {
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert audit log"),
				fault.Public("Failed to create session."),
			)
		}

		return exchangeResult{
			browserSessionID: browserSessionID,
			expiresAt:        browserExpiresAt,
			workspaceID:      sessionToken.WorkspaceID,
			externalID:       sessionToken.ExternalID,
			portalConfigID:   sessionToken.PortalConfigID,
		}, nil
	})
	if err != nil {
		return err
	}

	s.ResponseWriter().Header().Set("Cache-Control", "no-store")
	s.ResponseWriter().Header().Set("Pragma", "no-cache")

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2PortalExchangeSessionResponseData{
			Token:     result.browserSessionID,
			ExpiresAt: result.expiresAt,
		},
	})
}
