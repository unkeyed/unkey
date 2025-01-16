package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/api/routes"
	"github.com/unkeyed/unkey/go/pkg/api/session"
	"github.com/unkeyed/unkey/go/pkg/api/validation"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

type Router struct {
	mux              *http.ServeMux
	openapiValidator validation.OpenAPIValidator
}

func New(validator validation.OpenAPIValidator) *Router {
	return &Router{
		mux:              http.NewServeMux(),
		openapiValidator: validator,
	}
}

func (r *Router) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, r.mux)
}

func Register[TRequest any, TResponse any](router *Router, route routes.Route[TRequest, TResponse]) {
	router.mux.HandleFunc(fmt.Sprintf("%s %s", route.Method(), route.Path()), func(w http.ResponseWriter, r *http.Request) {
		log.Printf("REQ: %s %s\n", r.Method, r.URL.Path)
		requestID := uid.Request()

		validationErrors, valid := router.openapiValidator.Validate(r)
		if !valid {

			log.Println(validationErrors)
			validationErrors.RequestId = requestID
			w.WriteHeader(400)
			b, err := json.Marshal(validationErrors)
			if err != nil {
				panic(err)
			}
			_, err = w.Write(b)
			if err != nil {
				panic(err)
			}
			return
		}

		sess := session.New[TRequest, TResponse](requestID, w, r)
		err := route.Handle(sess)

		if err != nil {
			panic("HANDLE ME")
		}

	})
}
