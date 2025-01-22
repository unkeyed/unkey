package httpApi

type Middleware func(handler Handler) Handler
