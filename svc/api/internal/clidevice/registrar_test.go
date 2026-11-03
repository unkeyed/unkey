package clidevice

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var registeredAt = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func registrationFor(name string, redirectURIs ...string) ClientRegistration {
	return ClientRegistration{
		ClientName:              name,
		RedirectURIs:            redirectURIs,
		GrantTypes:              []string{grantAuthorizationCode, grantRefreshToken},
		ResponseTypes:           []string{"code"},
		TokenEndpointAuthMethod: "none",
	}
}

func TestRegisteredClientIDDescribesItself(t *testing.T) {
	registrar := newTestRegistrar(t)

	registered, err := registrar.Register(registrationFor("Claude Code", "http://localhost/callback", "https://claude.ai/api/mcp/auth_callback"), registeredAt)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(registered.ID, registeredClientPrefix))
	require.LessOrEqual(t, len(registered.ID), registeredClientIDLengthMax)

	found, err := registrar.Lookup(registered.ID)
	require.NoError(t, err)
	require.Equal(t, OAuthClient{
		ID:           registered.ID,
		Name:         "Claude Code",
		RedirectURIs: []string{"http://localhost/callback", "https://claude.ai/api/mcp/auth_callback"},
	}, found)
}

// TestRegisteredClientIDCannotBeForged guarantees that a client id is trusted
// only when this server signed it, so nobody can mint a registration that
// names an arbitrary redirect URI.
func TestRegisteredClientIDCannotBeForged(t *testing.T) {
	registrar := newTestRegistrar(t)
	registered, err := registrar.Register(registrationFor("Cursor", "http://localhost:8787/callback"), registeredAt)
	require.NoError(t, err)
	unsigned, mac, ok := strings.Cut(registered.ID, ".")
	require.True(t, ok)
	flipped, err := base64.RawURLEncoding.DecodeString(mac)
	require.NoError(t, err)
	flipped[0] ^= 0xff

	forgedPayload, err := json.Marshal(registration{
		Name:         "Cursor",
		RedirectURIs: []string{"https://attacker.example.com/callback"},
		IssuedAt:     registeredAt.Unix(),
	})
	require.NoError(t, err)
	foreign, err := newTestRegistrar(t, "another-signing-secret-32-bytes!!").Register(registrationFor("Cursor", "http://localhost:8787/callback"), registeredAt)
	require.NoError(t, err)

	for name, clientID := range map[string]string{
		"payload swapped under the original mac": registeredClientPrefix + base64.RawURLEncoding.EncodeToString(forgedPayload) + "." + mac,
		"mac flipped":                            unsigned + "." + base64.RawURLEncoding.EncodeToString(flipped),
		"mac missing":                            unsigned,
		"mac empty":                              unsigned + ".",
		"signed with another secret":             foreign.ID,
		"oversized":                              registered.ID + strings.Repeat("a", registeredClientIDLengthMax),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := registrar.Lookup(clientID)
			requireOAuthError(t, err, OAuthInvalidRequest)
		})
	}
}

// TestRegistrationsSurviveSigningSecretRotation guarantees that adding a new
// signing secret in front of the old one keeps every registered client
// working until the old secret is removed.
func TestRegistrationsSurviveSigningSecretRotation(t *testing.T) {
	const oldSecret = "old-signing-secret-32-bytes-long!"
	const newSecret = "new-signing-secret-32-bytes-long!"
	before, err := newTestRegistrar(t, oldSecret).Register(registrationFor("Cursor", "http://localhost:8787/callback"), registeredAt)
	require.NoError(t, err)

	rotating := newTestRegistrar(t, newSecret, oldSecret)
	_, err = rotating.Lookup(before.ID)
	require.NoError(t, err)
	after, err := rotating.Register(registrationFor("Cursor", "http://localhost:8787/callback"), registeredAt)
	require.NoError(t, err)

	rotated := newTestRegistrar(t, newSecret)
	_, err = rotated.Lookup(after.ID)
	require.NoError(t, err)
	_, err = rotated.Lookup(before.ID)
	requireOAuthError(t, err, OAuthInvalidRequest)
}

