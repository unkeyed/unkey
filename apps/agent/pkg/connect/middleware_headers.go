package connect

import (
	"net/http"
	"os"
)

type headerMiddleware struct {
	handler http.Handler
}

func newHeaderMiddleware(handler http.Handler) http.Handler {
	return &headerMiddleware{handler}
}

func (h *headerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Unkey-AWS-Region", os.Getenv("REGION"))
	h.handler.ServeHTTP(w, r)

}
