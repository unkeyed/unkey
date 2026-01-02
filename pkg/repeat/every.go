package repeat

import (
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
)

// Every runs the given function in a go routine every d duration until the returned function is called.
func Every(d time.Duration, fn func()) func() {
	t := time.NewTicker(d)
	done := make(chan struct{})

	fnWithRecovery := func() {
		defer func() {
			if r := recover(); r != nil {
				metrics.PanicsTotal.WithLabelValues("repeat.Every", "background").Inc()
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
