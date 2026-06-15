package checkpoints

import (
	"fmt"
	"strconv"
	"strings"
)

// byteUnits maps size suffixes to their multiplier, longest first so two-letter
// IEC suffixes (e.g. "Mi") match before the one-letter SI suffix ("M"). IEC
// suffixes are powers of 1024; SI suffixes are powers of 1000.
var byteUnits = []struct {
	suffix string
	mult   int64
}{
	{"Ki", 1 << 10}, {"Mi", 1 << 20}, {"Gi", 1 << 30}, {"Ti", 1 << 40},
	{"K", 1_000}, {"M", 1_000_000}, {"G", 1_000_000_000}, {"T", 1_000_000_000_000},
	{"B", 1},
}

// parseBytes parses a human byte size: a bare number (bytes) or a number with
// an IEC (Ki/Mi/Gi/Ti) or SI (K/M/G/T/B) suffix. Empty string is 0.
func parseBytes(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	mult := int64(1)
	for _, u := range byteUnits {
		if strings.HasSuffix(s, u.suffix) {
			mult = u.mult
			s = strings.TrimSuffix(s, u.suffix)
			break
		}
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("not a byte size: %w", err)
	}

	return int64(v * float64(mult)), nil
}
