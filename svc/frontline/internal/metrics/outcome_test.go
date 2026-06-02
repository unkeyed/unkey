package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
)

// TestUrnOutcomes_CoversAllFrontlineURNs walks codes.Frontline via
// codes.CollectURNs and asserts every URN declared there has an
// explicit entry in urnOutcomes.
//
// Adding a new code under codes.Frontline.* without classifying it here
// fails this test. The fail-safe default (OutcomeFrontlineFault) would
// still get us paged in production — this test catches the omission at
// CI time.
func TestUrnOutcomes_CoversAllFrontlineURNs(t *testing.T) {
	t.Parallel()

	for _, urn := range codes.CollectURNs(codes.Frontline) {
		_, ok := urnOutcomes[urn]
		require.Truef(t, ok,
			"frontline URN %q has no entry in urnOutcomes — add it to "+
				"svc/frontline/internal/metrics/outcome.go", urn)
	}
}

// TestOutcomeFor_UnknownURNDefaultsToFrontlineFault locks in the fail-
// safe: an unmapped URN must surface as a fault so we get paged rather
// than silently bucketed as success.
func TestOutcomeFor_UnknownURNDefaultsToFrontlineFault(t *testing.T) {
	t.Parallel()

	require.Equal(t, OutcomeFrontlineFault,
		OutcomeFor("err:made:up:nonexistent"))
}

func TestOutcomeFor_EmptyURNIsSuccess(t *testing.T) {
	t.Parallel()

	require.Equal(t, OutcomeSuccess, OutcomeFor(""))
}