// TestRegistrationKeepsOnlyRedirectURIsTheDashboardMayOpen guarantees that a
// client registering a custom scheme next to a loopback URI, as Cursor does,
// still registers, and that only the safe URIs are ever trusted.
func TestRegistrationKeepsOnlyRedirectURIsTheDashboardMayOpen(t *testing.T) {
	registrar := newTestRegistrar(t)

	registered, err := registrar.Register(registrationFor("Cursor",
		"cursor://anysphere.cursor-mcp/oauth/callback",
		"https://cursor.com/agents/oauth/callback",
		"http://localhost:8787/callback",
		"http://localhost:8787/callback",
		"http://cursor.com/insecure",
		"javascript:alert(1)",
	), registeredAt)
	require.NoError(t, err)
	require.Equal(t, []string{"https://cursor.com/agents/oauth/callback", "http://localhost:8787/callback"}, registered.RedirectURIs)
}

func TestRegistrationNamesAnUnnamedClientAfterItsRedirectHost(t *testing.T) {
	registered, err := newTestRegistrar(t).Register(registrationFor("  ", "https://client.example.com/callback"), registeredAt)
	require.NoError(t, err)
	require.Equal(t, "client.example.com", registered.Name)
}

func TestRegistrationRejectsUnsupportedOrOversizedMetadata(t *testing.T) {
	registrar := newTestRegistrar(t)
	loopback := "http://localhost:8787/callback"
	longURI := "https://client.example.com/" + strings.Repeat("a", redirectURIBytesMax-len("https://client.example.com/"))

	tests := []struct {
		name string
		req  ClientRegistration
		code OAuthErrorCode
	}{
		{name: "confidential client", req: func() ClientRegistration {
			r := registrationFor("Cursor", loopback)
			r.TokenEndpointAuthMethod = "client_secret_basic"
			return r
		}(), code: OAuthInvalidClientMetadata},
		{name: "client credentials grant", req: func() ClientRegistration {
			r := registrationFor("Cursor", loopback)
			r.GrantTypes = []string{"client_credentials"}
			return r
		}(), code: OAuthInvalidClientMetadata},
		{name: "implicit response type", req: func() ClientRegistration {
			r := registrationFor("Cursor", loopback)
			r.ResponseTypes = []string{"token"}
			return r
		}(), code: OAuthInvalidClientMetadata},
		{name: "name too long", req: registrationFor(strings.Repeat("a", clientNameBytesMax+1), loopback), code: OAuthInvalidClientMetadata},
		{name: "name with control characters", req: registrationFor("Cursor‮", loopback), code: OAuthInvalidClientMetadata},
		{name: "no redirect uri", req: registrationFor("Cursor"), code: OAuthInvalidRedirectURI},
		{name: "only custom scheme redirect uris", req: registrationFor("Cursor", "cursor://anysphere.cursor-mcp/oauth/callback"), code: OAuthInvalidRedirectURI},
		{name: "redirect uri too long", req: registrationFor("Cursor", longURI+"a"), code: OAuthInvalidRedirectURI},
		{name: "too many requested redirect uris", req: registrationFor("Cursor",
			"http://localhost:1/a", "http://localhost:1/b", "http://localhost:1/c", "http://localhost:1/d",
			"http://localhost:1/e", "http://localhost:1/f", "http://localhost:1/g", "http://localhost:1/h", "http://localhost:1/i",
		), code: OAuthInvalidRedirectURI},
		{name: "too many usable redirect uris", req: registrationFor("Cursor",
			"http://localhost:1/a", "http://localhost:1/b", "http://localhost:1/c", "http://localhost:1/d", "http://localhost:1/e",
		), code: OAuthInvalidRedirectURI},
		{name: "client id over the column width", req: registrationFor(strings.Repeat("a", clientNameBytesMax),
			longURI[:len(longURI)-1]+"1", longURI[:len(longURI)-1]+"2", longURI[:len(longURI)-1]+"3", longURI[:len(longURI)-1]+"4",
		), code: OAuthInvalidClientMetadata},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := registrar.Register(tt.req, registeredAt)
			requireOAuthError(t, err, tt.code)
		})
	}
}
