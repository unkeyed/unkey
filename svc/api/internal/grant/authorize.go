package grant

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
)

// StateBytesMax is the longest OAuth state value the server stores.
const StateBytesMax = 1024

// AuthorizeRequest is a verified OAuth authorization request. The caller
// loaded Client, matched RedirectURI against it, and parsed Challenge. Scope
// and resource were checked against the server's constants and are not
// stored: there is one of each.
type AuthorizeRequest struct {
	Client      Client
	RedirectURI RedirectURI
	State       string
	Challenge   Challenge
	RemoteIP    string
	UserAgent   string
}

// Authorized is where the browser goes next: the dashboard consent page with
// one opaque parameter that survives the dashboard's sign-in redirect.
type Authorized struct {
	ConsentURL string
}

// Authorize records a pending OAuth request and returns the consent URL. The
// authorization code does not exist yet; approval mints it.
func (s *Service) Authorize(ctx context.Context, req AuthorizeRequest) (Authorized, error) {
	if err := s.requireOAuth(); err != nil {
		return Authorized{}, err
	}
	if err := assert.All(
		assert.NotEmpty(string(req.Client.ID), "authorize needs a registered client"),
		assert.NotEmpty(req.RedirectURI.raw, "authorize needs a verified redirect uri"),
		assert.NotEmpty(req.Challenge.value, "authorize needs a pkce challenge"),
		assert.LessOrEqual(len(req.State), StateBytesMax, "state exceeds the stored length"),
	); err != nil {
		return Authorized{}, err
	}
	if err := s.limitAddress(ctx, "authorize", req.RemoteIP, authorizeLimitPerMinute, time.Minute, "Too many sign-in requests from this address. Try again in a minute."); err != nil {
		return Authorized{}, err
	}
	s.deleteExpired(ctx)

	userAgent := truncate(strings.TrimSpace(req.UserAgent), 512)
	now := s.Clock.Now().UnixMilli()
	var lastErr error
	for range 5 {
		handle, err := randomUserCode(consentHandleLength)
		if err != nil {
			return Authorized{}, err
		}
		consentURL, err := dashboardVerificationURL(strings.TrimRight(s.DashboardBaseURL, "/"), consentPath, handle)
		if err != nil {
			return Authorized{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("dashboard url is invalid"),
				fault.Public("OAuth is not configured on this API."),
			)
		}
		err = db.Query.InsertRedirectAuthorization(ctx, s.DB.RW(), db.InsertRedirectAuthorizationParams{
			ID:        string(uid.DeviceAuthorizationPrefix) + "_" + uid.Secure(24),
			UserCode:  handle,
			ExpiresAt: now + consentTTL.Milliseconds(),
			DeviceName: sql.NullString{
				String: req.Client.Name,
				Valid:  true,
			},
			RequesterIp: sql.NullString{
				String: req.RemoteIP,
				Valid:  req.RemoteIP != "",
			},
			RequesterUserAgent: sql.NullString{
				String: userAgent,
				Valid:  userAgent != "",
			},
			ClientID: sql.NullString{
				String: string(req.Client.ID),
				Valid:  true,
			},
			RedirectUri: sql.NullString{
				String: req.RedirectURI.raw,
				Valid:  true,
			},
			RedirectUriPresented: sql.NullBool{
				Bool:  req.RedirectURI.presented,
				Valid: true,
			},
			State: sql.NullString{
				String: req.State,
				Valid:  req.State != "",
			},
			CodeChallenge: sql.NullString{
				String: req.Challenge.value,
				Valid:  true,
			},
			CreatedAt: now,
		})
		if db.IsDuplicateKeyError(err) {
			lastErr = err
			continue
		}
		if err != nil {
			return Authorized{}, err
		}
		return Authorized{ConsentURL: consentURL}, nil
	}
	return Authorized{}, fault.Wrap(lastErr,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("could not allocate an oauth consent handle"),
		fault.Public("Could not start sign-in. Try again."),
	)
}
