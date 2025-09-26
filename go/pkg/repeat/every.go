package repeat

import (
	"sync"
	"time"

	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
)

// Every runs the given function in a go routine every d duration until the returned function is called.
func Every(d time.Duration, fn func()) func() {
	t := time.NewTicker(d)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							metrics.PanicsTotal.WithLabelValues("repeat.Every", "background").Inc()
						}
					}()
					fn()
				}()
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
