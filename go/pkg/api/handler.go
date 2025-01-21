package api

import (
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Handler[TRequest session.Redacter, TResponse session.Redacter] func(s session.Session[TRequest, TResponse]) error
