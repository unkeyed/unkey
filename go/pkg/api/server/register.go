package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/api/routes"
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

func register[TRequest any, TResponse any](mux *http.ServeMux, route routes.Route[TRequest, TResponse]) {
	mux.HandleFunc(fmt.Sprintf("%s %s", route.Method(), route.Path()), func(w http.ResponseWriter, r *http.Request) {
		log.Printf("REQ: %s %s\n", r.Method, r.URL.Path)

		sess := session.New[TRequest, TResponse](w, r)
		err := route.Handle(sess)

		if err != nil {
			panic("HANDLE ME")
		}

	})
}

func (s *server) registerRoutes() {

}
