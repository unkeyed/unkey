package billingmeter

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMeterValuesPositive(t *testing.T) {
	require.False(t, MeterValues{}.Positive())
	require.True(t, MeterValues{CPUSeconds: 1}.Positive())
	require.True(t, MeterValues{EgressGiB: 0.5}.Positive())
	require.False(t, MeterValues{CPUSeconds: -1}.Positive())
}

func TestFormatMeterValue(t *testing.T) {
	// Clean values stay clean (no trailing zeros, no trailing dot).
	require.Equal(t, "9000", formatMeterValue(9000))
	require.Equal(t, "12.25", formatMeterValue(12.25))
	require.Equal(t, "1.5", formatMeterValue(1.5))

	// A large month-to-date total is bounded by the 15-digit budget (7 integer
	// + 8 decimal here). This broke a fixed 12-decimal cap: 7 + 12 = 19 digits.
	require.Equal(t, "1781907.271686", formatMeterValue(1781907.271685999818))
	// A small-integer value is bounded by the 12-decimal cap, not the 15-digit
	// budget. This broke dropping the 12-decimal cap: 2 + 13 decimals > 12.
	require.Equal(t, "43.899664402008", formatMeterValue(43.8996644020081))
	// A sub-1 value keeps the full 12 decimals (1 integer digit, 12 < 14).
	require.Equal(t, "0.000006944", formatMeterValue(0.000006944))

	// Whatever the magnitude, neither Stripe limit is exceeded: at most 15
	// total digits and at most 12 decimal places.
	for _, v := range []float64{
		10.517493827819823, 0.123456789012345, 123456.987654321987,
		1781907.271685999818, 43.8996644020081, 2627999999.999999, 0.000000123456789,
	} {
		s := formatMeterValue(v)
		digits := len(strings.NewReplacer(".", "", "-", "").Replace(s))
		require.LessOrEqual(t, digits, stripeMaxValueDigits, "value %q exceeds %d digits", s, stripeMaxValueDigits)
		if dot := strings.IndexByte(s, '.'); dot >= 0 {
			require.LessOrEqual(t, len(s)-dot-1, stripeMaxValueDecimals, "value %q exceeds %d decimals", s, stripeMaxValueDecimals)
		}
	}
}
