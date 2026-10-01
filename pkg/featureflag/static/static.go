// Package static provides an OpenFeature provider with fixed boolean flag
// values, for local development and tests.
package static

import (
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/open-feature/go-sdk/openfeature/memprovider"
	"github.com/unkeyed/unkey/pkg/assert"
)

// Values maps flag keys to the value they return for every evaluation
// context. A flag missing from Values resolves as an error, not as false.
type Values map[string]bool

// Validate checks that at least one flag is listed.
func (v Values) Validate() error {
	return assert.True(len(v) > 0, "static feature flags must list at least one flag")
}

// New returns a provider that resolves each flag in values for every
// evaluation context. A flag missing from values resolves with FLAG_NOT_FOUND.
func New(values Values) openfeature.FeatureProvider {
	flags := make(map[string]memprovider.InMemoryFlag, len(values))
	for key, value := range values {
		variant := "off"
		if value {
			variant = "on"
		}
		flags[key] = memprovider.InMemoryFlag{
			Key:              key,
			State:            memprovider.Enabled,
			DefaultVariant:   variant,
			Variants:         map[string]any{"off": false, "on": true},
			ContextEvaluator: nil,
		}
	}
	return memprovider.NewInMemoryProvider(flags)
}
