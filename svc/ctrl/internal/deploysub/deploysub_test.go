package deploysub

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ids(keys ...string) map[string]struct{} {
	m := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		m[k] = struct{}{}
	}
	return m
}

// TestClassify locks in the cancel decision: which Stripe mutation each
// subscription shape maps to. Getting this wrong mutates billing incorrectly,
// so it is tested in isolation from the Stripe client.
func TestClassify(t *testing.T) {
	planFee := ids("price_starter")
	allDeploy := ids("price_starter", "price_cpu", "price_mem")

	t.Run("no deploy items is already-cancelled (none)", func(t *testing.T) {
		topology, planFeeItemIDs := classify([]itemPrice{
			{itemID: "si_api", priceID: "price_api"},
		}, planFee, allDeploy)
		require.Equal(t, TopologyNone, topology)
		require.Empty(t, planFeeItemIDs)
	})

	t.Run("empty subscription is none", func(t *testing.T) {
		topology, planFeeItemIDs := classify(nil, planFee, allDeploy)
		require.Equal(t, TopologyNone, topology)
		require.Empty(t, planFeeItemIDs)
	})

	t.Run("all deploy items is deploy-only", func(t *testing.T) {
		topology, planFeeItemIDs := classify([]itemPrice{
			{itemID: "si_fee", priceID: "price_starter"},
			{itemID: "si_cpu", priceID: "price_cpu"},
			{itemID: "si_mem", priceID: "price_mem"},
		}, planFee, allDeploy)
		require.Equal(t, TopologyDeployOnly, topology)
		// Deploy-only cancels at period end; no per-item removal.
		require.Empty(t, planFeeItemIDs)
	})

	t.Run("deploy plus api is mixed and removes only the plan-fee item", func(t *testing.T) {
		topology, planFeeItemIDs := classify([]itemPrice{
			{itemID: "si_api", priceID: "price_api"},
			{itemID: "si_fee", priceID: "price_starter"},
			{itemID: "si_cpu", priceID: "price_cpu"},
			{itemID: "si_mem", priceID: "price_mem"},
		}, planFee, allDeploy)
		require.Equal(t, TopologyMixed, topology)
		// Only the plan-fee item is removed; metered items stay to bill the
		// frozen usage at the boundary.
		require.Equal(t, []string{"si_fee"}, planFeeItemIDs)
	})
}
