// Package ratecard prices per-identity billable quantities against a
// customer-defined rate card. It is the Go twin of the tier model in
// web/internal/billing (tiers.ts): the same tier shape and allocation
// semantics, but computed with exact rational arithmetic — tiers.ts itself
// warns its float math is a display estimate and must not be used for
// billing. This package is the single amount-computation path shared by the
// billing export API and the period-close push, so both produce identical
// money (R19).
package ratecard

import (
	"encoding/json"
	"math/big"
	"regexp"

	"github.com/unkeyed/unkey/pkg/fault"
)

// centsPerUnitRE mirrors the billingTier zod schema: 1-15 integer digits,
// optionally a dot and 1-12 fractional digits.
var centsPerUnitRE = regexp.MustCompile(`^\d{1,15}(\.\d{1,12})?$`)

// Tier is one tiered-price step: units FirstUnit..LastUnit cost CentsPerUnit
// each. LastUnit nil means unbounded (only allowed on the final tier);
// CentsPerUnit nil means free. JSON tags match the RateCardConfig shape
// stored in the rate_cards.config column.
type Tier struct {
	FirstUnit    int64   `json:"firstUnit"`
	LastUnit     *int64  `json:"lastUnit"`
	CentsPerUnit *string `json:"centsPerUnit"`
}

// Config is a rate card's tiered prices per metered dimension. Omitted
// dimensions are not billed.
type Config struct {
	Verifications []Tier `json:"verifications,omitempty"`
	Credits       []Tier `json:"credits,omitempty"`
	Ratelimits    []Tier `json:"ratelimits,omitempty"`
}

// ParseConfig decodes a rate_cards.config JSON blob and validates every
// dimension's tiers.
func ParseConfig(raw []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fault.Wrap(err, fault.Internal("invalid rate card config json"))
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks every present dimension's tiers.
func (c Config) Validate() error {
	for name, tiers := range map[string][]Tier{
		"verifications": c.Verifications,
		"credits":       c.Credits,
		"ratelimits":    c.Ratelimits,
	} {
		if tiers == nil {
			continue
		}
		if err := ValidateTiers(tiers); err != nil {
			return fault.Wrap(err, fault.Internal("invalid tiers for dimension "+name))
		}
	}
	return nil
}

// ValidateTiers enforces the same rules as tiers.ts: at least one tier,
// firstUnit >= 1, a valid cents string when priced, every tier except the
// last bounded, and consecutive tiers contiguous (no gaps, no overlaps).
func ValidateTiers(tiers []Tier) error {
	if len(tiers) == 0 {
		return fault.New("at least one tier is required")
	}
	for i, tier := range tiers {
		if tier.FirstUnit < 1 {
			return fault.New("firstUnit must be >= 1")
		}
		if tier.LastUnit != nil && *tier.LastUnit < 1 {
			return fault.New("lastUnit must be >= 1 or null")
		}
		if tier.LastUnit != nil && *tier.LastUnit < tier.FirstUnit {
			return fault.New("lastUnit must be >= firstUnit")
		}
		if tier.CentsPerUnit != nil && !centsPerUnitRE.MatchString(*tier.CentsPerUnit) {
			return fault.New("centsPerUnit must be a decimal string or null")
		}
		if i == 0 {
			continue
		}
		previous := tiers[i-1]
		if previous.LastUnit == nil {
			return fault.New("every tier except the last one must have a lastUnit")
		}
		if tier.FirstUnit > *previous.LastUnit+1 {
			return fault.New("there is a gap between tiers")
		}
		if tier.FirstUnit < *previous.LastUnit+1 {
			return fault.New("there is an overlap between tiers")
		}
	}
	return nil
}

// TieredCents prices units against tiers with exact rational arithmetic,
// using the same greedy allocation as tiers.ts calculateTieredPrices.
// The result is in cents.
func TieredCents(tiers []Tier, units int64) (*big.Rat, error) {
	if err := ValidateTiers(tiers); err != nil {
		return nil, err
	}

	total := new(big.Rat)
	if units <= 0 {
		return total, nil
	}

	remaining := units
	for _, tier := range tiers {
		if remaining <= 0 {
			break
		}
		quantity := remaining
		if tier.LastUnit != nil {
			span := *tier.LastUnit - tier.FirstUnit + 1
			if span < quantity {
				quantity = span
			}
		}
		remaining -= quantity
		if tier.CentsPerUnit == nil {
			continue
		}
		price, ok := new(big.Rat).SetString(*tier.CentsPerUnit)
		if !ok {
			return nil, fault.New("centsPerUnit is not a valid decimal")
		}
		total.Add(total, price.Mul(price, new(big.Rat).SetInt64(quantity)))
	}

	return total, nil
}

// Amounts is the priced result for one identity and period, one exact cents
// amount per metered dimension plus the total. A nil dimension in the config
// contributes zero.
type Amounts struct {
	VerificationsCents *big.Rat
	CreditsCents       *big.Rat
	RatelimitsCents    *big.Rat
	TotalCents         *big.Rat
}

// Price computes the exact per-dimension and total cents for the given
// quantities under this config.
func (c Config) Price(verifications, credits, ratelimits int64) (Amounts, error) {
	amounts := Amounts{
		VerificationsCents: new(big.Rat),
		CreditsCents:       new(big.Rat),
		RatelimitsCents:    new(big.Rat),
		TotalCents:         new(big.Rat),
	}

	for _, dimension := range []struct {
		tiers  []Tier
		units  int64
		target *big.Rat
	}{
		{tiers: c.Verifications, units: verifications, target: amounts.VerificationsCents},
		{tiers: c.Credits, units: credits, target: amounts.CreditsCents},
		{tiers: c.Ratelimits, units: ratelimits, target: amounts.RatelimitsCents},
	} {
		if dimension.tiers == nil {
			continue
		}
		cents, err := TieredCents(dimension.tiers, dimension.units)
		if err != nil {
			return Amounts{}, err
		}
		dimension.target.Set(cents)
		amounts.TotalCents.Add(amounts.TotalCents, cents)
	}

	return amounts, nil
}

// CentsString renders an exact cents amount as a decimal string with
// trailing zeros trimmed (e.g. "27447.185", "500").
func CentsString(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}
	// 12 fractional digits covers the maximum precision of centsPerUnit
	// inputs, so FloatString is exact here, then trim.
	s := r.FloatString(12)
	i := len(s) - 1
	for s[i] == '0' {
		i--
	}
	if s[i] == '.' {
		i--
	}
	return s[:i+1]
}

// RoundedCents rounds an exact cents amount half-up to whole cents, for
// providers that require integer cents.
func RoundedCents(r *big.Rat) int64 {
	half := big.NewRat(1, 2)
	shifted := new(big.Rat).Add(r, half)
	quotient := new(big.Int).Quo(shifted.Num(), shifted.Denom())
	return quotient.Int64()
}
