package prometheus

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Listen(path string, port int) error {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())
	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
}
