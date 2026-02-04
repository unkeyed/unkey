package wide

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard email",
			input:    "john.doe@example.com",
			expected: "j*******@e******.c**",
		},
		{
			name:     "short local part",
			input:    "a@b.com",
			expected: "a@b.c**",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "no @ symbol",
			input:    "notanemail",
			expected: "notanemail",
		},
		{
			name:     "multiple @ symbols",
			input:    "bad@@email",
			expected: "bad@@email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmail(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unkey key format",
			input:    "key_abc123xyz789",
			expected: "key_***z789",
		},
		{
			name:     "stripe live key format",
			input:    "sk_live_abc123xyz789def",
			expected: "sk_live_***9def",
		},
		{
			name:     "short key",
			input:    "short",
			expected: "***",
		},
		{
			name:     "no prefix",
			input:    "abc123xyz789",
			expected: "***z789",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "email gets masked",
			input:    "john.doe@example.com",
			expected: "j*******@e******.c**",
		},
		{
			name:     "long identifier gets truncated",
			input:    "user_abc123xyz789def456",
			expected: "user_abc...f456",
		},
		{
			name:     "short identifier unchanged",
			input:    "user_123",
			expected: "user_123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIdentifier(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "long request ID",
			input:    "req_abc123xyz789def456",
			maxLen:   12,
			expected: "req_ab...456",
		},
		{
			name:     "already short enough",
			input:    "req_short",
			maxLen:   20,
			expected: "req_short",
		},
		{
			name:     "maxLen too small",
			input:    "req_abc123",
			maxLen:   4,
			expected: "req_abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateID(tt.input, tt.maxLen)
			require.Equal(t, tt.expected, result)
		})
	}
}
