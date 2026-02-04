package prompt

import (
	"fmt"
	"strconv"
	"strings"
)

// Suffix multipliers for human-readable number notation.
// These allow users to enter values like "1k" instead of "1000".
//
// Supported suffixes (case-insensitive):
//   - k, K: thousand (10^3 = 1,000)
//   - m, M: million (10^6 = 1,000,000)
//   - b, B: billion (10^9 = 1,000,000,000)
//   - t, T: trillion (10^12 = 1,000,000,000,000)
var suffixMultipliers = map[byte]int64{
	'k': 1_000,
	'K': 1_000,
	'm': 1_000_000,
	'M': 1_000_000,
	'b': 1_000_000_000,
	'B': 1_000_000_000,
	't': 1_000_000_000_000,
	'T': 1_000_000_000_000,
}

// parseHumanInt parses a human-readable integer string with optional suffix.
//
// Accepts formats:
//   - Plain integers: "42", "-100", "0"
//   - With suffix: "1k" (1000), "5m" (5000000), "-2b" (-2000000000)
//   - Decimals with suffix: "1.5k" (1500), "2.5m" (2500000)
//   - With whitespace: "  42  ", " 1k "
//
// Decimals without a suffix are rejected (e.g., "1.5" is invalid).
// Decimals with suffix must result in a whole number (e.g., "1.5k" = 1500 is valid,
// but "1.111k" = 1111.0 would need to be checked).
//
// Returns an error if:
//   - The input is empty
//   - The number format is invalid
//   - A decimal is used without a suffix
//   - The result would overflow int
func parseHumanInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty input")
	}

	// Check for suffix (last character)
	lastChar := s[len(s)-1]
	multiplier, hasSuffix := suffixMultipliers[lastChar]

	if hasSuffix {
		// Remove suffix and parse the number part
		numStr := strings.TrimSpace(s[:len(s)-1])
		if numStr == "" {
			return 0, fmt.Errorf("invalid number: missing value before suffix")
		}

		// Check if it contains a decimal point
		if strings.Contains(numStr, ".") {
			// Parse as float, multiply, then convert to int
			f, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", numStr)
			}

			result := f * float64(multiplier)

			// Check if result is a whole number (within floating point tolerance)
			rounded := float64(int64(result))
			if result-rounded > 0.0001 || rounded-result > 0.0001 {
				return 0, fmt.Errorf("result is not a whole number: %s (got %f)", s, result)
			}

			// Check for overflow
			if result > float64(maxInt) || result < float64(minInt) {
				return 0, fmt.Errorf("value out of range: %s", s)
			}

			return int(result), nil
		}

		// Parse as integer and multiply
		base, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", numStr)
		}

		result := base * multiplier

		// Check for overflow (result should have same sign as base if base != 0)
		if base > 0 && result < 0 || base < 0 && result > 0 {
			return 0, fmt.Errorf("value out of range: %s", s)
		}

		// Check int bounds
		if result > int64(maxInt) || result < int64(minInt) {
			return 0, fmt.Errorf("value out of range: %s", s)
		}

		return int(result), nil
	}

	// No suffix - reject decimals
	if strings.Contains(s, ".") {
		return 0, fmt.Errorf("decimal values require a suffix (k, m, b, t): %s", s)
	}

	// Parse as plain integer
	result, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}

	return result, nil
}

// Platform-specific int bounds.
// On 64-bit systems, int is 64 bits. On 32-bit systems, int is 32 bits.
// We use these constants to check for overflow after multiplication.
const (
	maxInt = int(^uint(0) >> 1) // Max int value for current platform
	minInt = -maxInt - 1        // Min int value for current platform
)

// parseHumanFloat parses a human-readable float string with optional suffix.
//
// Accepts formats:
//   - Plain floats: "42", "3.14", "-100.5"
//   - With suffix: "1.5k" (1500.0), "2.5m" (2500000.0)
//   - With whitespace: "  42.5  ", " 1.5k "
//
// Returns an error if:
//   - The input is empty
//   - The number format is invalid
func parseHumanFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty input")
	}

	// Check for suffix (last character)
	lastChar := s[len(s)-1]
	multiplier, hasSuffix := suffixMultipliers[lastChar]

	if hasSuffix {
		// Remove suffix and parse the number part
		numStr := strings.TrimSpace(s[:len(s)-1])
		if numStr == "" {
			return 0, fmt.Errorf("invalid number: missing value before suffix")
		}

		// Parse as float and multiply
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number: %s", numStr)
		}

		return f * float64(multiplier), nil
	}

	// No suffix - parse as plain float
	result, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return result, nil
}
