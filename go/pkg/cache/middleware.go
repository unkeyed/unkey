package cache

type Middleware[T any] func(Cache[T]) Cache[T]
