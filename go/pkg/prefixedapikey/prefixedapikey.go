package prefixedapikey

// This Go package is a port of the https://github.com/seamapi/prefixed-api-key, licensed under MIT.
// See License for copyright and license information.

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/base58"
)

// GenerateAPIKeyOptions holds the options for generating an API key
type GenerateAPIKeyOptions struct {
	KeyPrefix        string
	ShortTokenPrefix string
	ShortTokenLength int
	LongTokenLength  int
}

// APIKey represents the generated API key components
type APIKey struct {
	ShortToken    string
	LongToken     string
	LongTokenHash string
	Token         string
}

// hashLongTokenToBytes hashes a long token using SHA256 and returns the bytes
func hashLongTokenToBytes(longToken string) []byte {
	hash := sha256.Sum256([]byte(longToken))
	return hash[:]
}

// HashLongToken hashes a long token using SHA256 and returns hex string
func HashLongToken(longToken string) string {
	return hex.EncodeToString(hashLongTokenToBytes(longToken))
}

// padStart pads a string with a character to reach the specified length
func padStart(str string, length int, padChar string) string {
	if len(str) >= length {
		return str
	}
	padding := strings.Repeat(padChar, length-len(str))
	return padding + str
}

// GenerateAPIKey generates a new API key with the given options
func GenerateAPIKey(opts *GenerateAPIKeyOptions) (*APIKey, error) {
	// Set default values if not provided
	if opts == nil {
		opts = &GenerateAPIKeyOptions{}
	}
	if opts.KeyPrefix == "" {
		return &APIKey{}, nil
	}

	if opts.ShortTokenPrefix == "" {
		opts.ShortTokenPrefix = ""
	}

	if opts.ShortTokenLength == 0 {
		opts.ShortTokenLength = 8
	}

	if opts.LongTokenLength == 0 {
		opts.LongTokenLength = 24
	}

	// Generate random bytes for tokens
	shortTokenBytes := make([]byte, opts.ShortTokenLength)
	longTokenBytes := make([]byte, opts.LongTokenLength)

	if _, err := rand.Read(shortTokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate short token: %w", err)
	}
	if _, err := rand.Read(longTokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate long token: %w", err)
	}

	// Encode tokens using base58
	shortToken := padStart(
		base58.Encode(shortTokenBytes),
		opts.ShortTokenLength,
		"0",
	)
	if len(shortToken) > opts.ShortTokenLength {
		shortToken = shortToken[:opts.ShortTokenLength]
	}

	longToken := padStart(
		base58.Encode(longTokenBytes),
		opts.LongTokenLength,
		"0",
	)
	if len(longToken) > opts.LongTokenLength {
		longToken = longToken[:opts.LongTokenLength]
	}

	// Hash the long token
	longTokenHash := HashLongToken(longToken)

	// Add prefix to short token and trim if necessary
	shortToken = (opts.ShortTokenPrefix + shortToken)
	if len(shortToken) > opts.ShortTokenLength {
		shortToken = shortToken[:opts.ShortTokenLength]
	}

	// Construct the full token
	token := fmt.Sprintf("%s_%s_%s", opts.KeyPrefix, shortToken, longToken)

	return &APIKey{
		ShortToken:    shortToken,
		LongToken:     longToken,
		LongTokenHash: longTokenHash,
		Token:         token,
	}, nil
}

// ExtractLongToken extracts the long token from a full API key
func ExtractLongToken(token string) string {
	parts := strings.Split(token, "_")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// ExtractShortToken extracts the short token from a full API key
func ExtractShortToken(token string) string {
	parts := strings.Split(token, "_")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// ExtractLongTokenHash extracts and hashes the long token from a full API key
func ExtractLongTokenHash(token string) string {
	return HashLongToken(ExtractLongToken(token))
}

// TokenComponents represents the components of an API key
type TokenComponents struct {
	LongToken     string
	ShortToken    string
	LongTokenHash string
	Token         string
}

// GetTokenComponents extracts all components from a full API key
func GetTokenComponents(token string) *TokenComponents {
	return &TokenComponents{
		LongToken:     ExtractLongToken(token),
		ShortToken:    ExtractShortToken(token),
		LongTokenHash: HashLongToken(ExtractLongToken(token)),
		Token:         token,
	}
}

// CheckAPIKey verifies if a token matches the expected long token hash using constant-time comparison
func CheckAPIKey(token string, expectedLongTokenHash string) bool {
	expectedLongTokenHashBytes, err := hex.DecodeString(expectedLongTokenHash)
	if err != nil {
		return false
	}

	inputLongTokenHashBytes := hashLongTokenToBytes(ExtractLongToken(token))

	// Use constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(expectedLongTokenHashBytes, inputLongTokenHashBytes) == 1
}
