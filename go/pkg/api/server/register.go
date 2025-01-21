package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/api"
	apierrors "github.com/unkeyed/unkey/go/pkg/api/errors"
	"github.com/unkeyed/unkey/go/pkg/api/routes"
	v1RatelimitLimit "github.com/unkeyed/unkey/go/pkg/api/routes/v1_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/api/session"
)

// here we register all of the routes
// this function runs during startup
func (s *Server) registerRoutes() {

	svc := &api.Services{
		Logger: s.logger,
	}

	register(s, v1RatelimitLimit.New(svc))
}

// register registers a route on the http ServeMux
//
// this can not be receiver method on a struct, because routes are generic.
func register[TRequest session.Redacter, TResponse session.Redacter](srv *Server, route routes.Route[TRequest, TResponse]) {
	srv.mux.HandleFunc(fmt.Sprintf("%s %s", route.Method(), route.Path()), func(w http.ResponseWriter, r *http.Request) {
		log.Printf("REQ: %s %s\n", r.Method, r.URL.Path)

		sess, ok := srv.getSession().(session.Session[TRequest, TResponse])
		if !ok {
			panic("Unable to cast session")
		}
		defer func() {
			sess.Reset()
			srv.returnSession(sess)
		}()

		sess.Init(w, r)

		err := route.Handle(sess)

		if err != nil {
			base := apierrors.BaseError{}
			if errors.As(err, &base) {

				base.RequestID = sess.RequestID()
				b, err := base.Marshal()
				if err != nil {
					panic(fmt.Errorf("unable to marshal error response: %w", err))
				}
				_, err = w.Write(b)
				if err != nil {
					panic(fmt.Errorf("unable to write error response: %w", err))
				}
			}
			return
		}

		err = sess.Flush()
		if err != nil {
			panic(fmt.Errorf("unable to write http response: %w", err))
		}

	})
}
