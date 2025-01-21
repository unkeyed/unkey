package middleware

import (
	"github.com/unkeyed/unkey/go/pkg/api/session"
	"github.com/unkeyed/unkey/go/pkg/api/validation"
)

type OpenApiValidator[TRequest session.Redacter, TResponse session.Redacter] struct {
	next      session.Handler[TRequest, TResponse]
	validator validation.Validator
}

var _ session.Handler[session.Redacter, session.Redacter] = (*RequestLogger[session.Redacter, session.Redacter])(nil)

func (mw *OpenApiValidator[TRequest, TResponse]) Handle(sess session.Session[TRequest, TResponse]) error {

	err, valid := mw.validator.Validate(sess.Request().Raw())
	if !valid {
		panic("HANDLE ME")
	}
	return mw.next.Handle(sess)

}
