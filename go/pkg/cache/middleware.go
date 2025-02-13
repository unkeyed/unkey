package cache

type Middleware[K comparable, V any] func(Cache[K, V]) Cache[K, V]
