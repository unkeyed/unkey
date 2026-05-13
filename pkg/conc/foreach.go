package conc

import "context"

// ForEach processes items concurrently with bounded parallelism. It blocks
// until all items have been processed. Each item is passed by pointer to avoid
// copies of large Kubernetes structs.
//
// The semaphore is context-aware: if the context is cancelled, pending items
// are skipped rather than blocking on semaphore acquisition.
//
//	conc.ForEach(ctx, replicaSets.Items, func(ctx context.Context, rs *appsv1.ReplicaSet) {
//	    c.resyncReplicaSet(ctx, rs)
//	})
func ForEach[T any](ctx context.Context, items []T, fn func(ctx context.Context, item *T)) {
	sem := NewSem(DefaultConcurrency)
	for i := range items {
		item := &items[i]
		sem.Go(ctx, func(ctx context.Context) {
			fn(ctx, item)
		})
	}
	sem.Wait()
}
