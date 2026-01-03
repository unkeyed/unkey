package keys

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/base58"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
)

func (s *service) CreateKey(ctx context.Context, req CreateKeyRequest) (CreateKeyResponse, error) {
	// Validate input parameters
	err := assert.InRange(req.ByteLength, 16, 255, "byte length must be between 16 and 255")
	if err != nil {
		return CreateKeyResponse{}, fault.Wrap(err,
			fault.Public("Invalid key byte length. Must be between 16 and 255 bytes."),
		)
	}

	// Generate random bytes for the key
	keyBytes := make([]byte, req.ByteLength)
	_, err = rand.Read(keyBytes)
	if err != nil {
		return CreateKeyResponse{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to generate random key bytes"),
			fault.Public("Failed to generate secure key."),
		)
	}

	// Create the full key string
	encodedKey := base58.Encode(keyBytes)
	fullKey := encodedKey
	start := encodedKey[:4]

	// Add prefix if provided and not empty
	if req.Prefix != "" {
		fullKey = fmt.Sprintf("%s_%s", req.Prefix, encodedKey)
		start = fmt.Sprintf("%s_%s", req.Prefix, encodedKey[:4])
	}

	return CreateKeyResponse{
		Key:   fullKey,
		Hash:  hash.Sha256(fullKey),
		Start: start,
	}, nil
}
