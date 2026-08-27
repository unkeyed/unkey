package paseto

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestTokenParsing_RejectsMalformedTokens protects the three-or-four segment
// message format, exact v4.local header, unpadded Base64url rule, and minimum
// local body layout from malformed or ambiguous encodings.
func TestTokenParsing_RejectsMalformedTokens(t *testing.T) {
	canonicalBody := base64.RawURLEncoding.EncodeToString(make([]byte, tokenOverhead))
	nonCanonicalBody := canonicalBody[:len(canonicalBody)-1] + "B"
	tests := []struct {
		name  string
		token string
	}{
		{name: "empty", token: ""},
		{name: "header only", token: localHeader},
		{name: "wrong version", token: "v3.local." + canonicalBody},
		{name: "wrong purpose", token: "v4.public." + canonicalBody},
		{name: "uppercase header", token: "V4.local." + canonicalBody},
		{name: "missing purpose", token: "v4.." + canonicalBody},
		{name: "invalid payload alphabet", token: localHeader + "***"},
		{name: "padded payload", token: localHeader + canonicalBody + "="},
		{name: "non-canonical trailing bits", token: localHeader + nonCanonicalBody},
		{name: "line feed in payload", token: localHeader + canonicalBody[:4] + "\n" + canonicalBody[4:]},
		{name: "carriage return in payload", token: localHeader + canonicalBody[:4] + "\r" + canonicalBody[4:]},
		{name: "short payload", token: localHeader + base64.RawURLEncoding.EncodeToString(make([]byte, tokenOverhead-1))},
		{name: "invalid footer alphabet", token: localHeader + canonicalBody + ".***"},
		{name: "padded footer", token: localHeader + canonicalBody + ".YQ=="},
		{name: "line feed in footer", token: localHeader + canonicalBody + ".Y\nQ"},
		{name: "carriage return in footer", token: localHeader + canonicalBody + ".Y\rQ"},
		{name: "extra segment", token: localHeader + canonicalBody + ".YQ.extra"},
		{name: "leading whitespace", token: " " + localHeader + canonicalBody},
		{name: "trailing whitespace", token: localHeader + canonicalBody + " "},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseToken(test.token, localHeader)
			require.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

// TestTokenParsing_AcceptsEmptyFooterSegment guarantees decoders accept the
// four-segment form when the optional footer is empty. RFC section 2 says
// encoders should omit this segment, but it does not require decoders to do so.
func TestTokenParsing_AcceptsEmptyFooterSegment(t *testing.T) {
	body := base64.RawURLEncoding.EncodeToString(make([]byte, tokenOverhead))
	parsed, err := parseToken(localHeader+body+".", localHeader)
	require.NoError(t, err)
	require.Empty(t, parsed.footer)
}

// TestUnverifiedFooter_SeparatesKeySelectionFromAuthentication guarantees a
// caller can read an arbitrary footer before key selection, but a changed
// footer still fails token authentication.
func TestUnverifiedFooter_SeparatesKeySelectionFromAuthentication(t *testing.T) {
	local := newTestLocal[testClaims](t, make([]byte, localKeySize))
	token, err := local.Encrypt(testMessage())
	require.NoError(t, err)

	footer, err := UnverifiedFooter(token)
	require.NoError(t, err)
	require.Equal(t, testMessage().Footer, footer)

	changedToken := mutateTokenFooter(t, token)
	changedFooter, err := UnverifiedFooter(changedToken)
	require.NoError(t, err)
	require.NotEqual(t, footer, changedFooter)
	_, err = local.Decrypt(changedToken)
	require.ErrorIs(t, err, ErrInvalidToken)
}

func TestUnverifiedFooter_NoFooter(t *testing.T) {
	local := newTestLocal[testClaims](t, make([]byte, localKeySize))
	message := testMessage()
	message.Footer = nil
	token, err := local.Encrypt(message)
	require.NoError(t, err)

	footer, err := UnverifiedFooter(token)
	require.NoError(t, err)
	require.Empty(t, footer)
	require.Equal(t, 2, strings.Count(token, "."))
}

func TestUnverifiedFooter_PublicToken(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	token := signToken(privateKey, []byte(`{}`), []byte("footer"), nil)

	footer, err := UnverifiedFooter(token)
	require.NoError(t, err)
	require.Equal(t, []byte("footer"), footer)
}

func TestUnverifiedFooter_RejectsMalformedToken(t *testing.T) {
	_, err := UnverifiedFooter(localHeader + "***")
	require.ErrorIs(t, err, ErrInvalidToken)
}
