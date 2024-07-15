package ratelimit

import (
	"sync"
)

type metrics struct {
	sync.Mutex
	// key -> peerId -> count
	counters map[string]map[string]int
}

func newMetrics() *metrics {
	return &metrics{
		counters: make(map[string]map[string]int),
	}
}

func (m *metrics) Record(key, peerId string) {
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

func (m *metrics) ReadAndReset() map[string]map[string]int {
	m.Lock()
	defer m.Unlock()

	cpy := make(map[string]map[string]int)

	for key, peers := range m.counters {
		cpy[key] = make(map[string]int)
		for peer, count := range peers {
			cpy[key][peer] = count
		}
	}

	m.counters = make(map[string]map[string]int)

	return cpy
}
