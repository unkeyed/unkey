package clidevice

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	registeredClientPrefix      = "uoc_"
	registeredClientIDLengthMax = 2048
	registeredClientKeyInfo     = "unkey oauth client v1"
	clientNameBytesMax          = 128
	redirectURIBytesMax         = 512
	redirectURIsRequestedMax    = 8
	redirectURIsMax             = 4

	tokenEndpointAuthNone = "none"
)

// ClientRegistration is the RFC 7591 client metadata a dynamic registration
// request carries. Fields the server does not use are ignored.
type ClientRegistration struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// Registrar implements RFC 7591 dynamic client registration without storage.
// The client_id is the registration itself, signed:
//
//	uoc_<base64url(json{name, redirect_uris, iat})>.<base64url(HMAC-SHA256)>
//
// A table would add an unauthenticated write path and a cleanup job for
// clients that register on every connection, and nothing a client needs:
// registrations are not revocable, sessions are. Every key verifies and the
// first one signs, so rotating the CLI signing secret keeps existing client
// ids valid until the old secret is removed.
type Registrar struct {
	keys [][]byte
}

type registration struct {
	Name         string   `json:"n"`
	RedirectURIs []string `json:"r"`
	IssuedAt     int64    `json:"t"`
}

// NewRegistrar derives one MAC key from each signing secret, so a client id
// can never be confused with an access token signature.
func NewRegistrar(secrets [][]byte) (*Registrar, error) {
	if len(secrets) == 0 {
		return nil, errors.New("oauth client registration requires at least one signing secret")
	}
	keys := make([][]byte, 0, len(secrets))
	for _, secret := range secrets {
		key, err := hkdf.Key(sha256.New, secret, nil, registeredClientKeyInfo, sha256.Size)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return &Registrar{keys: keys}, nil
}

// Register validates the metadata and returns the client with its signed id.
// Redirect URIs that are neither https nor loopback http are dropped rather
// than rejected, because clients such as Cursor register a custom scheme next
// to a loopback URI they actually use. The returned client lists only the
// URIs that were kept, which RFC 7591 section 3.2.1 lets the server do.
func (r *Registrar) Register(req ClientRegistration, issuedAt time.Time) (OAuthClient, error) {
	if req.TokenEndpointAuthMethod != "" && req.TokenEndpointAuthMethod != tokenEndpointAuthNone {
		return OAuthClient{}, oauthError(OAuthInvalidClientMetadata, "Only public clients with token_endpoint_auth_method none are supported.")
	}
	for _, grant := range req.GrantTypes {
		if grant != grantAuthorizationCode && grant != grantRefreshToken {
			return OAuthClient{}, oauthError(OAuthInvalidClientMetadata, "grant_types may only contain authorization_code and refresh_token.")
		}
	}
	for _, responseType := range req.ResponseTypes {
		if responseType != "code" {
			return OAuthClient{}, oauthError(OAuthInvalidClientMetadata, "response_types may only contain code.")
		}
	}

	redirectURIs, err := registrableRedirectURIs(req.RedirectURIs)
	if err != nil {
		return OAuthClient{}, err
	}
	name, err := registrableClientName(req.ClientName, redirectURIs[0])
	if err != nil {
		return OAuthClient{}, err
	}

	payload, err := json.Marshal(registration{
		Name:         name,
		RedirectURIs: redirectURIs,
		IssuedAt:     issuedAt.Unix(),
	})
	if err != nil {
		return OAuthClient{}, err
	}
	unsigned := registeredClientPrefix + base64.RawURLEncoding.EncodeToString(payload)
	clientID := unsigned + "." + base64.RawURLEncoding.EncodeToString(sign(r.keys[0], unsigned))
	if len(clientID) > registeredClientIDLengthMax {
		return OAuthClient{}, oauthError(OAuthInvalidClientMetadata, "The client metadata is too large. Register a shorter client_name or fewer redirect_uris.")
	}
	return OAuthClient{
		ID:           clientID,
		Name:         name,
		RedirectURIs: redirectURIs,
	}, nil
}

// Lookup verifies a registered client id and returns the client it describes.
func (r *Registrar) Lookup(clientID string) (OAuthClient, error) {
	invalid := oauthError(OAuthInvalidRequest, "client_id is not a client this server registered.")
	if len(clientID) > registeredClientIDLengthMax {
		return OAuthClient{}, invalid
	}
	unsigned, encodedMAC, ok := strings.Cut(clientID, ".")
	if !ok || !strings.HasPrefix(unsigned, registeredClientPrefix) {
		return OAuthClient{}, invalid
	}
	mac, err := base64.RawURLEncoding.DecodeString(encodedMAC)
	if err != nil || !r.verify(unsigned, mac) {
		return OAuthClient{}, invalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(unsigned, registeredClientPrefix))
	if err != nil {
		return OAuthClient{}, invalid
	}
	var reg registration
	if err := json.Unmarshal(payload, &reg); err != nil || len(reg.RedirectURIs) == 0 {
		return OAuthClient{}, invalid
	}
	return OAuthClient{
		ID:           clientID,
		Name:         reg.Name,
		RedirectURIs: reg.RedirectURIs,
	}, nil
}

func (r *Registrar) verify(unsigned string, mac []byte) bool {
	for _, key := range r.keys {
		if hmac.Equal(mac, sign(key, unsigned)) {
			return true
		}
	}
	return false
}

func sign(key []byte, unsigned string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(unsigned))
	return h.Sum(nil)
}

func registrableRedirectURIs(requested []string) ([]string, error) {
	if len(requested) > redirectURIsRequestedMax {
		return nil, oauthError(OAuthInvalidRedirectURI, "Register at most 8 redirect_uris.")
	}
	kept := make([]string, 0, len(requested))
	for _, uri := range requested {
		if len(uri) > redirectURIBytesMax || checkRedirectURI(uri) != nil || slices.Contains(kept, uri) {
			continue
		}
		kept = append(kept, uri)
	}
	if len(kept) == 0 {
		return nil, oauthError(OAuthInvalidRedirectURI, "Register at least one redirect_uri that is an https URL, or http on a loopback address, of at most 512 bytes.")
	}
	if len(kept) > redirectURIsMax {
		return nil, oauthError(OAuthInvalidRedirectURI, "Register at most 4 https or loopback redirect_uris.")
	}
	return kept, nil
}

func registrableClientName(raw string, firstRedirectURI string) (string, error) {
	name := strings.TrimSpace(raw)
	unprintable := func(r rune) bool { return !unicode.IsPrint(r) }
	if len(name) > clientNameBytesMax || !utf8.ValidString(name) || strings.ContainsFunc(name, unprintable) {
		return "", oauthError(OAuthInvalidClientMetadata, "client_name must be at most 128 bytes of printable text.")
	}
	if name == "" {
		return hostOf(firstRedirectURI), nil
	}
	return name, nil
}
