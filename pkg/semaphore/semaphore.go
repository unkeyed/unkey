package semaphore

// Semaphore limits concurrent access to a resource.
type Semaphore struct {
	ch chan struct{}
}

// New creates a semaphore that allows up to n concurrent acquisitions.
// Panics if n < 1.
func New(n int) *Semaphore {
	if n < 1 {
		panic("semaphore: n must be at least 1")
	}
	return &Semaphore{ch: make(chan struct{}, n)}
}

// Do acquires a slot, runs fn in a new goroutine, and releases the slot when fn returns.
// It blocks until a slot is available.
func (s *Semaphore) Do(fn func()) {
	s.ch <- struct{}{}
	go func() {
		defer func() { <-s.ch }()
		fn()
	}()
}
