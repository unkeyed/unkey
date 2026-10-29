package grant

import (
	"context"
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// RedeemRequest is an RFC 6749 authorization_code token request from a public
// client. Values are passed as received; Redeem does the comparisons.
type RedeemRequest struct {
	Code         string
	ClientID     ClientID
	RedirectURI  string
	CodeVerifier string
	RemoteIP     string
	UserAgent    string
}

// Redeem exchanges an authorization code for a session held by the client.
// The code is single use, lives codeTTL from approval, and is bound to the
// client, the redirect URI, and the PKCE challenge of the authorization
// request.
//
// Every refusal wraps ErrRejected with the same message, so the response does
// not say which check failed. Redeeming a code that already issued a session
// revokes that session (RFC 6749 section 4.1.2): either the client is
// replaying or someone else redeemed it first.
func (s *Service) Redeem(ctx context.Context, req RedeemRequest) (Tokens, error) {
	if err := s.requireOAuth(); err != nil {
		return Tokens{}, err
	}
	return s.limitFailures(ctx, "redeem_failures", req.RemoteIP, failedRedeemLimitPerMinute, func() (Tokens, error) {
		return s.redeem(ctx, req)
	})
}

func (s *Service) redeem(ctx context.Context, req RedeemRequest) (Tokens, error) {
	if req.Code == "" {
		return Tokens{}, rejectedCode()
	}

	row, err := db.Query.FindDeviceAuthorizationByCodeHash(ctx, s.DB.RW(), sql.NullString{String: hashCode(req.Code), Valid: true})
	if err != nil {
		if db.IsNotFound(err) {
			return Tokens{}, rejectedCode()
		}
		return Tokens{}, err
	}
	parsed, err := parseRequest(row)
	if err != nil {
		return Tokens{}, err
	}
	d, ok := parsed.delivery.(redirectDelivery)
	if !ok {
		return Tokens{}, fault.New("a device login carries an authorization code hash",
			fault.Internal("device_authorizations row "+row.ID+" has a code hash but no client"),
		)
	}
	if !d.redeemable(req.ClientID, req.RedirectURI, req.CodeVerifier) {
		return Tokens{}, rejectedCode()
	}

	switch s.authorizationStatus(row) {
	case db.DeviceAuthorizationsStatusConsumed:
		return Tokens{}, s.rejectReplay(ctx, row.SessionID.String)
	case db.DeviceAuthorizationsStatusPending, db.DeviceAuthorizationsStatusDenied, db.DeviceAuthorizationsStatusExpired:
		return Tokens{}, rejectedCode()
	case db.DeviceAuthorizationsStatusApproved:
	}

	res, err := s.issue(ctx, parsed, req.RemoteIP, req.UserAgent)
	if errors.Is(err, ErrRejected) {
		return Tokens{}, rejectedCode()
	}
	if err != nil {
		return Tokens{}, err
	}
	switch res.outcome {
	case issueComplete:
		return res.tokens, nil
	case issueDenied:
		return Tokens{}, forbidden(res.detail)
	case issueChanged:
		return Tokens{}, rejectedCode()
	case issueReplayed:
		return Tokens{}, s.rejectReplay(ctx, res.sessionID)
	default:
		return Tokens{}, fault.New("unexpected issue outcome")
	}
}

// rejectReplay revokes the session a replayed code issued and returns the
// refusal. The revocation commits even though the request fails.
func (s *Service) rejectReplay(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return rejectedCode()
	}
	session, err := db.Query.FindCLISessionByID(ctx, s.DB.RW(), sessionID)
	if db.IsNotFound(err) {
		return rejectedCode()
	}
	if err != nil {
		return err
	}
	if err := s.endSession(ctx, session); err != nil {
		return err
	}
	return rejectedCode()
}
