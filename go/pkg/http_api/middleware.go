package httpApi

type Middleware[TRequest Redacter, TResponse Redacter] func(handler Handler[TRequest, TResponse]) Handler[TRequest, TResponse]
