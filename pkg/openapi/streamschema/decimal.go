package streamschema

import (
	"bytes"
	"cmp"
	"math"
)

// exponentAbsMax bounds the magnitude of a number's literal exponent. Larger
// exponents are rejected instead of approximated; the bound keeps every
// normalized exponent far from int64 overflow.
const exponentAbsMax = 1_000_000_000

// decimal is an exact JSON number: (-1)^negative × 0.digits × 10^exponent.
// digits holds the significant digits with no leading or trailing zeros, and is
// empty for zero, in which case negative is false and exponent is 0. Because the
// form is normalized, two decimals compare by sign, then exponent, then digits.
type decimal struct {
	negative bool
	digits   []byte
	exponent int64
}

type decimalStatus int

const (
	decimalOK decimalStatus = iota
	decimalSyntax
	decimalExponentLimit
)

// parseDecimal parses token as an RFC 8259 number into dst, reusing dst.digits.
// It reports decimalExponentLimit when the literal exponent exceeds
// exponentAbsMax in magnitude.
func parseDecimal(token []byte, dst *decimal) decimalStatus {
	dst.negative = false
	dst.digits = dst.digits[:0]
	dst.exponent = 0

	i := 0
	negative := false
	if i < len(token) && token[i] == '-' {
		negative = true
		i++
	}

	integerStart := i
	switch {
	case i < len(token) && token[i] == '0':
		i++
	case i < len(token) && isDigit(token[i]):
		for i < len(token) && isDigit(token[i]) {
			i++
		}
	default:
		return decimalSyntax
	}
	integer := token[integerStart:i]

	var fraction []byte
	if i < len(token) && token[i] == '.' {
		i++
		fractionStart := i
		for i < len(token) && isDigit(token[i]) {
			i++
		}
		if i == fractionStart {
			return decimalSyntax
		}
		fraction = token[fractionStart:i]
	}

	exponent := int64(0)
	exponentExceeded := false
	if i < len(token) && (token[i] == 'e' || token[i] == 'E') {
		i++
		exponentNegative := false
		if i < len(token) && (token[i] == '+' || token[i] == '-') {
			exponentNegative = token[i] == '-'
			i++
		}
		exponentStart := i
		for i < len(token) && isDigit(token[i]) {
			if !exponentExceeded {
				exponent = exponent*10 + int64(token[i]-'0')
				exponentExceeded = exponent > exponentAbsMax
			}
			i++
		}
		if i == exponentStart {
			return decimalSyntax
		}
		if exponentNegative {
			exponent = -exponent
		}
	}

	if i != len(token) {
		return decimalSyntax
	}
	if exponentExceeded {
		return decimalExponentLimit
	}

	// point is the position of the decimal point relative to the first digit
	// kept, so the value is 0.digits × 10^(point+exponent).
	point := int64(len(integer))
	for _, part := range [2][]byte{integer, fraction} {
		for _, digit := range part {
			if len(dst.digits) == 0 && digit == '0' {
				point--
				continue
			}
			dst.digits = append(dst.digits, digit)
		}
	}
	dst.digits = bytes.TrimRight(dst.digits, "0")
	if len(dst.digits) == 0 {
		return decimalOK
	}

	dst.negative = negative
	dst.exponent = point + exponent
	return decimalOK
}

func (d *decimal) compare(other *decimal) int {
	signD, signOther := d.sign(), other.sign()
	if signD != signOther {
		return cmp.Compare(signD, signOther)
	}
	if signD == 0 {
		return 0
	}

	magnitude := cmp.Compare(d.exponent, other.exponent)
	if magnitude == 0 {
		magnitude = bytes.Compare(d.digits, other.digits)
	}
	if d.negative {
		return -magnitude
	}
	return magnitude
}

func (d *decimal) sign() int {
	switch {
	case len(d.digits) == 0:
		return 0
	case d.negative:
		return -1
	default:
		return 1
	}
}

// isInteger reports whether d has no fractional part, so 1.0 and 1e0 are
// integers as JSON Schema requires.
func (d *decimal) isInteger() bool {
	return len(d.digits) == 0 || d.exponent >= int64(len(d.digits))
}

// int64Saturated returns d as an int64 when d is a non-negative integer,
// saturating at math.MaxInt64. Saturation is exact for comparisons with counts
// of bytes, code points, or items, none of which can reach math.MaxInt64.
func (d *decimal) int64Saturated() (int64, bool) {
	if d.negative || !d.isInteger() {
		return 0, false
	}
	if d.exponent > 19 {
		return math.MaxInt64, true
	}

	value := int64(0)
	for i := range d.exponent {
		digit := int64(0)
		if i < int64(len(d.digits)) {
			digit = int64(d.digits[i] - '0')
		}
		if value > (math.MaxInt64-digit)/10 {
			return math.MaxInt64, true
		}
		value = value*10 + digit
	}
	return value, true
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
