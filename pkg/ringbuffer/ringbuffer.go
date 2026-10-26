// Package ringbuffer provides a fixed-capacity buffer that overwrites its
// oldest item when it is full.
//
// A [Ring] is not safe for concurrent use. The caller must synchronize
// access.
package ringbuffer

// Ring holds the last `capacity` items pushed to it. It allocates as it
// fills, so an empty ring with a large capacity uses no memory for items.
type Ring[T any] struct {
	items    []T
	capacity int
	oldest   int
}

// New returns an empty ring that holds at most `capacity` items. A ring with
// a capacity of 0 holds nothing: every push returns its own item.
func New[T any](capacity int) *Ring[T] {
	return &Ring[T]{
		items:    nil, //faultline:equivalent nil is the zero slice
		capacity: max(0, capacity),
		oldest:   0, //faultline:equivalent 0 is the zero int
	}
}

// Push adds `item`. When the ring is full, Push overwrites the oldest item
// and returns it with `full` set to true.
func (r *Ring[T]) Push(item T) (overwritten T, full bool) {
	if len(r.items) < r.capacity {
		r.items = append(r.items, item)
		return overwritten, false //faultline:equivalent overwritten is the zero named result
	}
	if r.capacity == 0 {
		return item, true
	}
	overwritten = r.items[r.oldest]
	r.items[r.oldest] = item
	r.oldest = (r.oldest + 1) % r.capacity
	return overwritten, true
}

// Clear removes every item and frees their memory.
func (r *Ring[T]) Clear() {
	r.items = nil
	r.oldest = 0
}
