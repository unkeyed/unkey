package paseto

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestV4Local_AuthenticatesEveryStoredComponent guarantees the header, nonce,
// ciphertext, authentication tag, and footer cannot change without rejection.
// This protects the complete-message authentication rule in RFC section 3.1.
func TestV4Local_AuthenticatesEveryStoredComponent(t *testing.T) {
	local := newTestLocal[testClaims](t, make([]byte, localKeySize))
	token, err := local.Encrypt(testMessage())
	require.NoError(t, err)

	parsed, err := parseToken(token, localHeader)
	require.NoError(t, err)
	ciphertextOffset := nonceSize
	require.Greater(t, len(parsed.body)-tagSize, ciphertextOffset)

	tests := []struct {
		name  string
		token string
	}{
		{name: "header", token: strings.Replace(token, localHeader, publicHeader, 1)},
		{name: "nonce", token: mutateTokenBody(t, token, 0)},
		{name: "ciphertext", token: mutateTokenBody(t, token, ciphertextOffset)},
		{name: "authentication tag", token: mutateTokenBody(t, token, len(parsed.body)-1)},
		{name: "footer", token: mutateTokenFooter(t, token)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := local.Decrypt(test.token)
			require.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

// TestV4Public_AuthenticatesEveryStoredComponent guarantees the header,
// payload, signature, and footer cannot change without rejection. This
// protects the complete-message authentication rule in RFC section 3.1.
func TestV4Public_AuthenticatesEveryStoredComponent(t *testing.T) {
	signer, verifier := newTestPublicProcessors(t)
	token, err := signer.Sign(testMessage())
	require.NoError(t, err)
	parsed, err := parseToken(token, publicHeader)
	require.NoError(t, err)
	require.Greater(t, len(parsed.body), signatureSize)

	tests := []struct {
		name  string
		token string
	}{
		{name: "header", token: strings.Replace(token, publicHeader, localHeader, 1)},
		{name: "payload", token: mutateTokenBody(t, token, 0)},
		{name: "signature", token: mutateTokenBody(t, token, len(parsed.body)-1)},
		{name: "footer", token: mutateTokenFooter(t, token)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := verifier.Verify(test.token)
			require.ErrorIs(t, err, ErrInvalidToken)
		})
	}
}

// TestV4_RejectsWrongKeys guarantees a token authenticates only with the key
// that created it.
func TestV4_RejectsWrongKeys(t *testing.T) {
	local := newTestLocal[testClaims](t, make([]byte, localKeySize))
	localToken, err := local.Encrypt(testMessage())
	require.NoError(t, err)
	otherLocalKey := make([]byte, localKeySize)
	otherLocalKey[0] = 1
	otherLocal := newTestLocal[testClaims](t, otherLocalKey)
	_, err = otherLocal.Decrypt(localToken)
	require.ErrorIs(t, err, ErrInvalidToken)

	signer, _ := newTestPublicProcessors(t)
	publicToken, err := signer.Sign(testMessage())
	require.NoError(t, err)
	otherPrivateKey := ed25519.NewKeyFromSeed(bytesWithFirstByte(ed25519.SeedSize, 1))
	otherVerifier, err := NewVerifier[testClaims](otherPrivateKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	_, err = otherVerifier.Verify(publicToken)
	require.ErrorIs(t, err, ErrInvalidToken)
}

// TestProcessors_ConcurrentUse protects the concurrency guarantee on all four
// capability interfaces.
func TestProcessors_ConcurrentUse(t *testing.T) {
	local := newTestLocal[testClaims](t, make([]byte, localKeySize))
	signer, verifier := newTestPublicProcessors(t)
	message := testMessage()
	localToken, err := local.Encrypt(message)
	require.NoError(t, err)
	publicToken, err := signer.Sign(message)
	require.NoError(t, err)

	errorsFound := make(chan error, 64)
	waitGroup := sync.WaitGroup{}
	for range 16 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if _, err := local.Encrypt(message); err != nil {
				errorsFound <- fmt.Errorf("encrypt: %w", err)
			}
			if _, err := local.Decrypt(localToken); err != nil {
				errorsFound <- fmt.Errorf("decrypt: %w", err)
			}
			if _, err := signer.Sign(message); err != nil {
				errorsFound <- fmt.Errorf("sign: %w", err)
			}
			if _, err := verifier.Verify(publicToken); err != nil {
				errorsFound <- fmt.Errorf("verify: %w", err)
			}
		}()
	}
	waitGroup.Wait()
	close(errorsFound)
	for err := range errorsFound {
		require.NoError(t, err)
	}
}

func testMessage() Message[testClaims] {
	return Message[testClaims]{
		Payload: testClaims{
			Claims: Claims{
				Issuer:    "issuer",
				Subject:   "subject",
				ExpiresAt: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
			},
			Role: "admin",
		},
		Footer: []byte(`{"kid":"current"}`),
	}
}

func newTestLocal[T ClaimSet](t *testing.T, material []byte) *Local[T] {
	t.Helper()
	key, err := NewLocalKey(material)
	require.NoError(t, err)
	local, err := NewLocal[T](key)
	require.NoError(t, err)
	return local
}

func newTestPublicProcessors(t *testing.T) (*PublicSigner[testClaims], *PublicVerifier[testClaims]) {
	t.Helper()
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	signer, err := NewSigner[testClaims](privateKey)
	require.NoError(t, err)
	verifier, err := NewVerifier[testClaims](privateKey.Public().(ed25519.PublicKey))
	require.NoError(t, err)
	return signer, verifier
}

func mutateTokenBody(t *testing.T, token string, index int) string {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 4)
	body, err := strictBase64URL.DecodeString(parts[2])
	require.NoError(t, err)
	require.Less(t, index, len(body))
	body[index] ^= 1
	parts[2] = strictBase64URL.EncodeToString(body)
	return strings.Join(parts, ".")
}

func mutateTokenFooter(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	require.Len(t, parts, 4)
	footer, err := base64.RawURLEncoding.Strict().DecodeString(parts[3])
	require.NoError(t, err)
	require.NotEmpty(t, footer)
	footer[0] ^= 1
	parts[3] = base64.RawURLEncoding.EncodeToString(footer)
	return strings.Join(parts, ".")
}

func bytesWithFirstByte(length int, first byte) []byte {
	value := make([]byte, length)
	value[0] = first
	return value
}
