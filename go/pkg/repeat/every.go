package repeat

import "time"

// Every runs the given function in a go routine every d duration until the returned function is called.
func Every(d time.Duration, fn func()) func() {
	t := time.NewTicker(d)
	go func() {
		for range t.C {
			fn()
		}
	}()
	return func() {
		t.Stop()
	}
}
