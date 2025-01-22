package api

import "github.com/unkeyed/unkey/go/pkg/api/session"

type Handler[TRequest any, TResponse any] func(s session.Session[TRequest, TResponse]) error
