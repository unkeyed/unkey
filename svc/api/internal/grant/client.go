package grant

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	clientIDLength       = 16
	clientNameRunesMax   = 128
	clientNameDefault    = "Unnamed app"
	redirectURIBytesMax  = 2048
	redirectURIsMax      = 5
	clientInsertAttempts = 3
	registerLimitPerHour = 20
	registerLimitWindow  = time.Hour
)

// ClientID identifies a registered OAuth client. It is public: it appears in
// every authorization URL.
type ClientID string

// Client is a registered public OAuth client (token_endpoint_auth_method
// "none"). Clients are immutable and permanent because a client never learns
// that its client_id stopped working.
type Client struct {
	ID           ClientID
	Name         string
	RedirectURIs []string
	// IssuedAt is unix seconds, the RFC 7591 client_id_issued_at.
	IssuedAt int64
}

// RedirectURI is a redirect URI that matched a client's registration. Only
// Client.Redirect and parseRequest build one, so a browser redirect to an
// address nobody registered cannot be expressed.
type RedirectURI struct {
	raw string
	// presented is whether the authorization request named the URI. When it
	// did not, the token request must not need it (RFC 6749 section 4.1.3).
	presented bool
}

// Redirect matches raw against the registration byte for byte, except that
// an http loopback registration matches any port (RFC 8252 section 7.3). An
// empty raw selects the registered URI when there is exactly one.
func (c Client) Redirect(raw string) (RedirectURI, bool) {
	if raw == "" {
		if len(c.RedirectURIs) == 1 {
			return RedirectURI{raw: c.RedirectURIs[0], presented: false}, true
		}
		return RedirectURI{raw: "", presented: false}, false
	}
	for _, registered := range c.RedirectURIs {
		if raw == registered || loopbackMatch(registered, raw) {
			return RedirectURI{raw: raw, presented: true}, true
		}
	}
	return RedirectURI{raw: "", presented: false}, false
}

func (r RedirectURI) String() string { return r.raw }

// WithError is the authorization error response at this redirect URI: its
// query plus error, error_description when set, state when set, and iss
// (RFC 9207).
func (r RedirectURI) WithError(code string, description string, state string, issuer string) (string, error) {
	params := url.Values{"error": {code}}
	if description != "" {
		params.Set("error_description", description)
	}
	return r.with(params, state, issuer)
}

func (r RedirectURI) withCode(code string, state string, issuer string) (string, error) {
	return r.with(url.Values{"code": {code}}, state, issuer)
}

func (r RedirectURI) with(params url.Values, state string, issuer string) (string, error) {
	u, err := url.Parse(r.raw)
	if err != nil {
		return "", fault.Wrap(err, fault.Internal("registered redirect uri does not parse"))
	}
	query := u.Query()
	for name, values := range params {
		query[name] = values
	}
	if state != "" {
		query.Set("state", state)
	}
	query.Set("iss", issuer)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

// RegisterClientRequest is an RFC 7591 registration whose protocol fields
// (auth method, grant and response types) the handler already checked.
type RegisterClientRequest struct {
	Name         string
	RedirectURIs []string
	RemoteIP     string
	UserAgent    string
}

// RegisterClient stores a public client and returns it. Registering the same
// name and redirect URIs again returns the existing client, so retries do not
// add rows to a table that is never cleaned up.
func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) (Client, error) {
	if err := s.requireOAuth(); err != nil {
		return Client{}, err
	}
	if err := s.limitAddress(ctx, "register", req.RemoteIP, registerLimitPerHour, registerLimitWindow, "Too many app registrations from this address. Try again later."); err != nil {
		return Client{}, err
	}

	name, err := cleanClientName(req.Name)
	if err != nil {
		return Client{}, err
	}
	uris, err := s.cleanRedirectURIs(req.RedirectURIs)
	if err != nil {
		return Client{}, err
	}
	urisJSON, err := json.Marshal(uris)
	if err != nil {
		return Client{}, err
	}
	metadata, err := json.Marshal(struct {
		Name         string   `json:"name"`
		RedirectURIs []string `json:"redirect_uris"`
	}{Name: name, RedirectURIs: uris})
	if err != nil {
		return Client{}, err
	}
	metadataHash := hash.Sha256(string(metadata))
	userAgent := truncate(strings.TrimSpace(req.UserAgent), 512)

	var lastErr error
	for range clientInsertAttempts {
		params := db.InsertOAuthClientParams{
			ID:           uid.New(uid.OAuthClientPrefix, clientIDLength),
			Name:         name,
			RedirectUris: urisJSON,
			MetadataHash: metadataHash,
			RegistrarIp: sql.NullString{
				String: req.RemoteIP,
				Valid:  req.RemoteIP != "",
			},
			RegistrarUserAgent: sql.NullString{
				String: userAgent,
				Valid:  userAgent != "",
			},
			CreatedAt: s.Clock.Now().UnixMilli(),
		}
		err := db.Query.InsertOAuthClient(ctx, s.DB.RW(), params)
		if err == nil {
			return Client{
				ID:           ClientID(params.ID),
				Name:         name,
				RedirectURIs: uris,
				IssuedAt:     params.CreatedAt / 1000,
			}, nil
		}
		if !db.IsDuplicateKeyError(err) {
			return Client{}, err
		}
		row, findErr := db.Query.FindOAuthClientByMetadataHash(ctx, s.DB.RW(), metadataHash)
		if findErr == nil {
			return clientFromRow(row)
		}
		if !db.IsNotFound(findErr) {
			return Client{}, findErr
		}
		lastErr = err
	}
	return Client{}, fault.Wrap(lastErr,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("could not allocate an oauth client id"),
		fault.Public("Could not register this app. Try again."),
	)
}

