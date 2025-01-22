package httpApi

import "github.com/unkeyed/unkey/go/pkg/http_api/validation"

type OpenApiValidator[TRequest Redacter, TResponse Redacter] struct {
	next      Handler[TRequest, TResponse]
	validator validation.Validator
}

var _ Handler[Redacter, Redacter] = (*RequestLogger[Redacter, Redacter])(nil)

func (mw *OpenApiValidator[TRequest, TResponse]) Handle(sess *Session[TRequest, TResponse]) error {

	err, valid := mw.validator.Validate(sess.Request().Raw())
	if !valid {
		panic("HANDLE ME" + err.Type)
	}
	return mw.next.Handle(sess)

}
