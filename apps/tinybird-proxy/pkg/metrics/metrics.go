package metrics

import (
	"sync/atomic"
	"time"
)

type Record struct {
	NodeId   string `json:"nodeId"`
	Time     int64  `json:"time"`
	Rows     int64  `json:"rows"`
	Requests int64  `json:"requests"`
	Flushes  int64  `json:"flushes"`
}

type Metrics struct {
	nodeId    string
	rows      atomic.Int64
	flushes   atomic.Int64
	requests  atomic.Int64
	lastFlush time.Time
}

func New(nodeId string) *Metrics {
	return &Metrics{
		nodeId:    nodeId,
		flushes:   atomic.Int64{},
		rows:      atomic.Int64{},
		requests:  atomic.Int64{},
		lastFlush: time.Now(),
	}
}

func (m *Metrics) RecordRequest() {
	m.requests.Add(1)
}

func (m *Metrics) RecordRows(rows int64) {
	m.rows.Add(rows)
}

func (m *Metrics) RecordFlush() {
	m.flushes.Add(1)
}

func (m *Metrics) PeriodicallyFlush(flush func(record Record)) {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		flush(Record{
			NodeId:   m.nodeId,
			Time:     time.Now().UnixMilli(),
			Rows:     m.rows.Swap(0),
			Requests: m.requests.Swap(0),
			Flushes:  m.flushes.Swap(0),
		})

	}
}
