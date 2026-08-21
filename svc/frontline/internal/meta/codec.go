// Package meta encodes and authenticates metadata sent between Frontline
// regions.
package meta

import (
	"fmt"

	"github.com/unkeyed/unkey/pkg/jwt"
)

const maxTokenBytes = 4096

// Codec marshals and unmarshals metadata as an HS256 JWT.
type Codec struct {
	signer   *jwt.HS256Signer[Metadata]
	verifier *jwt.HS256Verifier[Metadata]
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
	jwt.RegisteredClaims
	Hops []Hop `json:"hops"`
}

// New creates a metadata codec.
func New(signingKey string) (*Codec, error) {
	signer, err := jwt.NewHS256Signer[Metadata]([]byte(signingKey))
	if err != nil {
		return nil, fmt.Errorf("create metadata signer: %w", err)
	}
	verifier, err := jwt.NewHS256Verifier[Metadata]([]byte(signingKey))
	if err != nil {
		return nil, fmt.Errorf("create metadata verifier: %w", err)
	}
	return &Codec{signer: signer, verifier: verifier}, nil
}

// Marshal returns signed metadata.
func (c *Codec) Marshal(payload *Metadata) (string, error) {
	if payload == nil {
		return "", fmt.Errorf("metadata payload is required")
	}

	token, err := c.signer.Sign(*payload)
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

	payload, err := c.verifier.Verify(token)
	if err != nil {
		return nil, fmt.Errorf("verify metadata: %w", err)
	}

	return &payload, nil
}
