package paseto

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/chacha20"
)

const (
	localKeySize            = 32
	nonceSize               = 32
	tagSize                 = 32
	encryptionKeyDomain     = "paseto-encryption-key"
	authenticationKeyDomain = "paseto-auth-key-for-aead"
)

// LocalKey contains a purpose-bound v4.local key. Its zero value is invalid.
type LocalKey struct {
	material [localKeySize]byte
	valid    bool
}

// NewLocalKey creates a v4.local key from 32 bytes. NewLocalKey copies the key
// material.
func NewLocalKey(material []byte) (LocalKey, error) {
	if len(material) != localKeySize {
		return LocalKey{}, fmt.Errorf("%w: local key must contain %d bytes", ErrInvalidKey, localKeySize)
	}
	key := LocalKey{
		material: [localKeySize]byte{},
		valid:    true,
	}
	copy(key.material[:], material)
	return key, nil
}

// GenerateLocalKey returns a new v4.local key from the operating system's
// cryptographically secure random number generator.
func GenerateLocalKey() (LocalKey, error) {
	var key LocalKey
	if _, err := io.ReadFull(rand.Reader, key.material[:]); err != nil {
		return LocalKey{}, fmt.Errorf("generate PASETO local key: %w", err)
	}
	key.valid = true
	return key, nil
}

// MarshalBinary returns a copy of the 32-byte key material. Treat the returned
// bytes as secret.
func (key LocalKey) MarshalBinary() ([]byte, error) {
	if !key.valid {
		return nil, ErrInvalidKey
	}
	return append([]byte(nil), key.material[:]...), nil
}

// Local encrypts and decrypts v4.local tokens for one payload type.
type Local[T ClaimSet] struct {
	key   [localKeySize]byte
	valid bool
}

// NewLocal creates a v4.local token processor.
func NewLocal[T ClaimSet](key LocalKey) (*Local[T], error) {
	if !key.valid {
		return nil, ErrInvalidKey
	}
	if err := validateClaimsType[T](); err != nil {
		return nil, err
	}
	local := &Local[T]{
		key:   key.material,
		valid: true,
	}
	return local, nil
}

// Encrypt encrypts and authenticates a message as a v4.local token.
func (local *Local[T]) Encrypt(message Message[T]) (string, error) {
	if local == nil || !local.valid {
		return "", ErrInvalidKey
	}
	payload, err := encodePayload(message.Payload)
	if err != nil {
		return "", err
	}
	return encryptToken(local.key[:], payload, message.Footer, nil, rand.Reader)
}

// Decrypt authenticates and decrypts a v4.local token. It does not evaluate
// temporal or application-specific claims.
func (local *Local[T]) Decrypt(token string) (Message[T], error) {
	if local == nil || !local.valid {
		return Message[T]{}, ErrInvalidKey
	}
	payloadJSON, footer, err := decryptToken(local.key[:], token, nil)
	if err != nil {
		return Message[T]{}, err
	}
	payload, err := decodePayload[T](payloadJSON)
	if err != nil {
		return Message[T]{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return Message[T]{Payload: payload, Footer: append([]byte(nil), footer...)}, nil
}

func encryptToken(key []byte, payload []byte, footer []byte, implicitAssertion []byte, randomSource io.Reader) (string, error) {
	localKey, err := localKeyBytes(key)
	if err != nil {
		return "", err
	}
	var nonce [nonceSize]byte
	if _, err := io.ReadFull(randomSource, nonce[:]); err != nil {
		return "", fmt.Errorf("generate PASETO nonce: %w", err)
	}
	encryptionKey, counterNonce, authenticationKey := splitLocalKey(localKey, nonce)
	// These array sizes satisfy the XChaCha20 key and nonce requirements.
	cipher, _ := chacha20.NewUnauthenticatedCipher(encryptionKey[:], counterNonce[:])
	ciphertext := make([]byte, len(payload))
	cipher.XORKeyStream(ciphertext, payload)

	authenticationInput := preAuthEncode([]byte(localHeader), nonce[:], ciphertext, footer, implicitAssertion)
	tag := keyedBlake2b(authenticationKey, authenticationInput, tagSize)
	body := make([]byte, 0, len(nonce)+len(ciphertext)+len(tag))
	body = append(body, nonce[:]...)
	body = append(body, ciphertext...)
	body = append(body, tag...)
	return formatToken(localHeader, body, footer), nil
}

func decryptToken(key []byte, token string, implicitAssertion []byte) ([]byte, []byte, error) {
	localKey, err := localKeyBytes(key)
	if err != nil {
		return nil, nil, err
	}
	parsed, err := parseToken(token, localHeader)
	if err != nil {
		return nil, nil, err
	}
	var nonce [nonceSize]byte
	copy(nonce[:], parsed.body[:nonceSize])
	tagOffset := len(parsed.body) - tagSize
	ciphertext := parsed.body[nonceSize:tagOffset]
	tag := parsed.body[tagOffset:]

	encryptionKey, counterNonce, authenticationKey := splitLocalKey(localKey, nonce)
	authenticationInput := preAuthEncode([]byte(localHeader), nonce[:], ciphertext, parsed.footer, implicitAssertion)
	expectedTag := keyedBlake2b(authenticationKey, authenticationInput, tagSize)
	if subtle.ConstantTimeCompare(tag, expectedTag) != 1 {
		return nil, nil, fmt.Errorf("%w: authentication failed", ErrInvalidToken)
	}

	// These array sizes satisfy the XChaCha20 key and nonce requirements.
	cipher, _ := chacha20.NewUnauthenticatedCipher(encryptionKey[:], counterNonce[:])
	payload := make([]byte, len(ciphertext))
	cipher.XORKeyStream(payload, ciphertext)
	return payload, parsed.footer, nil
}

func localKeyBytes(material []byte) ([localKeySize]byte, error) {
	if len(material) != localKeySize {
		return [localKeySize]byte{}, fmt.Errorf("%w: local key must contain %d bytes", ErrInvalidKey, localKeySize)
	}
	var key [localKeySize]byte
	copy(key[:], material)
	return key, nil
}

func splitLocalKey(key [localKeySize]byte, nonce [nonceSize]byte) (
	encryptionKey [32]byte,
	counterNonce [24]byte,
	authenticationKey [32]byte,
) {
	encryptionMaterial := keyedBlake2b(key, keyDerivationInput(encryptionKeyDomain, nonce[:]), 56)
	copy(encryptionKey[:], encryptionMaterial[:32])
	copy(counterNonce[:], encryptionMaterial[32:])
	copy(authenticationKey[:], keyedBlake2b(key, keyDerivationInput(authenticationKeyDomain, nonce[:]), 32))
	return encryptionKey, counterNonce, authenticationKey
}

func keyDerivationInput(domain string, nonce []byte) []byte {
	input := make([]byte, 0, len(domain)+len(nonce))
	input = append(input, domain...)
	return append(input, nonce...)
}

func keyedBlake2b(key [localKeySize]byte, message []byte, size int) []byte {
	// The key has an accepted BLAKE2b size. Callers use only valid digest sizes.
	hash, _ := blake2b.New(size, key[:])
	// hash.Hash.Write never returns an error.
	_, _ = hash.Write(message)
	return hash.Sum(nil)
}

var _ Encrypter[Claims] = (*Local[Claims])(nil)

var _ Decrypter[Claims] = (*Local[Claims])(nil)
