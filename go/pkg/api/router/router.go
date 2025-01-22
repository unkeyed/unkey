package router

import (
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/api"
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

type Router interface {
	Register(pattern string, h api.Handler[any, any])
}

type router struct {
	mux http.ServeMux
}

var _ Router = &router{}

func (r *router) Register(pattern string, handle api.Handler[any, any]) {
	r.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {

		sess := session.New[any, any](w, r)

		err := handle(sess)
		if err != nil {
			panic("HANDLE ME")
		}

	})
}
