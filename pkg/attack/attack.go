package attack

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type Rate struct {
	Freq int
	Per  time.Duration
}

func (r Rate) String() string {
	return fmt.Sprintf("%d per %s", r.Freq, r.Per)
}

// Attack executes the given function at the given rate for the given duration
// and returns a channel on which the results are sent.
//
// The caller must process the results as they arrive on the channel to avoid
// blocking the worker goroutines.
func Attack[Response any](t *testing.T, rate Rate, duration time.Duration, fn func() Response) <-chan Response {
	wg := sync.WaitGroup{}
	workers := 256

	ticks := make(chan struct{})
	responses := make(chan Response)

	totalRequests := rate.Freq * int(duration/rate.Per)
	dt := rate.Per / time.Duration(rate.Freq)

	wg.Add(totalRequests)

	go func() {
		for i := 0; i < totalRequests; i++ {
			ticks <- struct{}{}
			time.Sleep(dt)
		}
	}()

	for i := 0; i < workers; i++ {
		go func() {
			for range ticks {
				responses <- fn()
				wg.Done()

			}
		}()
	}

	go func() {
		wg.Wait()

		close(ticks)
		pending := len(responses)
		for pending > 0 {
			t.Logf("waiting for responses to be processed: %d", pending)
			time.Sleep(100 * time.Millisecond)
		}
		close(responses)

	}()

	return responses
}
