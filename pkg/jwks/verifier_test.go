package jwks

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/jwt"
)

// staticKeySet is a test KeySet that returns a fixed key.
type staticKeySet struct {
	keys map[string]*rsa.PublicKey
}

func (s *staticKeySet) GetKey(_ context.Context, kid string) (*rsa.PublicKey, error) {
	key, ok := s.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found", kid)
	}
	return key, nil
}

type testClaims struct {
	jwt.RegisteredClaims
	OrgID string `json:"org_id"`
	Role  string `json:"role"`
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, claims any) string {
	t.Helper()

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	}

	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)

	payloadJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	sigInput := encodedHeader + "." + encodedPayload

	hash := sha256.Sum256([]byte(sigInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	require.NoError(t, err)

	return sigInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifier_ValidToken(t *testing.T) {
	key := generateTestKeyPair(t)
	ks := &staticKeySet{keys: map[string]*rsa.PublicKey{"kid-1": &key.PublicKey}}
	v := NewVerifier[testClaims](ks)

	claims := testClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			Issuer:    "test-issuer",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		OrgID: "org-abc",
		Role:  "admin",
	}

	token := signToken(t, key, "kid-1", claims)

	got, err := v.Verify(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "user-123", got.Subject)
	require.Equal(t, "org-abc", got.OrgID)
	require.Equal(t, "admin", got.Role)
}

func TestVerifier_ExpiredToken(t *testing.T) {
	key := generateTestKeyPair(t)
	ks := &staticKeySet{keys: map[string]*rsa.PublicKey{"kid-1": &key.PublicKey}}
	v := NewVerifier[testClaims](ks)

	claims := testClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: time.Now().Add(-time.Hour).Unix(),
		},
	}

	token := signToken(t, key, "kid-1", claims)

	_, err := v.Verify(context.Background(), token)
	require.Error(t, err)
	require.True(t, errors.Is(err, jwt.ErrTokenExpired))
}

func TestVerifier_IssuerValidation(t *testing.T) {
	key := generateTestKeyPair(t)
	ks := &staticKeySet{keys: map[string]*rsa.PublicKey{"kid-1": &key.PublicKey}}
	v := NewVerifier[testClaims](ks, WithIssuer("https://api.workos.com"))

	t.Run("valid issuer", func(t *testing.T) {
		claims := testClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "https://api.workos.com",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
		}
		token := signToken(t, key, "kid-1", claims)
		_, err := v.Verify(context.Background(), token)
		require.NoError(t, err)
	})

	t.Run("invalid issuer", func(t *testing.T) {
		claims := testClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "https://evil.example.com",
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
		}
		token := signToken(t, key, "kid-1", claims)
		_, err := v.Verify(context.Background(), token)
		require.Error(t, err)
		require.True(t, errors.Is(err, jwt.ErrInvalidIssuer))
	})
}

func TestVerifier_UnknownKid(t *testing.T) {
	key := generateTestKeyPair(t)
	ks := &staticKeySet{keys: map[string]*rsa.PublicKey{"kid-1": &key.PublicKey}}
	v := NewVerifier[testClaims](ks)

	claims := testClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}

	token := signToken(t, key, "kid-unknown", claims)

	_, err := v.Verify(context.Background(), token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolving signing key")
}

func TestVerifier_WrongSigningKey(t *testing.T) {
	signingKey := generateTestKeyPair(t)
	wrongKey := generateTestKeyPair(t)
	ks := &staticKeySet{keys: map[string]*rsa.PublicKey{"kid-1": &wrongKey.PublicKey}}
	v := NewVerifier[testClaims](ks)

	claims := testClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		},
	}

	token := signToken(t, signingKey, "kid-1", claims)

	_, err := v.Verify(context.Background(), token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid signature")
}

func TestVerifier_MalformedToken(t *testing.T) {
	key := generateTestKeyPair(t)
	ks := &staticKeySet{keys: map[string]*rsa.PublicKey{"kid-1": &key.PublicKey}}
	v := NewVerifier[testClaims](ks)

	_, err := v.Verify(context.Background(), "not.a.valid.token")
	require.Error(t, err)

	_, err = v.Verify(context.Background(), "only-one-part")
	require.Error(t, err)
}

func TestVerifier_UnsupportedAlgorithm(t *testing.T) {
	// Craft a token with alg=HS256 header.
	header := map[string]string{"alg": "HS256", "typ": "JWT", "kid": "kid-1"}
	headerJSON, _ := json.Marshal(header)
	payload := map[string]any{"sub": "user"}
	payloadJSON, _ := json.Marshal(payload)

	token := base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(payloadJSON) + ".fakesig"

	key := generateTestKeyPair(t)
	ks := &staticKeySet{keys: map[string]*rsa.PublicKey{"kid-1": &key.PublicKey}}
	v := NewVerifier[testClaims](ks)

	_, err := v.Verify(context.Background(), token)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported algorithm")
}
