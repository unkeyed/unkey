package paseto_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/paseto"
)

type sessionClaims struct {
	paseto.Claims

	WorkspaceID string   `json:"workspace_id"`
	Permissions []string `json:"permissions"`
}

func encrypt[T paseto.ClaimSet](encrypter paseto.Encrypter[T], message paseto.Message[T]) (string, error) {
	return encrypter.Encrypt(message)
}

// TestPublicAPI_TypedPayloadRoundTrips guarantees an external caller can embed
// paseto.Claims, add typed custom claims, and use every capability interface.
func TestPublicAPI_TypedPayloadRoundTrips(t *testing.T) {
	localKeyMaterial := make([]byte, 32)
	for index := range localKeyMaterial {
		localKeyMaterial[index] = byte(index)
	}
	localKey, err := paseto.NewLocalKey(localKeyMaterial)
	require.NoError(t, err)
	local, err := paseto.NewLocal[sessionClaims](localKey)
	require.NoError(t, err)

	seed := make([]byte, ed25519.SeedSize)
	for index := range seed {
		seed[index] = byte(index + 1)
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	signer, err := paseto.NewSigner[sessionClaims](privateKey)
	require.NoError(t, err)
	verifier, err := paseto.NewVerifier[sessionClaims](publicKey)
	require.NoError(t, err)

	var encrypter paseto.Encrypter[sessionClaims] = local
	var decrypter paseto.Decrypter[sessionClaims] = local
	var publicSigner paseto.Signer[sessionClaims] = signer
	var publicVerifier paseto.Verifier[sessionClaims] = verifier

	message := paseto.Message[sessionClaims]{
		Payload: sessionClaims{
			Claims: paseto.Claims{
				Issuer:    "api.unkey.com",
				Subject:   "user_123",
				Audience:  "dashboard",
				ExpiresAt: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
				NotBefore: time.Date(2029, 1, 2, 3, 4, 5, 0, time.UTC),
				IssuedAt:  time.Date(2029, 1, 2, 3, 4, 5, 0, time.UTC),
				TokenID:   "token_123",
			},
			WorkspaceID: "ws_123",
			Permissions: []string{"keys.read", "keys.write"},
		},
		Footer: []byte(`{"kid":"current"}`),
	}

	localToken, err := encrypt(encrypter, message)
	require.NoError(t, err)
	decrypted, err := decrypter.Decrypt(localToken)
	require.NoError(t, err)
	require.Equal(t, message, decrypted)

	publicToken, err := publicSigner.Sign(message)
	require.NoError(t, err)
	verified, err := publicVerifier.Verify(publicToken)
	require.NoError(t, err)
	require.Equal(t, message, verified)

	parts := strings.Split(publicToken, ".")
	require.Len(t, parts, 4)
	body, err := base64.RawURLEncoding.Strict().DecodeString(parts[2])
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(body), ed25519.SignatureSize)
	payloadJSON := body[:len(body)-ed25519.SignatureSize]
	require.True(t, json.Valid(payloadJSON))
	require.JSONEq(t, `{
		"iss":"api.unkey.com",
		"sub":"user_123",
		"aud":"dashboard",
		"exp":"2030-01-02T03:04:05Z",
		"nbf":"2029-01-02T03:04:05Z",
		"iat":"2029-01-02T03:04:05Z",
		"jti":"token_123",
		"workspace_id":"ws_123",
		"permissions":["keys.read","keys.write"]
	}`, string(payloadJSON))
}

// TestPublicAPI_DoesNotApplyTemporalPolicy documents that PASETO authenticates
// exp and nbf claim values but leaves their policy to the caller.
func TestPublicAPI_DoesNotApplyTemporalPolicy(t *testing.T) {
	localKey, err := paseto.NewLocalKey(make([]byte, 32))
	require.NoError(t, err)
	local, err := paseto.NewLocal[sessionClaims](localKey)
	require.NoError(t, err)
	message := paseto.Message[sessionClaims]{
		Payload: sessionClaims{
			Claims: paseto.Claims{
				ExpiresAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			WorkspaceID: "ws_123",
		},
		Footer: nil,
	}

	token, err := local.Encrypt(message)
	require.NoError(t, err)
	decrypted, err := local.Decrypt(token)
	require.NoError(t, err)
	require.Equal(t, message, decrypted)
}

// TestPublicAPI_RegisteredClaimsOnlyRoundTrips guarantees callers do not need
// a custom payload type when they use only registered claims.
func TestPublicAPI_RegisteredClaimsOnlyRoundTrips(t *testing.T) {
	localKey, err := paseto.NewLocalKey(make([]byte, 32))
	require.NoError(t, err)
	local, err := paseto.NewLocal[paseto.Claims](localKey)
	require.NoError(t, err)
	message := paseto.Message[paseto.Claims]{
		Payload: paseto.Claims{
			Issuer:  "issuer",
			Subject: "subject",
		},
		Footer: nil,
	}

	token, err := local.Encrypt(message)
	require.NoError(t, err)
	decrypted, err := local.Decrypt(token)
	require.NoError(t, err)
	require.Equal(t, message, decrypted)
}
