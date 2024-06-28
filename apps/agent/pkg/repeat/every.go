package repeat

import "time"

func Every(d time.Duration, fn func()) func() {
	stop := make(chan struct{})
	t := time.NewTicker(d)
	defer t.Stop()
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
