package connect

import (
	"fmt"
	"net/http"
	"time"
)

type headerMiddleware struct {
	handler http.Handler
}

func newHeaderMiddleware(handler http.Handler) http.Handler {
	return &headerMiddleware{handler}
}

func (h *headerMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.handler.ServeHTTP(w, r)
	serviceLatency := time.Since(start).Milliseconds()
	w.Header().Add("Unkey-Latency", fmt.Sprintf("service=%d", serviceLatency))

}
