package prefixedapikey

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/base58"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

const (
	exampleLongToken     = "abcdefghijklmnopqrstuvwx"
	exampleLongTokenHash = "93b0cabf8668e0c534c52a568957499e12a284f59d97dc9b2725ef836804875b"
	exampleShortToken    = "12345678"
	exampleToken         = "test_12345678_abcdefghijklmnopqrstuvwx"
)

func TestHashLongToken(t *testing.T) {
	t.Run("matches an independent SHA-256 vector", func(t *testing.T) {
		expectedBytes := sha256.Sum256([]byte(exampleLongToken))

		require.Equal(t, exampleLongTokenHash, hex.EncodeToString(expectedBytes[:]))
		require.Equal(t, expectedBytes[:], hashLongTokenToBytes(exampleLongToken))
		require.Equal(t, exampleLongTokenHash, HashLongToken(exampleLongToken))
	})
}

func TestPadStart(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		padChar  string
		expected string
	}{
		{
			name:     "longer input",
			input:    "token",
			length:   3,
			padChar:  "0",
			expected: "token",
		},
		{
			name:     "equal input",
			input:    "token",
			length:   5,
			padChar:  "0",
			expected: "token",
		},
		{
			name:     "shorter input",
			input:    "key",
			length:   5,
			padChar:  "0",
			expected: "00key",
		},
		{
			name:     "empty input",
			input:    "",
			length:   3,
			padChar:  "0",
			expected: "000",
		},
		{
			name:     "empty pad character",
			input:    "key",
			length:   5,
			padChar:  "",
			expected: "key",
		},
		{
			name:     "multi-byte pad character",
			input:    "x",
			length:   3,
			padChar:  "界",
			expected: "界界x",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, padStart(test.input, test.length, test.padChar))
		})
	}
}

func TestGenerateAPIKey(t *testing.T) {
	t.Run("requires options", func(t *testing.T) {
		key, err := GenerateAPIKey(nil)

		require.ErrorIs(t, err, ErrMissingKeyPrefix)
		require.Nil(t, key)
	})

	t.Run("requires a key prefix", func(t *testing.T) {
		key, err := GenerateAPIKey(&GenerateAPIKeyOptions{})

		require.ErrorIs(t, err, ErrMissingKeyPrefix)
		require.Nil(t, key)
	})

	t.Run("uses default token lengths", func(t *testing.T) {
		key, err := GenerateAPIKey(&GenerateAPIKeyOptions{KeyPrefix: "test"})
		require.NoError(t, err)

		requireGeneratedKey(t, key, "test", 8, 24)
	})

	t.Run("uses configured token lengths and short token prefix", func(t *testing.T) {
		key, err := GenerateAPIKey(&GenerateAPIKeyOptions{
			KeyPrefix:        "api",
			ShortTokenPrefix: "dev",
			ShortTokenLength: 16,
			LongTokenLength:  32,
		})
		require.NoError(t, err)

		requireGeneratedKey(t, key, "api", 16, 32)
		require.True(t, strings.HasPrefix(key.ShortToken, "dev"))
	})

	t.Run("generates different key material", func(t *testing.T) {
		first, err := GenerateAPIKey(&GenerateAPIKeyOptions{KeyPrefix: "test"})
		require.NoError(t, err)
		second, err := GenerateAPIKey(&GenerateAPIKeyOptions{KeyPrefix: "test"})
		require.NoError(t, err)

		require.NotEqual(t, first.ShortToken, second.ShortToken)
		require.NotEqual(t, first.LongToken, second.LongToken)
		require.NotEqual(t, first.Token, second.Token)
	})
}

func TestExtractTokenParts(t *testing.T) {
	tests := []struct {
		name               string
		token              string
		expectedShortToken string
		expectedLongToken  string
	}{
		{
			name:               "generated token format",
			token:              exampleToken,
			expectedShortToken: exampleShortToken,
			expectedLongToken:  exampleLongToken,
		},
		{
			name:               "no separator",
			token:              "opaque",
			expectedShortToken: "",
			expectedLongToken:  "opaque",
		},
		{
			name:               "too few parts",
			token:              "prefix_long",
			expectedShortToken: "long",
			expectedLongToken:  "long",
		},
		{
			name:               "too many parts",
			token:              "prefix_short_extra_long",
			expectedShortToken: "short",
			expectedLongToken:  "long",
		},
		{
			name:               "empty token",
			token:              "",
			expectedShortToken: "",
			expectedLongToken:  "",
		},
		{
			name:               "prefix contains separator",
			token:              "prefix_with_short_long",
			expectedShortToken: "with",
			expectedLongToken:  "long",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expectedShortToken, ExtractShortToken(test.token))
			require.Equal(t, test.expectedLongToken, ExtractLongToken(test.token))
		})
	}
}

