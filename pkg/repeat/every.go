package repeat

import (
	"sync"
	"time"
)

// Every runs the given function in a go routine every d duration until the returned function is called.
func Every(d time.Duration, m Metrics, fn func()) func() {
	if m == nil {
		m = NoopMetrics{}
	}

	t := time.NewTicker(d)
	done := make(chan struct{})

	fnWithRecovery := func() {
		defer func() {
			if r := recover(); r != nil {
				m.RecordPanic("repeat.Every", "background")
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
