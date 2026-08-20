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

// exchangeAlreadyClaimed reports whether the session behind exchangeCodeHash was
// already redeemed with accessTokenHash, meaning an earlier attempt of this same
// call committed before the caller saw a transient failure.
//
// A false result covers every genuine rejection: no such code, or a code
// redeemed by some other request. Both keep the caller's undifferentiated 401.
func exchangeAlreadyClaimed(ctx context.Context, tx db.DBTX, exchangeCodeHash, accessTokenHash string) (bool, error) {
	session, err := db.Query.FindPortalSessionByExchangeCodeHash(ctx, tx, exchangeCodeHash)
	if err != nil {
		if db.IsNotFound(err) {
			return false, nil
		}
		return false, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error checking whether portal code was already redeemed"),
			fault.Public("Failed to exchange session."),
		)
	}

	return session.AccessTokenHash.Valid && session.AccessTokenHash.String == accessTokenHash, nil
}

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

	// The access token is a bearer credential: crypto/rand via uid.Secure (never
	// uid.New, which is math/rand), returned once here, stored only as a hash.
	//
	// Minted once, outside the retry loop, so every attempt writes the same hash.
	// That is what makes a replayed attempt recognizable below.
	accessToken := string(uid.PortalAccessTokenPrefix) + "_" + uid.Secure()
	accessTokenHash := hash.Sha256(accessToken)
	accessTokenExpiresAt := nowMs + int64(accessTokenTTL/time.Millisecond)

	// TxRetry runs the closure again on a transient error. Counting attempts
	// keeps the replay lookup off the first pass, where no earlier attempt can
	// have committed, so an unauthenticated flood of bogus codes still costs one
	// query rather than two. TxRetry loops synchronously, so a plain int needs no
	// synchronization.
	attempt := 0

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		attempt++

		exchangeRes, txErr := db.Query.ExchangePortalSessionCode(ctx, tx, db.ExchangePortalSessionCodeParams{
			AccessTokenHash:      sql.NullString{String: accessTokenHash, Valid: true},
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
		//
		// It also covers one success: TxRetry re-runs this closure on a transient
		// connection error, which is exactly the class where the commit may in
		// fact have landed. A replayed attempt then finds access_token_hash
		// already set and would reject a token this very call had persisted,
		// burning the single-use code and locking the user out with no way to
		// retry. Distinguish that case by the hash: only this call knows the
		// plaintext it minted, so a stored hash equal to ours means our own
		// earlier attempt won.
		if rowsAffected == 0 {
			if attempt > 1 {
				claimedByThisCall, replayErr := exchangeAlreadyClaimed(ctx, tx, exchangeCodeHash, accessTokenHash)
				if replayErr != nil {
					return replayErr
				}
				if claimedByThisCall {
					// The committed attempt wrote the audit log in the same
					// transaction, so it is already durable. Do not write a second.
					return nil
				}
			}

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