// FindClient loads a registered client, or returns ErrUnknownClient.
func (s *Service) FindClient(ctx context.Context, id string) (Client, error) {
	if err := s.requireOAuth(); err != nil {
		return Client{}, err
	}
	if !strings.HasPrefix(id, string(uid.OAuthClientPrefix)+"_") {
		return Client{}, ErrUnknownClient
	}
	row, err := db.Query.FindOAuthClientByID(ctx, s.DB.RO(), id)
	if db.IsNotFound(err) {
		// A client registered a moment ago may not have reached the replica.
		row, err = db.Query.FindOAuthClientByID(ctx, s.DB.RW(), id)
	}
	if db.IsNotFound(err) {
		return Client{}, ErrUnknownClient
	}
	if err != nil {
		return Client{}, err
	}
	return clientFromRow(row)
}

func clientFromRow(row db.OauthClient) (Client, error) {
	var uris []string
	if err := json.Unmarshal(row.RedirectUris, &uris); err != nil {
		return Client{}, fault.Wrap(err, fault.Internal("oauth client redirect uris are not a json string array"))
	}
	return Client{
		ID:           ClientID(row.ID),
		Name:         row.Name,
		RedirectURIs: uris,
		IssuedAt:     row.CreatedAt / 1000,
	}, nil
}

func cleanClientName(raw string) (string, error) {
	for _, r := range raw {
		if unicode.IsControl(r) {
			return "", invalidClientMetadata("client_name must not contain control characters.")
		}
	}
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" {
		return clientNameDefault, nil
	}
	if utf8.RuneCountInString(name) > clientNameRunesMax {
		return "", invalidClientMetadata("client_name must be at most 128 characters.")
	}
	return name, nil
}

func (s *Service) cleanRedirectURIs(raw []string) ([]string, error) {
	if len(raw) == 0 || len(raw) > redirectURIsMax {
		return nil, invalidRedirectURI("Register between 1 and 5 redirect URIs.")
	}
	for _, uri := range raw {
		if err := validateRedirectURI(uri, s.RedirectOrigins); err != nil {
			return nil, err
		}
	}
	uris := slices.Clone(raw)
	slices.Sort(uris)
	return slices.Compact(uris), nil
}

// validateRedirectURI accepts an absolute https URI, or http on the 127.0.0.1
// and [::1] loopback literals (RFC 8252 section 7.3), with no fragment or
// userinfo. "localhost" is rejected (RFC 8252 section 8.3) and so are custom
// schemes: the intended client is a web gateway. When origins is non-empty the
// URI's origin must be one of them.
func validateRedirectURI(raw string, origins []string) error {
	if len(raw) > redirectURIBytesMax {
		return invalidRedirectURI("Redirect URIs must be at most 2048 bytes.")
	}
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Host == "" || u.Opaque != "" {
		return invalidRedirectURI("Redirect URIs must be absolute URLs.")
	}
	if u.Fragment != "" || strings.Contains(raw, "#") {
		return invalidRedirectURI("Redirect URIs must not contain a fragment.")
	}
	if u.User != nil {
		return invalidRedirectURI("Redirect URIs must not contain user information.")
	}
	switch {
	case u.Scheme == "https":
	case u.Scheme == "http" && loopbackHost(u.Hostname()):
	default:
		return invalidRedirectURI("Redirect URIs must use https, or http on 127.0.0.1 or [::1].")
	}
	if len(origins) > 0 && !slices.ContainsFunc(origins, func(origin string) bool {
		return strings.EqualFold(strings.TrimRight(origin, "/"), u.Scheme+"://"+u.Host)
	}) {
		return invalidRedirectURI("This redirect URI's origin is not allowed to register.")
	}
	return nil
}

func loopbackHost(host string) bool {
	return host == "127.0.0.1" || host == "::1"
}

func loopbackMatch(registered string, presented string) bool {
	want, err := url.Parse(registered)
	if err != nil || want.Scheme != "http" || !loopbackHost(want.Hostname()) {
		return false
	}
	got, err := url.Parse(presented)
	if err != nil || got.User != nil || got.Fragment != "" || strings.Contains(presented, "#") {
		return false
	}
	return got.Scheme == want.Scheme &&
		got.Hostname() == want.Hostname() &&
		got.EscapedPath() == want.EscapedPath() &&
		got.RawQuery == want.RawQuery
}

func invalidRedirectURI(message string) error {
	return fault.Wrap(ErrInvalidRedirectURI,
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(message),
		fault.Public(message),
	)
}

func invalidClientMetadata(message string) error {
	return fault.Wrap(ErrInvalidClientMetadata,
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal(message),
		fault.Public(message),
	)
}
