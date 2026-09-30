package paseto

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type caseSensitiveClaims struct {
	Claims

	UpperIssuer string `json:"ISS,omitempty"`
}

type nonObjectMarshalerClaims struct {
	Claims
}

func (nonObjectMarshalerClaims) MarshalJSON() ([]byte, error) {
	return []byte(`[]`), nil
}

// TestPayloadDecoding_RejectsInvalidAuthenticatedJSON guarantees a valid
// Ed25519 signature cannot make a payload valid when it violates the payload
// processing rules or the registered-claim type table.
func TestPayloadDecoding_RejectsInvalidAuthenticatedJSON(t *testing.T) {
	privateKey, verifier := newPayloadVerifier[testClaims](t)

	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "empty input", payload: []byte{}},
		{name: "null", payload: []byte(`null`)},
		{name: "array", payload: []byte(`[]`)},
		{name: "string", payload: []byte(`"payload"`)},
		{name: "trailing value", payload: []byte(`{} {}`)},
		{name: "invalid UTF-8", payload: []byte{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}},
		{name: "duplicate top-level key", payload: []byte(`{"role":"reader","role":"admin"}`)},
		{name: "duplicate nested key", payload: []byte(`{"nested":{"key":1,"key":2}}`)},
		{name: "issuer number", payload: []byte(`{"iss":1}`)},
		{name: "issuer null", payload: []byte(`{"iss":null}`)},
		{name: "subject object", payload: []byte(`{"sub":{}}`)},
		{name: "audience array", payload: []byte(`{"aud":["dashboard"]}`)},
		{name: "token ID boolean", payload: []byte(`{"jti":true}`)},
		{name: "expiration number", payload: []byte(`{"exp":1}`)},
		{name: "expiration null", payload: []byte(`{"exp":null}`)},
		{name: "expiration empty", payload: []byte(`{"exp":""}`)},
		{name: "expiration absent zero value", payload: []byte(`{"exp":"0001-01-01T00:00:00Z"}`)},
		{name: "expiration without offset", payload: []byte(`{"exp":"2030-01-02T03:04:05"}`)},
		{name: "expiration comma fraction", payload: []byte(`{"exp":"2030-01-02T03:04:05,1Z"}`)},
		{name: "expiration one-digit hour", payload: []byte(`{"exp":"2030-01-02T3:04:05Z"}`)},
		{name: "expiration invalid calendar date", payload: []byte(`{"exp":"2030-02-30T03:04:05Z"}`)},
		{name: "expiration invalid leap second", payload: []byte(`{"exp":"2030-01-02T03:04:60Z"}`)},
		{name: "expiration offset hour 24", payload: []byte(`{"exp":"2030-01-02T03:04:05+24:00"}`)},
		{name: "expiration offset minute 60", payload: []byte(`{"exp":"2030-01-02T03:04:05+00:60"}`)},
		{name: "not-before lowercase separator", payload: []byte(`{"nbf":"2030-01-02t03:04:05Z"}`)},
		{name: "issued-at lowercase UTC", payload: []byte(`{"iat":"2030-01-02T03:04:05z"}`)},
		{name: "custom claim has wrong type", payload: []byte(`{"role":1}`)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token := signToken(privateKey, test.payload, nil, nil)
			_, err := verifier.Verify(token)
			require.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

func TestPayloadEncoding_RejectsMarshaledNonObject(t *testing.T) {
	_, err := encodePayload(nonObjectMarshalerClaims{})
	require.ErrorIs(t, err, ErrInvalidClaims)
}

func TestPayloadDecoding_RejectsMissingEmbeddedClaims(t *testing.T) {
	_, err := decodePayload[methodOnlyClaims]([]byte(`{}`))
	require.ErrorIs(t, err, ErrInvalidClaims)
}

func TestPayloadDecoding_RejectsMalformedStringClaim(t *testing.T) {
	_, err := parseStringClaim(map[string]json.RawMessage{
		"iss": json.RawMessage(`"\x"`),
	}, "iss")
	require.Error(t, err)
}

// TestUniqueJSONObject_RejectsEveryTruncationBoundary guarantees malformed
// objects fail whether truncation occurs in a value, array, or object.
func TestUniqueJSONObject_RejectsEveryTruncationBoundary(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "empty input", payload: ``},
		{name: "trailing object", payload: `{} {}`},
		{name: "invalid trailing data", payload: `{} x`},
		{name: "missing value", payload: `{"key":`},
		{name: "unterminated array", payload: `{"key":[`},
		{name: "invalid value in array", payload: `{"key":[{"nested":`},
		{name: "unterminated object key", payload: `{"key":1,"unterminated`},
		{name: "unterminated object", payload: `{"key":1`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, requireUniqueJSONObject([]byte(test.payload)))
		})
	}
}

