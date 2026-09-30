package paseto

import (
	"crypto/ed25519"
	cryptorand "crypto/rand"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestKeyConstruction_RejectsInvalidLengths guarantees each purpose accepts
// only the key length fixed by the v4 protocol.
func TestKeyConstruction_RejectsInvalidLengths(t *testing.T) {
	for _, length := range []int{0, localKeySize - 1, localKeySize + 1} {
		_, err := NewLocalKey(make([]byte, length))
		require.ErrorIs(t, err, ErrInvalidKey)
	}
	for _, length := range []int{0, ed25519.PrivateKeySize - 1, ed25519.PrivateKeySize + 1} {
		_, err := NewSigner[testClaims](make(ed25519.PrivateKey, length))
		require.ErrorIs(t, err, ErrInvalidKey)
	}
	for _, length := range []int{0, ed25519.PublicKeySize - 1, ed25519.PublicKeySize + 1} {
		_, err := NewVerifier[testClaims](make(ed25519.PublicKey, length))
		require.ErrorIs(t, err, ErrInvalidKey)
	}
}

// TestNewSigner_RejectsInconsistentPrivateKey guarantees the public half of an
// Ed25519 private key must correspond to its seed.
func TestNewSigner_RejectsInconsistentPrivateKey(t *testing.T) {
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	key[len(key)-1] ^= 1
	_, err := NewSigner[testClaims](key)
	require.ErrorIs(t, err, ErrInvalidKey)
}

// TestKeyConstruction_CopiesInput guarantees later changes to caller-owned key
// slices cannot change a processor's cryptographic identity.
func TestKeyConstruction_CopiesInput(t *testing.T) {
	localInput := make([]byte, localKeySize)
	localKey, err := NewLocalKey(localInput)
	require.NoError(t, err)
	local, err := NewLocal[testClaims](localKey)
	require.NoError(t, err)
	localInput[0] = 1
	token, err := local.Encrypt(testMessage())
	require.NoError(t, err)
	originalLocal := newTestLocal[testClaims](t, make([]byte, localKeySize))
	_, err = originalLocal.Decrypt(token)
	require.NoError(t, err)

	privateInput := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	publicInput := append(ed25519.PublicKey(nil), privateInput.Public().(ed25519.PublicKey)...)
	signer, err := NewSigner[testClaims](privateInput)
	require.NoError(t, err)
	verifier, err := NewVerifier[testClaims](publicInput)
	require.NoError(t, err)
	privateInput[0] = 1
	publicInput[0] ^= 1
	publicToken, err := signer.Sign(testMessage())
	require.NoError(t, err)
	_, err = verifier.Verify(publicToken)
	require.NoError(t, err)
}

// TestZeroValues_FailClosed guarantees bypassing constructors cannot create an
// active local key or processor with incomplete key state.
func TestZeroValues_FailClosed(t *testing.T) {
	message := testMessage()
	var localKey LocalKey
	_, err := NewLocal[testClaims](localKey)
	require.ErrorIs(t, err, ErrInvalidKey)
	_, err = localKey.MarshalBinary()
	require.ErrorIs(t, err, ErrInvalidKey)

	var local Local[testClaims]
	_, err = local.Encrypt(message)
	require.ErrorIs(t, err, ErrInvalidKey)
	_, err = local.Decrypt("token")
	require.ErrorIs(t, err, ErrInvalidKey)
	var nilLocal *Local[testClaims]
	_, err = nilLocal.Encrypt(message)
	require.ErrorIs(t, err, ErrInvalidKey)
	_, err = nilLocal.Decrypt("token")
	require.ErrorIs(t, err, ErrInvalidKey)

	var signer PublicSigner[testClaims]
	_, err = signer.Sign(message)
	require.ErrorIs(t, err, ErrInvalidKey)
	var nilSigner *PublicSigner[testClaims]
	_, err = nilSigner.Sign(message)
	require.ErrorIs(t, err, ErrInvalidKey)
	var verifier PublicVerifier[testClaims]
	_, err = verifier.Verify("token")
	require.ErrorIs(t, err, ErrInvalidKey)
	var nilVerifier *PublicVerifier[testClaims]
	_, err = nilVerifier.Verify("token")
	require.ErrorIs(t, err, ErrInvalidKey)
}

// TestGenerateLocalKey_ReturnsUsableKey guarantees generated keys can be
// persisted without exposing mutable package-owned material.
func TestGenerateLocalKey_ReturnsUsableKey(t *testing.T) {
	key, err := GenerateLocalKey()
	require.NoError(t, err)
	material, err := key.MarshalBinary()
	require.NoError(t, err)
	require.Len(t, material, localKeySize)
	_, err = NewLocal[testClaims](key)
	require.NoError(t, err)

	material[0] ^= 1
	materialAgain, err := key.MarshalBinary()
	require.NoError(t, err)
	require.NotEqual(t, material, materialAgain)
}

func TestGenerateLocalKey_PropagatesRandomSourceFailure(t *testing.T) {
	originalReader := cryptorand.Reader
	cryptorand.Reader = errorReader{}
	t.Cleanup(func() {
		cryptorand.Reader = originalReader
	})

	_, err := GenerateLocalKey()
	require.ErrorContains(t, err, "random source failed")
}

// TestProcessors_RejectInvalidClaimTypes guarantees all public constructors
// enforce the same typed-payload contract.
func TestProcessors_RejectInvalidClaimTypes(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	_, err := NewSigner[methodOnlyClaims](privateKey)
	require.ErrorIs(t, err, ErrInvalidClaims)
	_, err = NewVerifier[methodOnlyClaims](privateKey.Public().(ed25519.PublicKey))
	require.ErrorIs(t, err, ErrInvalidClaims)
}

func TestLocalProtocol_RejectsInvalidKeyLength(t *testing.T) {
	_, err := encryptToken(make([]byte, localKeySize-1), []byte(`{}`), nil, nil, errorReader{})
	require.ErrorIs(t, err, ErrInvalidKey)
	_, _, err = decryptToken(make([]byte, localKeySize-1), "token", nil)
	require.ErrorIs(t, err, ErrInvalidKey)
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("random source failed")
}

// TestEncrypt_PropagatesRandomSourceFailure guarantees encryption cannot
// continue with a missing or partial nonce.
func TestEncrypt_PropagatesRandomSourceFailure(t *testing.T) {
	_, err := encryptToken(make([]byte, localKeySize), []byte(`{}`), nil, nil, errorReader{})
	require.ErrorContains(t, err, "random source failed")
}
