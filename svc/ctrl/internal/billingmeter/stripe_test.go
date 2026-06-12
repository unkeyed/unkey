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

func TestMeterEventsFor(t *testing.T) {
	const ts = int64(1_700_000_400)

	t.Run("one event per positive meter, decimal value, in fixed order", func(t *testing.T) {
		req := PushRequest{
			WorkspaceID:      "ws_x",
			StripeCustomerID: "cus_1",
			Timestamp:        ts,
			Values: MeterValues{
				CPUSeconds:       12.25,
				MemoryGiBSeconds: 9000,
				EgressGiB:        0, // skipped
				DiskGiBSeconds:   3600,
			},
		}

		events := meterEventsFor(req)
		require.Len(t, events, 3)

		require.Equal(t, eventCPU, events[0].eventName)
		require.Equal(t, "12.25", events[0].value)

		require.Equal(t, eventMemory, events[1].eventName)
		require.Equal(t, "9000", events[1].value)

		require.Equal(t, eventDisk, events[2].eventName)
		require.Equal(t, "3600", events[2].value)
	})

	t.Run("skips zero and negative meters", func(t *testing.T) {
		req := PushRequest{
			WorkspaceID:      "ws_y",
			StripeCustomerID: "cus_2",
			Timestamp:        ts,
			Values:           MeterValues{EgressGiB: 1.5},
		}
		events := meterEventsFor(req)
		require.Len(t, events, 1)
		require.Equal(t, eventEgress, events[0].eventName)
		require.Equal(t, "1.5", events[0].value)
	})

	t.Run("no positive meters yields no events", func(t *testing.T) {
		require.Empty(t, meterEventsFor(PushRequest{WorkspaceID: "ws_z", Timestamp: ts}))
	})
}

func TestFormatMeterValue(t *testing.T) {
	// Stripe rejects values with more than 12 decimal places, and full float64
	// precision overruns that.
	require.Equal(t, "10.51749382782", formatMeterValue(10.517493827819823))
	// Clean values stay clean (no trailing zeros, no trailing dot).
	require.Equal(t, "9000", formatMeterValue(9000))
	require.Equal(t, "12.25", formatMeterValue(12.25))
	require.Equal(t, "1.5", formatMeterValue(1.5))

	for _, v := range []float64{10.517493827819823, 0.123456789012345, 123456.987654321987} {
		s := formatMeterValue(v)
		if dot := strings.IndexByte(s, '.'); dot >= 0 {
			require.LessOrEqual(t, len(s)-dot-1, stripeMaxValueDecimals, "value %q exceeds 12 decimals", s)
		}
	}
}
