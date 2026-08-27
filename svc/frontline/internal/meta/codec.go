// Package meta encodes and authenticates metadata sent between Frontline
// regions.
package meta

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"

	"github.com/unkeyed/unkey/pkg/paseto"
)

const maxTokenBytes = 4096

// Codec marshals and unmarshals metadata as a PASETO v4.public token.
type Codec struct {
	signer   *paseto.PublicSigner[Metadata]
	verifier *paseto.PublicVerifier[Metadata]
}

// Hop records one cross-region forward.
type Hop struct {
	Region        string `json:"region"`
	RequestID     string `json:"request_id"`
	FrontlineID   string `json:"frontline_id"`
	TimeUnixMilli int64  `json:"time"`
}

// Metadata is the trusted data sent between Frontline regions.
type Metadata struct {
	paseto.Claims
	Hops []Hop `json:"hops"`
}

// New creates a metadata codec from a hex-encoded Ed25519 seed.
func New(signingKey string) (*Codec, error) {
	seed, err := hex.DecodeString(signingKey)
	if err != nil {
		return nil, fmt.Errorf("decode metadata signing key: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("metadata signing key must contain %d hexadecimal characters", ed25519.SeedSize*2)
	}

	privateKey := ed25519.NewKeyFromSeed(seed)
	signer, err := paseto.NewSigner[Metadata](privateKey)
	if err != nil {
		return nil, fmt.Errorf("create metadata signer: %w", err)
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	verifier, err := paseto.NewVerifier[Metadata](publicKey)
	if err != nil {
		return nil, fmt.Errorf("create metadata verifier: %w", err)
	}
	return &Codec{signer: signer, verifier: verifier}, nil
}

// Marshal returns signed metadata.
func (c *Codec) Marshal(metadata *Metadata) (string, error) {
	if metadata == nil {
		return "", fmt.Errorf("metadata is required")
	}

	token, err := c.signer.Sign(paseto.Message[Metadata]{
		Payload: *metadata,
		Footer:  nil,
	})
	if err != nil {
		return "", fmt.Errorf("sign metadata: %w", err)
	}
	if len(token) > maxTokenBytes {
		return "", fmt.Errorf("encoded metadata exceeds %d bytes", maxTokenBytes)
	}

	return token, nil
}

// Unmarshal verifies and decodes signed metadata.
func (c *Codec) Unmarshal(token string) (*Metadata, error) {
	if token == "" {
		return nil, fmt.Errorf("encoded metadata is required")
	}
	if len(token) > maxTokenBytes {
		return nil, fmt.Errorf("encoded metadata exceeds %d bytes", maxTokenBytes)
	}

	message, err := c.verifier.Verify(token)
	if err != nil {
		return nil, fmt.Errorf("verify metadata: %w", err)
	}

	return &message.Payload, nil
}
