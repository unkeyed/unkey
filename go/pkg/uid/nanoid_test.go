package uid

import (
	"strings"
	"testing"
)

// TestNano verifies that Nano generates strings with correct format and length.
func TestNano(t *testing.T) {
	tests := []struct {
		name    string
		length  []int
		wantLen int
	}{
		{
			name:    "default length",
			length:  nil,
			wantLen: 8, // 8 default chars
		},
		{
			name:    "custom length 12",
			length:  []int{12},
			wantLen: 12,
		},
		{
			name:    "custom length 5",
			length:  []int{5},
			wantLen: 5,
		},
		{
			name:    "zero length",
			length:  []int{0},
			wantLen: 0,
		},
		{
			name:    "large length",
			length:  []int{32},
			wantLen: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := Nano(tt.length...)

			// Check total length
			if len(id) != tt.wantLen {
				t.Errorf("Nano() length = %v, want %v", len(id), tt.wantLen)
			}

			// Check that all characters are valid
			for _, char := range id {
				if !strings.ContainsRune(nanoAlphabet, char) {
					t.Errorf("Nano() contains invalid character: %c", char)
				}
			}
		})
	}
}

// TestNanoUniqueness verifies that consecutive calls produce different strings.
// This is a basic smoke test - not a guarantee of randomness quality.
func TestNanoUniqueness(t *testing.T) {
	const iterations = 100
	seen := make(map[string]bool)

	for range iterations {
		id := Nano() // Using default length
		if seen[id] {
			t.Errorf("Duplicate ID generated: %v", id)
		}
		seen[id] = true
	}
}

// TestNanoWithPrefix demonstrates how to use Nano with manual prefixing.
func TestNanoWithPrefix(t *testing.T) {
	// Example of using Nano with a manual prefix
	prefix := "usr_"
	id := prefix + Nano()

	if !strings.HasPrefix(id, prefix) {
		t.Errorf("Expected ID to start with prefix %v, got %v", prefix, id)
	}

	// Total length should be prefix + 8 default chars
	expectedLen := len(prefix) + 8
	if len(id) != expectedLen {
		t.Errorf("Expected total length %v, got %v", expectedLen, len(id))
	}
}
