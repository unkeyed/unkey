package ratelimit

import (
	"sync"
	"time"

	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/repeat"
)

type consistencyChecker struct {
	sync.Mutex
	// key -> peerId -> count
	counters map[string]map[string]int

	logger logging.Logger
}

func newConsistencyChecker(logger logging.Logger) *consistencyChecker {
	m := &consistencyChecker{
		counters: make(map[string]map[string]int),
		logger:   logger,
	}

	repeat.Every(time.Minute, func() {
		m.Lock()
		defer m.Unlock()

		for key, peers := range m.counters {
			if len(peers) > 1 {
				// Our hashring ensures that a single key is only ever sent to a single node for pushpull
				// In theory at least..
				m.logger.Warn().Str("key", key).Interface("peers", peers).Msg("ratelimit used multiple origins")
			}

		}
		// Reset the counters
		m.counters = make(map[string]map[string]int)
	})

	return m
}

func (m *consistencyChecker) Record(key, peerId string) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.counters[key]; !ok {
		m.counters[key] = make(map[string]int)
	}

	if _, ok := m.counters[key][peerId]; !ok {
		m.counters[key][peerId] = 0
	}
	m.counters[key][peerId]++
}
