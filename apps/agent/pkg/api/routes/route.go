package routes

import (
	"github.com/gofiber/fiber/v2"
)

type Route struct {
	method  string
	path    string
	handler []fiber.Handler
}

func NewRoute(method string, path string, handler ...fiber.Handler) *Route {
	return &Route{
		method:  method,
		path:    path,
		handler: handler,
	}
}

func (r *Route) WithMiddleware(mw ...fiber.Handler) *Route {
	for _, m := range mw {
		r.handler = append([]fiber.Handler{m}, r.handler...)
	}
	return r
}

func (r *Route) Register(app *fiber.App) {
	app.Add(r.method, r.path, r.handler...)
}

func (r *Route) Method() string {
	return r.method
}

func (r *Route) Path() string {
	return r.path
}
