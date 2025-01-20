package api

import (
	"net/http"
	"net/url"

	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Handler[TRequest any, TResponse any] func(s session.Session[TRequest, TResponse]) error

type Request[TBody Redacter[TBody]] struct {
	Query  url.Values
	Header http.Header
	Body   TBody
}

type Response[TBody Redacter[TBody]] struct {
	Status int
	Header http.Header
	Body   TBody
}
