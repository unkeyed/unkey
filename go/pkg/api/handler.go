package api

import (
	"net/http"
	"net/url"

	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Request[TBody session.Redacter] struct {
	Query  url.Values
	Header http.Header
	Body   TBody
}

func (r Request[TBody]) Redact() {
	for k := range r.Header {
		if k == "authorization" {
			r.Header.Set(k, "<REDACTED>")
		}
	}
	r.Body.Redact()
}

type Response[TBody session.Redacter] struct {
	Status int
	Header http.Header
	Body   TBody
}

func (r Response[TBody]) Redact() {
	r.Body.Redact()
}

type Handler[TRequest session.Redacter, TResponse session.Redacter] func(s session.Session[TRequest, TResponse]) error
