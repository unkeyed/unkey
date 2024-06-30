package repeat

import "time"

// Every runs the given function in a go routine every d duration until the returned function is called.
func Every(d time.Duration, fn func()) func() {
	stop := make(chan struct{})
	t := time.NewTicker(d)
	go func() {
		for {

			select {
			case <-t.C:
				fn()
			case <-stop:
				return
			}
		}
	}()
	return func() {
		stop <- struct{}{}
	}
}
