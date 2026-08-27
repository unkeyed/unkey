package paseto

import (
	"crypto/ed25519"
	"crypto/subtle"
	"fmt"
)

const signatureSize = ed25519.SignatureSize

// PublicSigner signs v4.public tokens for one payload type.
type PublicSigner[T ClaimSet] struct {
	key   [ed25519.PrivateKeySize]byte
	valid bool
}

// PublicVerifier verifies v4.public tokens for one payload type.
type PublicVerifier[T ClaimSet] struct {
	key   [ed25519.PublicKeySize]byte
	valid bool
}

// NewSigner creates a v4.public signer. NewSigner copies and validates the
// Ed25519 private key.
func NewSigner[T ClaimSet](key ed25519.PrivateKey) (*PublicSigner[T], error) {
	if len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("%w: Ed25519 private key must contain %d bytes", ErrInvalidKey, ed25519.PrivateKeySize)
	}
	expectedKey := ed25519.NewKeyFromSeed(key[:ed25519.SeedSize])
	if subtle.ConstantTimeCompare(key, expectedKey) != 1 {
		return nil, fmt.Errorf("%w: Ed25519 private key has an inconsistent public key", ErrInvalidKey)
	}
	if err := validateClaimsType[T](); err != nil {
		return nil, err
	}
	signer := &PublicSigner[T]{
		key:   [ed25519.PrivateKeySize]byte{},
		valid: true,
	}
	copy(signer.key[:], key)
	return signer, nil
}

// NewVerifier creates a v4.public verifier. NewVerifier copies the Ed25519
// public key.
func NewVerifier[T ClaimSet](key ed25519.PublicKey) (*PublicVerifier[T], error) {
	if len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: Ed25519 public key must contain %d bytes", ErrInvalidKey, ed25519.PublicKeySize)
	}
	if err := validateClaimsType[T](); err != nil {
		return nil, err
	}
	verifier := &PublicVerifier[T]{
		key:   [ed25519.PublicKeySize]byte{},
		valid: true,
	}
	copy(verifier.key[:], key)
	return verifier, nil
}

// Sign signs and authenticates a message as a v4.public token.
func (signer *PublicSigner[T]) Sign(message Message[T]) (string, error) {
	if signer == nil || !signer.valid {
		return "", ErrInvalidKey
	}
	payload, err := encodePayload(message.Payload)
	if err != nil {
		return "", err
	}
	return signToken(ed25519.PrivateKey(signer.key[:]), payload, message.Footer, nil), nil
}

// Verify authenticates a v4.public token. It does not evaluate temporal or
// application-specific claims.
func (verifier *PublicVerifier[T]) Verify(token string) (Message[T], error) {
	if verifier == nil || !verifier.valid {
		return Message[T]{}, ErrInvalidKey
	}
	payloadJSON, footer, err := verifyToken(ed25519.PublicKey(verifier.key[:]), token, nil)
	if err != nil {
		return Message[T]{}, err
	}
	payload, err := decodePayload[T](payloadJSON)
	if err != nil {
		return Message[T]{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return Message[T]{Payload: payload, Footer: append([]byte(nil), footer...)}, nil
}

func signToken(key ed25519.PrivateKey, payload []byte, footer []byte, implicitAssertion []byte) string {
	authenticationInput := preAuthEncode([]byte(publicHeader), payload, footer, implicitAssertion)
	signature := ed25519.Sign(key, authenticationInput)
	body := make([]byte, 0, len(payload)+len(signature))
	body = append(body, payload...)
	body = append(body, signature...)
	return formatToken(publicHeader, body, footer)
}

func verifyToken(key ed25519.PublicKey, token string, implicitAssertion []byte) ([]byte, []byte, error) {
	parsed, err := parseToken(token, publicHeader)
	if err != nil {
		return nil, nil, err
	}
	signatureOffset := len(parsed.body) - signatureSize
	payload := parsed.body[:signatureOffset]
	signature := parsed.body[signatureOffset:]
	authenticationInput := preAuthEncode([]byte(publicHeader), payload, parsed.footer, implicitAssertion)
	if !ed25519.Verify(key, authenticationInput, signature) {
		return nil, nil, fmt.Errorf("%w: signature verification failed", ErrInvalidToken)
	}
	return payload, parsed.footer, nil
}

var _ Signer[Claims] = (*PublicSigner[Claims])(nil)

var _ Verifier[Claims] = (*PublicVerifier[Claims])(nil)
