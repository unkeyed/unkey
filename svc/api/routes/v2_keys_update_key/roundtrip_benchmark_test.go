package handler_test

import (
	"testing"
	"time"
)

const benchmarkDatabaseRTT = 225 * time.Millisecond

type updateKeyProtocolSimulator struct {
	latency   time.Duration
	exchanges int64
}

func (s *updateKeyProtocolSimulator) exchange() {
	time.Sleep(s.latency)
	s.exchanges++
}

func BenchmarkUpdateKeyProtocolExchanges(b *testing.B) {
	scenarios := []struct {
		name       string
		operations []string
	}{
		{name: "before/simple", operations: []string{
			"key read", "begin", "key update", "audit outbox", "commit",
		}},
		{name: "after/simple", operations: []string{
			"key read", "write batch",
		}},
		{name: "before/maximal", operations: []string{
			"key read", "begin", "resource lookup", "project lookup", "project insert",
			"identity insert", "key update", "ratelimit delete", "ratelimit insert",
			"relation delete", "permission insert", "permission relation insert",
			"role relation insert", "audit outbox", "commit",
		}},
		{name: "after/maximal", operations: []string{
			"key read", "resource lookup", "write batch",
		}},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			simulator := updateKeyProtocolSimulator{latency: benchmarkDatabaseRTT}
			b.ResetTimer()
			for range b.N {
				for range scenario.operations {
					simulator.exchange()
				}
			}
			b.StopTimer()
			b.ReportMetric(float64(simulator.exchanges)/float64(b.N), "exchanges/op")
		})
	}
}
