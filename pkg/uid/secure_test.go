package uid

import (
	"bytes"
	"testing"
)

// TestSecureRejectsBiasedBytes verifies rejection sampling ignores bytes that
// would create modulo bias instead of mapping them into the alphabet. With a
// 62-character alphabet, byte values 248..255 would wrap around to the first 8
// characters and make them more likely. It calls the private reader-backed
// helper so the test can inject those bytes directly without replacing the
// global crypto/rand.Reader used by Secure.
func TestSecureRejectsBiasedBytes(t *testing.T) {
	reader := bytes.NewReader([]byte{
		0, 7, 8, 61, 62, 247, 248, 249, 255,
		1, 2, 3, 4, 5, 6, 7, 8, 9,
	})

	id := secure(9, reader)

	if id != "ahi9a9bcd" {
		t.Errorf("secure() = %q, want %q", id, "ahi9a9bcd")
	}
}

// TestSecureUniformDistributionAcrossLargeSample generates many Secure tokens
// and verifies no alphabet character is meaningfully over- or under-represented.
// This is a practical guard against accidentally reintroducing byte % alphabet
// mapping, where the first few characters would appear more often over time.
func TestSecureUniformDistributionAcrossLargeSample(t *testing.T) {
	const (
		tokens      = 10_000
		tokenLength = 24
		tolerance   = 0.20
	)

	counts := make(map[byte]int, len(defaultAlphabet))
	for range tokens {
		id := Secure(tokenLength)
		for i := range id {
			counts[id[i]]++
		}
	}

	sampleSize := tokens * tokenLength
	wantCount := float64(sampleSize) / float64(len(defaultAlphabet))
	minCount := int(wantCount * (1 - tolerance))
	maxCount := int(wantCount * (1 + tolerance))
	for i := range defaultAlphabet {
		char := defaultAlphabet[i]
		if counts[char] < minCount || counts[char] > maxCount {
			t.Errorf("Secure() generated %q %d times, want between %d and %d", char, counts[char], minCount, maxCount)
		}
	}
}
