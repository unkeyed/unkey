package buffer

type noopBuffer[T any] struct {
	c <-chan *T
}

// NewNoop discards items without starting background workers.
// Consume returns a closed channel, including before Close is called.
func NewNoop[T any]() Buffer[T] {
	c := make(chan *T)
	close(c)
	return &noopBuffer[T]{c: c}
}

func (*noopBuffer[T]) Buffer(T) {}

func (b *noopBuffer[T]) Consume() <-chan *T {
	return b.c
}

func (*noopBuffer[T]) Size() int {
	return 0
}

func (*noopBuffer[T]) Close() {}
