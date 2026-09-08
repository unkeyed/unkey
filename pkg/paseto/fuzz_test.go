package paseto

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzV4Public_AuthenticatedPayloadParsingIsSafe guarantees that an
// authenticated byte sequence is either rejected or decoded as valid claims.
// This protects the JSON and registered-claim parsing boundary.
func FuzzV4Public_AuthenticatedPayloadParsingIsSafe(f *testing.F) {
	fuzz.Seed(f)

	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	verifier, err := NewVerifier[testClaims](privateKey.Public().(ed25519.PublicKey))
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, data []byte) {
		consumer := fuzz.New(t, data)
		payload := consumer.BytesN(consumer.Remaining())
		token := signToken(privateKey, payload, nil, nil)

		message, err := verifier.Verify(token)
		if err != nil {
			require.ErrorIs(t, err, ErrInvalidToken)
			return
		}
		_, err = encodePayload(message.Payload)
		require.NoError(t, err)
	})
}

// FuzzV4_RoundTripPreservesTypedMessage guarantees that both v4 purposes
// preserve typed claims and arbitrary footer bytes.
func FuzzV4_RoundTripPreservesTypedMessage(f *testing.F) {
	fuzz.Seed(f)
	f.Add(make([]byte, localKeySize+4))
	privateKey := ed25519.NewKeyFromSeed(bytesWithFirstByte(ed25519.SeedSize, 1))
	signer, err := NewSigner[testClaims](privateKey)
	require.NoError(f, err)
	verifier, err := NewVerifier[testClaims](privateKey.Public().(ed25519.PublicKey))
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, data []byte) {
		consumer := fuzz.New(t, data)
		key := consumer.BytesN(localKeySize)
		role := base64.RawURLEncoding.EncodeToString(consumer.BytesN(consumer.Remaining() / 2))
		footer := append([]byte(nil), consumer.BytesN(consumer.Remaining())...)
		message := Message[testClaims]{
			Payload: testClaims{
				Claims: Claims{
					Issuer:    "issuer",
					Subject:   "subject",
					Audience:  "audience",
					ExpiresAt: time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC),
					NotBefore: time.Date(2029, 1, 2, 3, 4, 5, 0, time.UTC),
					IssuedAt:  time.Date(2029, 1, 2, 3, 4, 5, 0, time.UTC),
					TokenID:   "token-id",
				},
				Role: role,
			},
			Footer: footer,
		}

		localKey, err := NewLocalKey(key)
		require.NoError(t, err)
		local, err := NewLocal[testClaims](localKey)
		require.NoError(t, err)
		localToken, err := local.Encrypt(message)
		require.NoError(t, err)
		decrypted, err := local.Decrypt(localToken)
		require.NoError(t, err)
		require.Equal(t, message, decrypted)

		publicToken, err := signer.Sign(message)
		require.NoError(t, err)
		verified, err := verifier.Verify(publicToken)
		require.NoError(t, err)
		require.Equal(t, message, verified)
	})
}

// FuzzV4Local_TamperingIsRejected guarantees that changing any stored body
// byte causes local-token authentication to fail.
func FuzzV4Local_TamperingIsRejected(f *testing.F) {
	fuzz.Seed(f)
	seed := make([]byte, localKeySize+5)
	seed[len(seed)-1] = 1
	f.Add(seed)

	f.Fuzz(func(t *testing.T, data []byte) {
		consumer := fuzz.New(t, data)
		key := consumer.BytesN(localKeySize)
		if consumer.Remaining() < 3 {
			t.Skip("input cannot select and change a body byte")
		}
		role := base64.RawURLEncoding.EncodeToString(consumer.BytesN(consumer.Remaining() - 3))
		position := consumer.Uint16()
		mask := consumer.Uint8()
		if mask == 0 {
			t.Skip("XOR mask does not change the token")
		}

		localKey, err := NewLocalKey(key)
		require.NoError(t, err)
		local, err := NewLocal[testClaims](localKey)
		require.NoError(t, err)
		token, err := local.Encrypt(Message[testClaims]{
			Payload: testClaims{
				Claims: Claims{},
				Role:   role,
			},
			Footer: []byte("footer"),
		})
		require.NoError(t, err)
		parsed, err := parseToken(token, localHeader)
		require.NoError(t, err)

		body := append([]byte(nil), parsed.body...)
		body[int(position)%len(body)] ^= mask
		tampered := formatToken(localHeader, body, parsed.footer)
		_, err = local.Decrypt(tampered)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}

// FuzzV4Public_TamperingIsRejected guarantees that changing any stored body
// byte causes public-token signature verification to fail.
func FuzzV4Public_TamperingIsRejected(f *testing.F) {
	fuzz.Seed(f)
	seed := make([]byte, 5)
	seed[len(seed)-1] = 1
	f.Add(seed)

	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	signer, err := NewSigner[testClaims](privateKey)
	require.NoError(f, err)
	verifier, err := NewVerifier[testClaims](privateKey.Public().(ed25519.PublicKey))
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, data []byte) {
		consumer := fuzz.New(t, data)
		if consumer.Remaining() < 3 {
			t.Skip("input cannot select and change a body byte")
		}
		role := base64.RawURLEncoding.EncodeToString(consumer.BytesN(consumer.Remaining() - 3))
		position := consumer.Uint16()
		mask := consumer.Uint8()
		if mask == 0 {
			t.Skip("XOR mask does not change the token")
		}

		token, err := signer.Sign(Message[testClaims]{
			Payload: testClaims{
				Claims: Claims{},
				Role:   role,
			},
			Footer: []byte("footer"),
		})
		require.NoError(t, err)
		parsed, err := parseToken(token, publicHeader)
		require.NoError(t, err)

		body := append([]byte(nil), parsed.body...)
		body[int(position)%len(body)] ^= mask
		tampered := formatToken(publicHeader, body, parsed.footer)
		_, err = verifier.Verify(tampered)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}

// FuzzV4TokenParsing_ArbitraryInputIsSafe guarantees that token APIs
// reject malformed input without returning an unrelated error or panicking.
func FuzzV4TokenParsing_ArbitraryInputIsSafe(f *testing.F) {
	fuzz.Seed(f)

	localKey, err := NewLocalKey(make([]byte, localKeySize))
	require.NoError(f, err)
	local, err := NewLocal[testClaims](localKey)
	require.NoError(f, err)
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	verifier, err := NewVerifier[testClaims](privateKey.Public().(ed25519.PublicKey))
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, data []byte) {
		consumer := fuzz.New(t, data)
		token := string(consumer.BytesN(consumer.Remaining()))

		if _, err := UnverifiedFooter(token); err != nil {
			require.ErrorIs(t, err, ErrInvalidToken)
		}
		if _, err := local.Decrypt(token); err != nil {
			require.ErrorIs(t, err, ErrInvalidToken)
		}
		if _, err := verifier.Verify(token); err != nil {
			require.ErrorIs(t, err, ErrInvalidToken)
		}
	})
}
