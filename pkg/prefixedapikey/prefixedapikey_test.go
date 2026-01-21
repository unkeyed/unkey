package prefixedapikey

import (
	"testing"
)

// exampleKey represents a known test key for consistent testing
var exampleKey = &APIKey{
	ShortToken:    "12345678",
	LongToken:     "abcdefghijklmnopqrstuvwx",
	LongTokenHash: HashLongToken("abcdefghijklmnopqrstuvwx"),
	Token:         "test_12345678_abcdefghijklmnopqrstuvwx",
}

func TestHashLongToken(t *testing.T) {
	result := HashLongToken(exampleKey.LongToken)
	expected := exampleKey.LongTokenHash

	if result != expected {
		t.Errorf("HashLongToken() = %v, want %v", result, expected)
	}
}

func TestExtractLongToken(t *testing.T) {
	result := ExtractLongToken(exampleKey.Token)
	expected := exampleKey.LongToken

	if result != expected {
		t.Errorf("ExtractLongToken() = %v, want %v", result, expected)
	}

	// Additional test cases
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "standard token format",
			token:    "test_12345678_abcdefghijklmnopqrstuvwx",
			expected: "abcdefghijklmnopqrstuvwx",
		},
		{
			name:     "token with multiple underscores",
			token:    "prefix_with_underscores_short_long",
			expected: "long",
		},
		{
			name:     "single underscore",
			token:    "prefix_longtoken",
			expected: "longtoken",
		},
		{
			name:     "no underscores",
			token:    "notokenstructure",
			expected: "notokenstructure",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractLongToken(tt.token)
			if result != tt.expected {
				t.Errorf("ExtractLongToken(%v) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestExtractShortToken(t *testing.T) {
	result := ExtractShortToken(exampleKey.Token)
	expected := exampleKey.ShortToken

	if result != expected {
		t.Errorf("ExtractShortToken() = %v, want %v", result, expected)
	}

	// Additional test cases
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "standard token format",
			token:    "test_12345678_abcdefghijklmnopqrstuvwx",
			expected: "12345678",
		},
		{
			name:     "token with multiple underscores",
			token:    "prefix_with_underscores_short_long",
			expected: "with",
		},
		{
			name:     "single underscore",
			token:    "prefix_shorttoken",
			expected: "shorttoken",
		},
		{
			name:     "no underscores",
			token:    "notokenstructure",
			expected: "",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "",
		},
		{
			name:     "only prefix",
			token:    "prefix_",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractShortToken(tt.token)
			if result != tt.expected {
				t.Errorf("ExtractShortToken(%v) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestGetTokenComponents(t *testing.T) {
	result := GetTokenComponents(exampleKey.Token)

	expected := &TokenComponents{
		LongToken:     exampleKey.LongToken,
		ShortToken:    exampleKey.ShortToken,
		LongTokenHash: exampleKey.LongTokenHash,
		Token:         exampleKey.Token,
	}

	if result.LongToken != expected.LongToken {
		t.Errorf("GetTokenComponents().LongToken = %v, want %v", result.LongToken, expected.LongToken)
	}

	if result.ShortToken != expected.ShortToken {
		t.Errorf("GetTokenComponents().ShortToken = %v, want %v", result.ShortToken, expected.ShortToken)
	}

	if result.LongTokenHash != expected.LongTokenHash {
		t.Errorf("GetTokenComponents().LongTokenHash = %v, want %v", result.LongTokenHash, expected.LongTokenHash)
	}

	if result.Token != expected.Token {
		t.Errorf("GetTokenComponents().Token = %v, want %v", result.Token, expected.Token)
	}
}

func TestCheckAPIKey(t *testing.T) {
	result := CheckAPIKey(exampleKey.Token, exampleKey.LongTokenHash)
	expected := true

	if result != expected {
		t.Errorf("CheckAPIKey() = %v, want %v", result, expected)
	}

	// Additional test cases
	invalidHash := "invalid_hash"
	tests := []struct {
		name         string
		token        string
		expectedHash string
		expected     bool
	}{
		{
			name:         "valid token and hash",
			token:        exampleKey.Token,
			expectedHash: exampleKey.LongTokenHash,
			expected:     true,
		},
		{
			name:         "invalid hash",
			token:        exampleKey.Token,
			expectedHash: invalidHash,
			expected:     false,
		},
		{
			name:         "empty token",
			token:        "",
			expectedHash: exampleKey.LongTokenHash,
			expected:     false,
		},
		{
			name:         "empty hash",
			token:        exampleKey.Token,
			expectedHash: "",
			expected:     false,
		},
		{
			name:         "malformed token",
			token:        "malformed_token",
			expectedHash: exampleKey.LongTokenHash,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckAPIKey(tt.token, tt.expectedHash)
			if result != tt.expected {
				t.Errorf("CheckAPIKey(%v, %v) = %v, want %v", tt.token, tt.expectedHash, result, tt.expected)
			}
		})
	}
}

func TestGenerateAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		opts     *GenerateAPIKeyOptions
		hasError bool
	}{
		{
			name: "standard generation",
			opts: &GenerateAPIKeyOptions{
				KeyPrefix:        "test",
				ShortTokenPrefix: "",
				ShortTokenLength: 8,
				LongTokenLength:  24,
			},
			hasError: false,
		},
		{
			name: "with short token prefix",
			opts: &GenerateAPIKeyOptions{
				KeyPrefix:        "api",
				ShortTokenPrefix: "dev",
				ShortTokenLength: 10,
				LongTokenLength:  32,
			},
			hasError: false,
		},
		{
			name: "minimal options",
			opts: &GenerateAPIKeyOptions{
				KeyPrefix:        "min",
				ShortTokenPrefix: "",
				ShortTokenLength: 0,
				LongTokenLength:  0,
			},
			hasError: false,
		},
		{
			name:     "nil options returns error",
			opts:     nil,
			hasError: true,
		},
		{
			name: "empty key prefix returns error",
			opts: &GenerateAPIKeyOptions{
				KeyPrefix:        "",
				ShortTokenPrefix: "",
				ShortTokenLength: 0,
				LongTokenLength:  0,
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateAPIKey(tt.opts)

			if tt.hasError {
				if err == nil {
					t.Errorf("GenerateAPIKey() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateAPIKey() unexpected error: %v", err)
				return
			}

			// Validate the generated key structure
			if result.Token == "" {
				t.Errorf("GenerateAPIKey() generated empty token")
			}

			if result.ShortToken == "" {
				t.Errorf("GenerateAPIKey() generated empty short token")
			}

			if result.LongToken == "" {
				t.Errorf("GenerateAPIKey() generated empty long token")
			}

			if result.LongTokenHash == "" {
				t.Errorf("GenerateAPIKey() generated empty long token hash")
			}

			// Verify the hash matches
			expectedHash := HashLongToken(result.LongToken)
			if result.LongTokenHash != expectedHash {
				t.Errorf("GenerateAPIKey() hash mismatch: got %v, want %v", result.LongTokenHash, expectedHash)
			}

			// Verify token structure
			components := GetTokenComponents(result.Token)
			if components.LongToken != result.LongToken {
				t.Errorf("GenerateAPIKey() token structure invalid: long token mismatch")
			}

			if components.ShortToken != result.ShortToken {
				t.Errorf("GenerateAPIKey() token structure invalid: short token mismatch")
			}

			// Verify token can be validated
			if !CheckAPIKey(result.Token, result.LongTokenHash) {
				t.Errorf("GenerateAPIKey() generated token fails validation")
			}
		})
	}
}

func TestGenerateAPIKeyConsistency(t *testing.T) {
	opts := &GenerateAPIKeyOptions{
		KeyPrefix:        "test",
		ShortTokenLength: 8,
		LongTokenLength:  24,
	}

	// Generate multiple keys to ensure they're different
	keys := make([]*APIKey, 10)
	for i := 0; i < 10; i++ {
		key, err := GenerateAPIKey(opts)
		if err != nil {
			t.Fatalf("GenerateAPIKey() unexpected error: %v", err)
		}
		keys[i] = key
	}

	// Ensure all generated keys are unique
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i].Token == keys[j].Token {
				t.Errorf("GenerateAPIKey() generated duplicate tokens: %v", keys[i].Token)
			}
			if keys[i].LongToken == keys[j].LongToken {
				t.Errorf("GenerateAPIKey() generated duplicate long tokens")
			}
			if keys[i].ShortToken == keys[j].ShortToken {
				t.Errorf("GenerateAPIKey() generated duplicate short tokens")
			}
		}
	}
}

func TestExtractLongTokenHash(t *testing.T) {
	token := "test_12345678_abcdefghijklmnopqrstuvwx"

	result := ExtractLongTokenHash(token)
	expected := HashLongToken("abcdefghijklmnopqrstuvwx")

	if result != expected {
		t.Errorf("ExtractLongTokenHash(%v) = %v, want %v", token, result, expected)
	}
}