func TestLocalDecrypt_RejectsAuthenticatedInvalidPayload(t *testing.T) {
	key := make([]byte, localKeySize)
	token, err := encryptToken(key, []byte(`[]`), nil, nil, bytes.NewReader(make([]byte, nonceSize)))
	require.NoError(t, err)
	local := newTestLocal[testClaims](t, key)

	_, err = local.Decrypt(token)
	require.ErrorIs(t, err, ErrInvalidToken)
}

// TestPayloadDecoding_AcceptsRFC3339OffsetsAndFractions guarantees the current
// DateTime profile supports offsets and fractional seconds.
func TestPayloadDecoding_AcceptsRFC3339OffsetsAndFractions(t *testing.T) {
	privateKey, verifier := newPayloadVerifier[testClaims](t)
	token := signToken(privateKey, []byte(`{"exp":"2030-01-02T03:04:05.123456789+05:30"}`), nil, nil)

	message, err := verifier.Verify(token)
	require.NoError(t, err)
	expected, err := time.Parse(time.RFC3339Nano, "2030-01-02T03:04:05.123456789+05:30")
	require.NoError(t, err)
	require.Equal(t, expected, message.Payload.ExpiresAt)
}

// TestPayloadDecoding_AcceptsRFC3339LeapSecond guarantees a registered
// DateTime claim can represent the leap-second syntax allowed by RFC 3339.
func TestPayloadDecoding_AcceptsRFC3339LeapSecond(t *testing.T) {
	privateKey, verifier := newPayloadVerifier[testClaims](t)
	token := signToken(privateKey, []byte(`{"exp":"1990-12-31T23:59:60Z"}`), nil, nil)

	message, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Equal(t, time.Date(1991, 1, 1, 0, 0, 0, 0, time.UTC), message.Payload.ExpiresAt)
}

// TestPayloadDecoding_TreatsRegisteredNamesAsCaseSensitive guarantees custom
// keys that differ only by case do not populate a registered claim. The PASETO
// registered-claims table states that names are case-sensitive.
func TestPayloadDecoding_TreatsRegisteredNamesAsCaseSensitive(t *testing.T) {
	privateKey, verifier := newPayloadVerifier[caseSensitiveClaims](t)

	token := signToken(privateKey, []byte(`{"ISS":"custom"}`), nil, nil)
	message, err := verifier.Verify(token)
	require.NoError(t, err)
	require.Empty(t, message.Payload.Issuer)
	require.Equal(t, "custom", message.Payload.UpperIssuer)

	token = signToken(privateKey, []byte(`{"iSs":"ignored"}`), nil, nil)
	message, err = verifier.Verify(token)
	require.NoError(t, err)
	require.Empty(t, message.Payload.Issuer)
	require.Empty(t, message.Payload.UpperIssuer)
}

func newPayloadVerifier[T ClaimSet](t *testing.T) (ed25519.PrivateKey, *PublicVerifier[T]) {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	verifier, err := NewVerifier[T](privateKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	return privateKey, verifier
}
