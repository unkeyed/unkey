// Package conc provides bounded-concurrency helpers.
package conc

import (
	"context"
	"sync"

	"golang.org/x/sync/semaphore"
)

// DefaultConcurrency is the maximum number of items processed in parallel.
const DefaultConcurrency int64 = 10

// Sem is a context-aware bounded-concurrency scheduler. It wraps
// [semaphore.Weighted] with [sync.WaitGroup] so callers only provide the work
// function.
//
//	sem := conc.NewSem(10)
//	for _, item := range items {
//	    sem.Go(ctx, func(ctx context.Context) { process(item) })
//	}
//	sem.Wait()
type Sem struct {
	sem *semaphore.Weighted
	wg  sync.WaitGroup
}

// NewSem creates a [Sem] that allows up to n concurrent goroutines.
func NewSem(n int64) *Sem {
	return &Sem{
		sem: semaphore.NewWeighted(n),
		wg:  sync.WaitGroup{},
	}
}

// Go acquires a semaphore slot, then runs fn in a new goroutine. If the
// context is cancelled before a slot is available, fn is not executed.
func (s *Sem) Go(ctx context.Context, fn func(ctx context.Context)) {
	if err := s.sem.Acquire(ctx, 1); err != nil {
		return
	}
	s.wg.Go(func() {
		defer s.sem.Release(1)
		fn(ctx)
	})
}

// Wait blocks until all goroutines launched via [Sem.Go] have finished.
func (s *Sem) Wait() {
	s.wg.Wait()
}
