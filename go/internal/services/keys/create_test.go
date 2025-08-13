package keys

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/hash"
)

func TestCreateKey_WithoutPrefix(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	req := CreateKeyRequest{
		Prefix:     "",
		ByteLength: 16,
	}

	resp, err := s.CreateKey(ctx, req)
	require.NoError(t, err)

	// Verify response structure
	require.NotEmpty(t, resp.Key)
	require.NotEmpty(t, resp.Hash)
	require.NotEmpty(t, resp.Start)

	// Verify key doesn't contain underscore (no prefix)
	require.False(t, strings.Contains(resp.Key, "_"))

	// Verify hash matches the generated key
	expectedHash := hash.Sha256(resp.Key)
	require.Equal(t, expectedHash, resp.Hash)

	if len(resp.Key) <= 8 {
		require.Equal(t, resp.Key, resp.Start)
	} else {
		require.Equal(t, resp.Key[:4], resp.Start)
	}
}

func TestCreateKey_WithPrefix(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	prefix := "test"
	req := CreateKeyRequest{
		Prefix:     prefix,
		ByteLength: 16,
	}

	resp, err := s.CreateKey(ctx, req)
	require.NoError(t, err)

	// Verify response structure
	require.NotEmpty(t, resp.Key)
	require.NotEmpty(t, resp.Hash)
	require.NotEmpty(t, resp.Start)

	// Verify key starts with prefix
	require.True(t, strings.HasPrefix(resp.Key, "test_"))

	// Verify hash matches the generated key
	expectedHash := hash.Sha256(resp.Key)
	require.Equal(t, expectedHash, resp.Hash)

	// Verify start is correct
	require.Equal(t, resp.Key[:9], resp.Start)
}

func TestCreateKey_EmptyPrefix(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	req := CreateKeyRequest{
		Prefix:     "",
		ByteLength: 16,
	}

	resp, err := s.CreateKey(ctx, req)
	require.NoError(t, err)

	// Verify key doesn't contain underscore (empty prefix treated as no prefix)
	require.False(t, strings.Contains(resp.Key, "_"))
}

func TestCreateKey_DifferentByteLengths(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	testCases := []struct {
		name       string
		byteLength int
	}{
		{"minimum", 16},
		{"medium", 32},
		{"large", 64},
		{"maximum", 255},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := CreateKeyRequest{
				Prefix:     "",
				ByteLength: tc.byteLength,
			}

			resp, err := s.CreateKey(ctx, req)
			require.NoError(t, err)

			// Verify key was generated
			require.NotEmpty(t, resp.Key)

			// Verify hash is correct
			expectedHash := hash.Sha256(resp.Key)
			require.Equal(t, expectedHash, resp.Hash)
		})
	}
}

func TestCreateKey_LongPrefix(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	req := CreateKeyRequest{
		Prefix:     "verylongprefix",
		ByteLength: 16,
	}

	resp, err := s.CreateKey(ctx, req)
	require.NoError(t, err)

	// Verify key starts with the long prefix
	require.True(t, strings.HasPrefix(resp.Key, "verylongprefix_"))

	// Verify start is truncated to 8 characters when key is longer
	require.Equal(t, resp.Key[:19], resp.Start)
	require.Len(t, resp.Start, 19)
}

func TestCreateKey_Uniqueness(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	// Generate multiple keys with same parameters
	keys := make(map[string]bool)
	hashes := make(map[string]bool)

	for range 10 {
		req := CreateKeyRequest{
			Prefix:     "",
			ByteLength: 16,
		}

		resp, err := s.CreateKey(ctx, req)
		require.NoError(t, err)

		// Verify each key is unique
		require.False(t, keys[resp.Key], "Generated duplicate key: %s", resp.Key)
		require.False(t, hashes[resp.Hash], "Generated duplicate hash: %s", resp.Hash)

		keys[resp.Key] = true
		hashes[resp.Hash] = true
	}
}

func TestCreateKey_ConsistentHashing(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	req := CreateKeyRequest{
		Prefix:     "",
		ByteLength: 16,
	}

	resp, err := s.CreateKey(ctx, req)
	require.NoError(t, err)

	// Verify that hashing the key again produces the same result
	rehashed := hash.Sha256(resp.Key)
	require.Equal(t, resp.Hash, rehashed)

	// Verify hash is not empty and has expected format
	require.NotEmpty(t, resp.Hash)
	require.Greater(t, len(resp.Hash), 40) // SHA-256 base64 should be longer than 40 chars
}

func TestCreateKey_ByteLengthValidation_TooSmall(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	req := CreateKeyRequest{
		Prefix:     "",
		ByteLength: 8, // Less than minimum of 16
	}

	resp, err := s.CreateKey(ctx, req)
	require.Error(t, err)
	require.Empty(t, resp.Key)
	require.Contains(t, err.Error(), "byte length must be between 16 and 255")
}

func TestCreateKey_ByteLengthValidation_TooLarge(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	req := CreateKeyRequest{
		Prefix:     "",
		ByteLength: 300, // Greater than maximum of 255
	}

	resp, err := s.CreateKey(ctx, req)
	require.Error(t, err)
	require.Empty(t, resp.Key)
	require.Contains(t, err.Error(), "byte length must be between 16 and 255")
}

func TestCreateKey_ByteLengthValidation_BoundaryValues(t *testing.T) {
	t.Parallel()

	s := &service{}
	ctx := context.Background()

	testCases := []struct {
		name       string
		byteLength int
		shouldPass bool
	}{
		{"below minimum", 15, false},
		{"minimum valid", 16, true},
		{"maximum valid", 255, true},
		{"above maximum", 256, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := CreateKeyRequest{
				Prefix:     "",
				ByteLength: tc.byteLength,
			}

			resp, err := s.CreateKey(ctx, req)

			if tc.shouldPass {
				require.NoError(t, err)
				require.NotEmpty(t, resp.Key)
			} else {
				require.Error(t, err)
				require.Empty(t, resp.Key)
				require.Contains(t, err.Error(), "byte length must be between 16 and 255")
			}
		})
	}
}
