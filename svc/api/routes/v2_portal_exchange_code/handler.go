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
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PortalExchangeCodeRequestBody
	Response = openapi.V2PortalExchangeCodeResponseBody
)

// exchangeResult holds the values produced by the atomic exchange transaction.
type exchangeResult struct {
	sessionID   string
	accessToken string
	expiresAt   int64
}

// Handler implements zen.Route for the portal code exchange endpoint.
// This endpoint is unauthenticated — it validates the exchange code itself.
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.exchangeCode" }

// Handle consumes a short-lived exchange code and returns a portal access token.
// The exchange atomically consumes the code and activates the existing session.
// Concurrent exchanges race on the UPDATE and only one succeeds.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if req.Code == "" {
		return fault.New("code is required",
			fault.Code(codes.UnkeyPortalErrorsSessionTokenMissing),
			fault.Internal("missing code"),
			fault.Public("code is required."),
		)
	}

	nowMs := time.Now().UnixMilli()
	exchangeCodeHash := hash.Sha256(req.Code)
	var mintedResult exchangeResult
	var accessTokenHash string

	result, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (exchangeResult, error) {
		var zero exchangeResult

		// Find the valid, unexpired, unexchanged exchange code.
		portalSession, txErr := db.Query.FindValidPortalSessionByExchangeCode(ctx, tx, db.FindValidPortalSessionByExchangeCodeParams{
			ExchangeCodeHash: sql.NullString{String: exchangeCodeHash, Valid: true},
			Now:              nowMs,
		})
		if txErr != nil {
			if db.IsNotFound(txErr) {
				return zero, fault.New("invalid or expired session",
					fault.Code(codes.Portal.Session.SessionNotFound.URN()),
					fault.Internal("exchange code not found, already exchanged, or expired"),
					fault.Public("Session is invalid, expired, or has already been used."),
				)
			}
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error finding exchange code"),
				fault.Public("Failed to exchange session."),
			)
		}

		// Mint once per request and reuse the same credential if the transaction
		// helper retries after a transient database error.
		if mintedResult.accessToken == "" {
			mintedResult = exchangeResult{
				sessionID:   portalSession.ID,
				accessToken: uid.Secure(),
				expiresAt:   nowMs + int64(24*time.Hour/time.Millisecond),
			}
			accessTokenHash = hash.Sha256(mintedResult.accessToken)
		}

		// Atomically consume the exchange code and activate the session. The
		// WHERE clause ensures concurrent exchanges race on the same row.
		exchangeRes, txErr := db.Query.ConsumePortalSessionExchangeCode(ctx, tx, db.ConsumePortalSessionExchangeCodeParams{
			AccessTokenHash:      sql.NullString{String: accessTokenHash, Valid: true},
			AccessTokenCreatedAt: sql.NullInt64{Int64: nowMs, Valid: true},
			AccessTokenExpiresAt: sql.NullInt64{Int64: mintedResult.expiresAt, Valid: true},
			ID:                   portalSession.ID,
			ExchangeCodeHash:     sql.NullString{String: exchangeCodeHash, Valid: true},
			Now:                  nowMs,
		})
		if txErr != nil {
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error consuming exchange code"),
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
				fault.Internal("concurrent exchange: exchange code already used"),
				fault.Public("Session is invalid, expired, or has already been used."),
			)
		}

		txErr = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				Event:         auditlog.PortalSessionExchangeEvent,
				WorkspaceID:   portalSession.WorkspaceID,
				ActorType:     auditlog.SystemActor,
				ActorID:       portalSession.ID,
				ActorName:     "portal session",
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Exchanged portal session for %s", portalSession.ExternalID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          portalSession.ID,
						DisplayName: portalSession.ExternalID,
						Name:        portalSession.ExternalID,
						Meta:        map[string]any{"portalId": portalSession.PortalID},
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

		return mintedResult, nil
	})
	if err != nil {
		// A connection can fail while acknowledging COMMIT even when MySQL
		// committed it. If the exact minted hash is active, return the in-memory
		// credential rather than orphaning a committed access token.
		if mintedResult.accessToken == "" {
			return err
		}

		committedSession, findErr := db.Query.FindValidPortalSession(ctx, h.DB.RW(), db.FindValidPortalSessionParams{
			AccessTokenHash: sql.NullString{String: accessTokenHash, Valid: true},
			Now:             sql.NullInt64{Int64: nowMs, Valid: true},
		})
		if findErr != nil || committedSession.ID != mintedResult.sessionID {
			return err
		}
		result = mintedResult
	}

	s.ResponseWriter().Header().Set("Cache-Control", "no-store")
	s.ResponseWriter().Header().Set("Pragma", "no-cache")

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2PortalExchangeCodeResponseData{
			AccessToken: result.accessToken,
			ExpiresAt:   result.expiresAt,
		},
	})
}