func TestExtractLongTokenHash(t *testing.T) {
	t.Run("hashes the extracted long token", func(t *testing.T) {
		require.Equal(t, exampleLongTokenHash, ExtractLongTokenHash(exampleToken))
	})
}

func TestGetTokenComponents(t *testing.T) {
	t.Run("returns all token components", func(t *testing.T) {
		require.Equal(t, &TokenComponents{
			LongToken:     exampleLongToken,
			ShortToken:    exampleShortToken,
			LongTokenHash: exampleLongTokenHash,
			Token:         exampleToken,
		}, GetTokenComponents(exampleToken))
	})
}

func TestCheckAPIKey(t *testing.T) {
	tests := []struct {
		name         string
		token        string
		expectedHash string
		expected     bool
	}{
		{
			name:         "matching long token hash",
			token:        exampleToken,
			expectedHash: exampleLongTokenHash,
			expected:     true,
		},
		{
			name:         "changed long token",
			token:        "test_12345678_abcdefghijklmnopqrstuvwy",
			expectedHash: exampleLongTokenHash,
			expected:     false,
		},
		{
			name:         "different hash",
			token:        exampleToken,
			expectedHash: strings.Repeat("0", sha256.Size*2),
			expected:     false,
		},
		{
			name:         "invalid hex hash",
			token:        exampleToken,
			expectedHash: "invalid_hash",
			expected:     false,
		},
		{
			name:         "short hash",
			token:        exampleToken,
			expectedHash: "00",
			expected:     false,
		},
		{
			name:         "empty token",
			token:        "",
			expectedHash: exampleLongTokenHash,
			expected:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, CheckAPIKey(test.token, test.expectedHash))
		})
	}
}

// FuzzAPIKeyParsingAndVerification guarantees that untrusted token and hash input cannot cause a panic.
func FuzzAPIKeyParsingAndVerification(f *testing.F) {
	fuzz.Seed(f)

	key, err := GenerateAPIKey(&GenerateAPIKeyOptions{KeyPrefix: "fuzz"})
	require.NoError(f, err)
	require.True(f, CheckAPIKey(key.Token, key.LongTokenHash))

	replacement := byte('1')
	if key.Token[len(key.Token)-1] == replacement {
		replacement = '2'
	}
	changedToken := key.Token[:len(key.Token)-1] + string(replacement)
	require.False(f, CheckAPIKey(changedToken, key.LongTokenHash))

	f.Fuzz(func(t *testing.T, data []byte) {
		consumer := fuzz.New(t, data)
		input := string(consumer.BytesN(consumer.Remaining()))

		require.NotPanics(t, func() {
			_ = GetTokenComponents(input)
			_ = ExtractLongToken(input)
			_ = CheckAPIKey(input, exampleLongTokenHash)
			_ = CheckAPIKey(exampleToken, input)
		})
	})
}

func requireGeneratedKey(t *testing.T, key *APIKey, keyPrefix string, shortTokenLength int, longTokenLength int) {
	t.Helper()

	require.Len(t, key.ShortToken, shortTokenLength)
	require.Len(t, key.LongToken, longTokenLength)
	requireBase58Alphabet(t, key.ShortToken)
	requireBase58Alphabet(t, key.LongToken)
	require.Equal(t, HashLongToken(key.LongToken), key.LongTokenHash)
	require.Equal(t, keyPrefix+"_"+key.ShortToken+"_"+key.LongToken, key.Token)
	require.True(t, CheckAPIKey(key.Token, key.LongTokenHash))
}

func requireBase58Alphabet(t *testing.T, token string) {
	t.Helper()

	for _, character := range token {
		require.Contains(t, base58.Alphabet, string(character))
	}
}
