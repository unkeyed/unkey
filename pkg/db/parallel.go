package db

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// ParallelGroup runs multiple read operations concurrently, each on its own
// connection from the pool. The first non-nil error from any scheduled
// function cancels the shared context, which the MySQL driver honors by
// aborting in-flight sibling queries.
//
// Use this when a handler needs several independent reads before doing work
// that depends on all of them. It is not meant for writes inside a single
// transaction (a tx is pinned to one connection and cannot be parallelized).
type ParallelGroup struct {
	eg  *errgroup.Group
	ctx context.Context
}

// NewParallelGroup constructs a ParallelGroup whose internal context is
// derived from ctx via errgroup.WithContext.
func NewParallelGroup(ctx context.Context) *ParallelGroup {
	eg, egCtx := errgroup.WithContext(ctx)
	return &ParallelGroup{eg: eg, ctx: egCtx}
}

// Wait blocks until every scheduled function returns, then yields the first
// error that occurred (if any). On a non-nil return value, the result
// pointers passed to Go are not guaranteed to be populated.
func (g *ParallelGroup) Wait() error {
	return g.eg.Wait()
}

// Go schedules f to run concurrently. On success, the returned value is
// written to *out. f receives the group's derived context, so a sibling's
// failure cancels f's query in flight.
//
// Each call must pass a distinct out pointer to avoid data races.
//
// Free function rather than a method because Go does not allow type
// parameters on methods.
func Go[T any](g *ParallelGroup, out *T, f func(context.Context) (T, error)) {
	g.eg.Go(func() error {
		v, err := f(g.ctx)
		if err != nil {
			return err
		}
		*out = v
		return nil
	})
}
