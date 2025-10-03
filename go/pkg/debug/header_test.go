package debug

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheHeader_String(t *testing.T) {
	tests := []struct {
		name     string
		header   CacheHeader
		expected string
	}{
		{
			name: "millisecond duration",
			header: CacheHeader{
				CacheName: "api_by_id",
				Latency:   1500 * time.Microsecond, // 1.5ms
				Status:    "FRESH",
			},
			expected: "api_by_id:1.50ms:FRESH",
		},
		{
			name: "microsecond duration",
			header: CacheHeader{
				CacheName: "verification_key_by_hash",
				Latency:   750 * time.Microsecond, // 750us
				Status:    "MISS",
			},
			expected: "verification_key_by_hash:750us:MISS",
		},
		{
			name: "large millisecond duration",
			header: CacheHeader{
				CacheName: "permissions_by_api_id",
				Latency:   23080 * time.Microsecond, // 23.08ms
				Status:    "STALE",
			},
			expected: "permissions_by_api_id:23.08ms:STALE",
		},
		{
			name: "error status",
			header: CacheHeader{
				CacheName: "test_cache",
				Latency:   100 * time.Microsecond,
				Status:    "ERROR",
			},
			expected: "test_cache:100us:ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.header.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      CacheHeader
		expectError   bool
		errorContains string
	}{
		{
			name:  "valid millisecond header",
			input: "api_by_id:1.50ms:FRESH",
			expected: CacheHeader{
				CacheName: "api_by_id",
				Latency:   1500 * time.Microsecond,
				Status:    "FRESH",
			},
		},
		{
			name:  "valid microsecond header",
			input: "verification_key_by_hash:750us:MISS",
			expected: CacheHeader{
				CacheName: "verification_key_by_hash",
				Latency:   750 * time.Microsecond,
				Status:    "MISS",
			},
		},
		{
			name:  "decimal milliseconds",
			input: "permissions_by_api_id:23.08ms:STALE",
			expected: CacheHeader{
				CacheName: "permissions_by_api_id",
				Latency:   23080 * time.Microsecond,
				Status:    "STALE",
			},
		},
		{
			name:  "integer milliseconds",
			input: "test_cache:5ms:ERROR",
			expected: CacheHeader{
				CacheName: "test_cache",
				Latency:   5 * time.Millisecond,
				Status:    "ERROR",
			},
		},
		{
			name:          "invalid format - too few parts",
			input:         "cache_name:1ms",
			expectError:   true,
			errorContains: "expected 3 parts",
		},
		{
			name:          "invalid format - too many parts",
			input:         "cache:name:1ms:FRESH:extra",
			expectError:   true,
			errorContains: "expected 3 parts",
		},
		{
			name:          "invalid latency format",
			input:         "cache_name:invalid:FRESH",
			expectError:   true,
			errorContains: "failed to parse latency",
		},
		{
			name:          "unsupported duration unit",
			input:         "cache_name:1s:FRESH",
			expectError:   true,
			errorContains: "unsupported duration format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseCacheHeader(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCacheHeader_Equal(t *testing.T) {
	header1 := CacheHeader{
		CacheName: "api_by_id",
		Latency:   1500 * time.Microsecond,
		Status:    "FRESH",
	}

	header2 := CacheHeader{
		CacheName: "api_by_id",
		Latency:   1500 * time.Microsecond,
		Status:    "FRESH",
	}

	header3 := CacheHeader{
		CacheName: "different_cache",
		Latency:   1500 * time.Microsecond,
		Status:    "FRESH",
	}

	// Test equality
	assert.Equal(t, header1, header2)

	// Test inequality
	assert.NotEqual(t, header1, header3)
}

func TestNewCacheHeader(t *testing.T) {
	header := NewCacheHeader("test_cache", "MISS", 2*time.Millisecond)

	assert.Equal(t, "test_cache", header.CacheName)
	assert.Equal(t, 2*time.Millisecond, header.Latency)
	assert.Equal(t, "MISS", header.Status)
}

func TestCacheHeader_StringMethod(t *testing.T) {
	header := CacheHeader{
		CacheName: "api_by_id",
		Latency:   1500 * time.Microsecond,
		Status:    "FRESH",
	}

	// Test that String() method works correctly
	result := header.String()
	assert.Equal(t, "api_by_id:1.50ms:FRESH", result)
}

func TestRoundTripConsistency(t *testing.T) {
	// Test that String -> ParseCacheHeader -> String produces consistent results
	original := CacheHeader{
		CacheName: "verification_key_by_hash",
		Latency:   23080 * time.Microsecond, // This caused the original test failure
		Status:    "MISS",
	}

	// Convert to string
	headerString := original.String()

	// Parse back from string
	parsed, err := ParseCacheHeader(headerString)
	require.NoError(t, err)

	// Should be equal to original
	assert.Equal(t, original, parsed)

	// Converting back to string should produce the same result
	assert.Equal(t, headerString, parsed.String())
}
