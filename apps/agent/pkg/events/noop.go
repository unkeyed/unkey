package events

type noopTopic[E any] struct{}

func newNoopTopic[E any]() Topic[E] {
	return &noopTopic[E]{}
}

func (t *noopTopic[E]) Emit(event E) {}

func (t *noopTopic[E]) Subscribe() <-chan E {
	return make(chan E)
}
