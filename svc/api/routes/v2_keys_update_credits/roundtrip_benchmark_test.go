package handler_test

import (
	"context"
	"testing"
	"time"
)

const simulatedDatabaseRTT = 225 * time.Millisecond

type latencyProtocol struct {
	delay     time.Duration
	exchanges int
}

func (p *latencyProtocol) exchange(ctx context.Context) error {
	p.exchanges++
	timer := time.NewTimer(p.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// BenchmarkUpdateCreditsProtocolExchanges models only database protocol time.
// Run with: go test ./svc/api/routes/v2_keys_update_credits -run '^$' -bench '^BenchmarkUpdateCreditsProtocolExchanges$' -benchtime=1x
func BenchmarkUpdateCreditsProtocolExchanges(b *testing.B) {
	cases := []struct {
		name            string
		beforeExchanges int
		afterExchanges  int
	}{
		{name: "set_finite", beforeExchanges: 6, afterExchanges: 2},
		{name: "increment", beforeExchanges: 6, afterExchanges: 2},
		{name: "decrement", beforeExchanges: 6, afterExchanges: 2},
		{name: "set_unlimited", beforeExchanges: 7, afterExchanges: 2},
		{name: "overflow", beforeExchanges: 4, afterExchanges: 2},
	}

	for _, test := range cases {
		for _, path := range []struct {
			name      string
			exchanges int
		}{
			{name: "before", exchanges: test.beforeExchanges},
			{name: "after", exchanges: test.afterExchanges},
		} {
			b.Run(test.name+"/"+path.name, func(b *testing.B) {
				protocol := &latencyProtocol{delay: simulatedDatabaseRTT}
				b.ResetTimer()
				for range b.N {
					for range path.exchanges {
						if err := protocol.exchange(b.Context()); err != nil {
							b.Fatal(err)
						}
					}
				}
				b.StopTimer()
				if protocol.exchanges != b.N*path.exchanges {
					b.Fatalf("got %d exchanges, want %d", protocol.exchanges, b.N*path.exchanges)
				}
				b.ReportMetric(float64(path.exchanges), "exchanges/op")
			})
		}
	}
}
