package repeat

import (
	"sync"
	"time"

	panicmetrics "github.com/unkeyed/unkey/pkg/prometheus/metrics"
)

// Every runs the given function in a go routine every d duration until the returned function is called.
// If panic metrics are provided, panics will be counted; pass nil to skip panic counting.
func Every(d time.Duration, fn func(), m *panicmetrics.Metrics) func() {
	t := time.NewTicker(d)
	done := make(chan struct{})

	fnWithRecovery := func() {
		defer func() {
			if r := recover(); r != nil {
				if m != nil {
					m.PanicsTotal.WithLabelValues("repeat.Every", "background").Inc()
				}
			}
		}()
		fn()
	}

	go func() {
		fnWithRecovery()
		for {
			select {
			case <-t.C:
				fnWithRecovery()

			case <-done:
				return
			}
		}
	}()

	return sync.OnceFunc(func() {
		t.Stop()
		close(done)
	})
}
