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

// accessTokenTTL is how long an exchanged portal access token stays valid.
const accessTokenTTL = 24 * time.Hour

// Handler implements zen.Route for the portal code exchange endpoint.
// This endpoint is unauthenticated — it validates the exchange code itself.
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/portal.exchangeCode" }

// Handle exchanges a short-lived code for a long-lived access token.
//
// The code and the token are two credentials on one session row, so the
// exchange is a single conditional UPDATE rather than a copy between tables.
// Single use falls out of that predicate: concurrent redemptions race on one
// row and exactly one reports an affected row.
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

	// The access token is a bearer credential: returned once here, stored only
	// as a hash.
	accessToken := uid.New(uid.PortalSessionPrefix)
	accessTokenExpiresAt := nowMs + int64(accessTokenTTL/time.Millisecond)

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		exchangeRes, txErr := db.Query.ExchangePortalSessionCode(ctx, tx, db.ExchangePortalSessionCodeParams{
			AccessTokenHash:      sql.NullString{String: hash.Sha256(accessToken), Valid: true},
			AccessTokenCreatedAt: sql.NullInt64{Int64: nowMs, Valid: true},
			AccessTokenExpiresAt: sql.NullInt64{Int64: accessTokenExpiresAt, Valid: true},
			ExchangeCodeHash:     exchangeCodeHash,
			Now:                  nowMs,
		})
		if txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error exchanging portal code"),
				fault.Public("Failed to exchange session."),
			)
		}

		rowsAffected, txErr := exchangeRes.RowsAffected()
		if txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("failed to check rows affected"),
				fault.Public("An internal error occurred."),
			)
		}

		// Zero rows covers every rejection the predicate encodes — unknown code,
		// already redeemed, or expired — and they are deliberately not
		// distinguished to the caller.
		if rowsAffected == 0 {
			return fault.New("invalid or expired code",
				fault.Code(codes.Portal.Session.SessionNotFound.URN()),
				fault.Internal("exchange code not found, already redeemed, or expired"),
				fault.Public("Session is invalid, expired, or has already been used."),
			)
		}

		// Safe to read back unconditionally: the hash is UNIQUE, so this is the
		// row just claimed, and the caller established it won the race.
		session, txErr := db.Query.FindPortalSessionByExchangeCodeHash(ctx, tx, exchangeCodeHash)
		if txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error reading exchanged portal session"),
				fault.Public("Failed to exchange session."),
			)
		}

		txErr = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				Event:       auditlog.PortalSessionExchangeEvent,
				WorkspaceID: session.WorkspaceID,
				ActorType:   auditlog.SystemActor,
				// The session's row handle, never the exchange code: the actor
				// field is stored and surfaced, and the code is a credential.
				ActorID:       session.ID,
				ActorName:     "portal session",
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Exchanged portal session for %s", session.ExternalID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						ID:          session.ID,
						DisplayName: session.ExternalID,
						Name:        session.ExternalID,
						Meta:        map[string]any{"portalId": session.PortalID},
						Type:        auditlog.PortalSessionResourceType,
					},
				},
			},
		})
		if txErr != nil {
			return fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("failed to insert audit log"),
				fault.Public("Failed to create session."),
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.ResponseWriter().Header().Set("Cache-Control", "no-store")
	s.ResponseWriter().Header().Set("Pragma", "no-cache")

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2PortalExchangeCodeResponseData{
			AccessToken: accessToken,
			ExpiresAt:   accessTokenExpiresAt,
		},
	})
}
