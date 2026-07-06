package ratecard

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

// The pricing vectors below are ported verbatim from
// web/internal/billing/src/tiers.test.ts so the Go engine provably matches
// the TypeScript tier semantics (expected values exact, not float-close).
func TestTieredCentsMatchesTiersTS(t *testing.T) {
	threeTiers := []Tier{
		{FirstUnit: 1, LastUnit: ptr(int64(10)), CentsPerUnit: ptr("100")},
		{FirstUnit: 11, LastUnit: ptr(int64(20)), CentsPerUnit: ptr("50")},
		{FirstUnit: 21, LastUnit: nil, CentsPerUnit: ptr("25")},
	}

	testCases := []struct {
		name     string
		tiers    []Tier
		units    int64
		expected string
	}{
		{name: "only reaches the first tier", tiers: threeTiers, units: 5, expected: "500"},
		{name: "only reaches the second tier", tiers: threeTiers, units: 15, expected: "1250"},
		{name: "reaches the third tier", tiers: threeTiers, units: 25, expected: "1625"},
		{
			name:     "single tier",
			tiers:    []Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("32.3")}},
			units:    12125,
			expected: "391637.5",
		},
		{
			name: "real world usage",
			tiers: []Tier{
				{FirstUnit: 1, LastUnit: ptr(int64(2_500)), CentsPerUnit: nil},
				{FirstUnit: 2_501, LastUnit: ptr(int64(100_000)), CentsPerUnit: ptr("0.02")},
				{FirstUnit: 100_001, LastUnit: ptr(int64(500_000)), CentsPerUnit: ptr("0.015")},
				{FirstUnit: 500_001, LastUnit: ptr(int64(1_000_000)), CentsPerUnit: ptr("0.01")},
				{FirstUnit: 1_000_001, LastUnit: nil, CentsPerUnit: ptr("0.005")},
			},
			units:    3_899_437,
			expected: "27447.185",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cents, err := TieredCents(tc.tiers, tc.units)
			require.NoError(t, err)
			require.Equal(t, tc.expected, CentsString(cents))
		})
	}
}

func TestTieredCentsZeroAndFree(t *testing.T) {
	tiers := []Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("100")}}

	cents, err := TieredCents(tiers, 0)
	require.NoError(t, err)
	require.Equal(t, "0", CentsString(cents))

	free := []Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: nil}}
	cents, err = TieredCents(free, 1_000_000)
	require.NoError(t, err)
	require.Equal(t, "0", CentsString(cents))
}

func TestValidateTiersRejectsInvalidShapes(t *testing.T) {
	testCases := []struct {
		name  string
		tiers []Tier
	}{
		{name: "empty", tiers: []Tier{}},
		{
			name: "gap between tiers",
			tiers: []Tier{
				{FirstUnit: 1, LastUnit: ptr(int64(10)), CentsPerUnit: ptr("1")},
				{FirstUnit: 12, LastUnit: nil, CentsPerUnit: ptr("1")},
			},
		},
		{
			name: "overlap between tiers",
			tiers: []Tier{
				{FirstUnit: 1, LastUnit: ptr(int64(10)), CentsPerUnit: ptr("1")},
				{FirstUnit: 10, LastUnit: nil, CentsPerUnit: ptr("1")},
			},
		},
		{
			name: "unbounded tier before the last",
			tiers: []Tier{
				{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("1")},
				{FirstUnit: 11, LastUnit: nil, CentsPerUnit: ptr("1")},
			},
		},
		{
			name:  "firstUnit below one",
			tiers: []Tier{{FirstUnit: 0, LastUnit: nil, CentsPerUnit: ptr("1")}},
		},
		{
			name:  "lastUnit below firstUnit (negative span)",
			tiers: []Tier{{FirstUnit: 10, LastUnit: ptr(int64(5)), CentsPerUnit: ptr("1")}},
		},
		{
			name:  "negative price",
			tiers: []Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("-5")}},
		},
		{
			name:  "malformed price",
			tiers: []Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("1.2.3")}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Error(t, ValidateTiers(tc.tiers))
		})
	}
}

func TestConfigPrice(t *testing.T) {
	cfg := Config{
		Verifications: []Tier{
			{FirstUnit: 1, LastUnit: ptr(int64(1000)), CentsPerUnit: nil},
			{FirstUnit: 1001, LastUnit: nil, CentsPerUnit: ptr("0.1")},
		},
		Credits:    []Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("0.01")}},
		Ratelimits: nil, // dimension not billed
	}

	amounts, err := cfg.Price(1500, 200, 999)
	require.NoError(t, err)
	// 500 over the free tier at 0.1 cents.
	require.Equal(t, "50", CentsString(amounts.VerificationsCents))
	// 200 credits at 0.01 cents.
	require.Equal(t, "2", CentsString(amounts.CreditsCents))
	// Ratelimits dimension omitted: free.
	require.Equal(t, "0", CentsString(amounts.RatelimitsCents))
	require.Equal(t, "52", CentsString(amounts.TotalCents))
}

func TestParseConfig(t *testing.T) {
	raw := []byte(`{
		"verifications": [
			{"firstUnit": 1, "lastUnit": 100, "centsPerUnit": null},
			{"firstUnit": 101, "lastUnit": null, "centsPerUnit": "0.5"}
		]
	}`)
	cfg, err := ParseConfig(raw)
	require.NoError(t, err)
	require.Len(t, cfg.Verifications, 2)
	require.Nil(t, cfg.Credits)

	_, err = ParseConfig([]byte(`{"verifications": [{"firstUnit": 0}]}`))
	require.Error(t, err)

	_, err = ParseConfig([]byte(`not json`))
	require.Error(t, err)
}

func TestCentsRendering(t *testing.T) {
	cents, err := TieredCents([]Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("0.005")}}, 3)
	require.NoError(t, err)
	require.Equal(t, "0.015", CentsString(cents))
	// Half-up rounding to whole cents.
	require.Equal(t, int64(0), RoundedCents(cents))

	cents, err = TieredCents([]Tier{{FirstUnit: 1, LastUnit: nil, CentsPerUnit: ptr("0.5")}}, 3)
	require.NoError(t, err)
	require.Equal(t, "1.5", CentsString(cents))
	require.Equal(t, int64(2), RoundedCents(cents))
}
